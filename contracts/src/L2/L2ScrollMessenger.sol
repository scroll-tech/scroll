// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { PausableUpgradeable } from "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";

import { IL2ScrollMessenger } from "./IL2ScrollMessenger.sol";
import { L2MessageQueue } from "./predeploys/L2MessageQueue.sol";
import { IL1BlockContainer } from "./predeploys/IL1BlockContainer.sol";

import { PatriciaMerkleTrieVerifier } from "../libraries/verifier/PatriciaMerkleTrieVerifier.sol";
import { ScrollConstants } from "../libraries/ScrollConstants.sol";
import { ScrollMessengerBase } from "../libraries/ScrollMessengerBase.sol";

/// @title L2ScrollMessenger
/// @notice The `L2ScrollMessenger` contract can:
///
/// 1. send messages from layer 2 to layer 1;
/// 2. relay messages from layer 1 layer 2;
/// 3. drop expired message due to sequencer problems.
///
/// @dev It should be a predeployed contract in layer 2 and should hold infinite amount
/// of Ether (Specifically, `uint256(-1)`), which can be initialized in Genesis Block.
contract L2ScrollMessenger is ScrollMessengerBase, PausableUpgradeable, IL2ScrollMessenger {
  /**********
   * Events *
   **********/

  /// @notice Emitted when the maximum number of times each message can fail in L2 is updated.
  /// @param maxFailedExecutionTimes The new maximum number of times each message can fail in L2.
  event UpdateMaxFailedExecutionTimes(uint256 maxFailedExecutionTimes);

  /*************
   * Constants *
   *************/

  /// @notice The address of L2MessageQueue.
  address public immutable messageQueue;

  /// @notice The contract contains the list of L1 blocks.
  address public immutable blockContainer;

  /*************
   * Variables *
   *************/

  /// @notice Mapping from relay id to relay status.
  mapping(bytes32 => bool) public isMessageRelayed;

  /// @notice Mapping from message hash to sent status.
  mapping(bytes32 => bool) public isMessageSent;

  /// @notice Mapping from message hash to execution status.
  mapping(bytes32 => bool) public isMessageExecuted;

  /// @notice Mapping from message hash to the number of failed times.
  mapping(bytes32 => uint256) public messageFailedTimes;

  /// @notice The maximum number of times each message can fail in L2.
  uint256 public maxFailedExecutionTimes;

  /// @notice The address of L1ScrollMessenger contract in L1.
  address public counterpart;

  /// @notice The address of fee vault, collecting cross domain messaging fee.
  address public feeVault;

  /***************
   * Constructor *
   ***************/

  constructor(address _blockContainer, address _messageQueue) {
    blockContainer = _blockContainer;
    messageQueue = _messageQueue;
  }

  function initialize(address _counterpart, address _feeVault) external initializer {
    PausableUpgradeable.__Pausable_init();
    ScrollMessengerBase._initialize();

    counterpart = _counterpart;
    feeVault = _feeVault;

    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;
  }

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Check whether the l1 message is included in the corresponding L1 block.
  /// @param _blockHash The block hash where the message should in.
  /// @param _msgHash The hash of the message to check.
  /// @param _proof The encoded storage proof from eth_getProof.
  /// @return bool Return true is the message is included in L1, otherwise return false.
  function verifyMessageInclusionStatus(
    bytes32 _blockHash,
    bytes32 _msgHash,
    bytes calldata _proof
  ) public view returns (bool) {
    bytes32 _expectedStateRoot = IL1BlockContainer(blockContainer).getStateRoot(_blockHash);
    require(_expectedStateRoot != bytes32(0), "Block not imported");

    // @todo fix the actual slot later.
    bytes32 _storageKey;
    // `mapping(bytes32 => bool) public isMessageSent` is the 104-nd slot of contract `L1ScrollMessenger`.
    assembly {
      mstore(0x00, _msgHash)
      mstore(0x20, 103)
      _storageKey := keccak256(0x00, 0x40)
    }

    (bytes32 _computedStateRoot, bytes32 _storageValue) = PatriciaMerkleTrieVerifier.verifyPatriciaProof(
      counterpart,
      _storageKey,
      _proof
    );
    require(_computedStateRoot == _expectedStateRoot, "State root mismatch");

    return uint256(_storageValue) == 1;
  }

  /// @notice Check whether the message is executed in the corresponding L1 block.
  /// @param _blockHash The block hash where the message should in.
  /// @param _msgHash The hash of the message to check.
  /// @param _proof The encoded storage proof from eth_getProof.
  /// @return bool Return true is the message is executed in L1, otherwise return false.
  function verifyMessageExecutionStatus(
    bytes32 _blockHash,
    bytes32 _msgHash,
    bytes calldata _proof
  ) external view returns (bool) {
    bytes32 _expectedStateRoot = IL1BlockContainer(blockContainer).getStateRoot(_blockHash);
    require(_expectedStateRoot != bytes32(0), "Block not imported");

    // @todo fix the actual slot later.
    bytes32 _storageKey;
    // `mapping(bytes32 => bool) public isMessageExecuted` is the 105-th slot of contract `L1ScrollMessenger`.
    assembly {
      mstore(0x00, _msgHash)
      mstore(0x20, 104)
      _storageKey := keccak256(0x00, 0x40)
    }

    (bytes32 _computedStateRoot, bytes32 _storageValue) = PatriciaMerkleTrieVerifier.verifyPatriciaProof(
      counterpart,
      _storageKey,
      _proof
    );
    require(_computedStateRoot == _expectedStateRoot, "State root mismatch");

    return uint256(_storageValue) == 1;
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @inheritdoc IL2ScrollMessenger
  function sendMessage(
    address _to,
    uint256 _value,
    bytes memory _message,
    uint256
  ) external payable override whenNotPaused {
    require(msg.value >= _value, "value not enough");

    // @todo it's better to charge a minimum fee or relay the fee to L1 to relayer.
    if (msg.value > _value) {
      (bool _success, ) = feeVault.call{ value: msg.value - _value }("");
      require(_success, "failed to deduct fee");
    }

    uint256 _nonce = L2MessageQueue(messageQueue).nextMessageIndex();
    bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(msg.sender, _to, _value, _nonce, _message));

    require(!isMessageSent[_xDomainCalldataHash], "duplicated message");
    isMessageSent[_xDomainCalldataHash] = true;

    L2MessageQueue(messageQueue).appendMessage(_xDomainCalldataHash);

    emit SentMessage(msg.sender, _to, _value, _message, _nonce);
  }

  /// @inheritdoc IL2ScrollMessenger
  function relayMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _nonce,
    bytes memory _message
  ) external override whenNotPaused onlyWhitelistedSender(msg.sender) {
    // anti reentrance
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "Already in execution");

    // @todo address unalis to check sender is L1ScrollMessenger

    // solhint-disable-next-line not-rely-on-time
    // @note disable for now since we may encounter various situation in testnet.
    // require(_deadline >= block.timestamp, "Message expired");

    bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(_from, _to, _value, _nonce, _message));

    require(!isMessageExecuted[_xDomainCalldataHash], "Message successfully executed");

    _executeMessage(_from, _to, _value, _message, _xDomainCalldataHash);

    bytes32 _relayId = keccak256(abi.encodePacked(_xDomainCalldataHash, msg.sender, block.number));

    isMessageRelayed[_relayId] = true;
  }

  /// @inheritdoc IL2ScrollMessenger
  function retryMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _nonce,
    bytes memory _message,
    L1MessageProof calldata _proof
  ) external override whenNotPaused {
    // anti reentrance
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "Already in execution");

    // check message status
    bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(_from, _to, _value, _nonce, _message));
    require(!isMessageExecuted[_xDomainCalldataHash], "Message successfully executed");
    require(messageFailedTimes[_xDomainCalldataHash] > 0, "Message not relayed before");

    require(
      verifyMessageInclusionStatus(_proof.blockHash, _xDomainCalldataHash, _proof.stateRootProof),
      "Message not included"
    );

    _executeMessage(_from, _to, _value, _message, _xDomainCalldataHash);
  }

  /************************
   * Restricted Functions *
   ************************/

  /// @notice Pause the contract
  /// @dev This function can only called by contract owner.
  function pause() external onlyOwner {
    _pause();
  }

  function updateMaxFailedExecutionTimes(uint256 _maxFailedExecutionTimes) external onlyOwner {
    maxFailedExecutionTimes = _maxFailedExecutionTimes;

    emit UpdateMaxFailedExecutionTimes(_maxFailedExecutionTimes);
  }

  /**********************
   * Internal Functions *
   **********************/

  function _executeMessage(
    address _from,
    address _to,
    uint256 _value,
    bytes memory _message,
    bytes32 _xDomainCalldataHash
  ) internal {
    // @todo check `_to` address to avoid attack.

    // @note This usually will never happen, just in case.
    require(_from != xDomainMessageSender, "Invalid message sender");

    xDomainMessageSender = _from;
    // solhint-disable-next-line avoid-low-level-calls
    (bool success, ) = _to.call{ value: _value }(_message);
    // reset value to refund gas.
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

    if (success) {
      isMessageExecuted[_xDomainCalldataHash] = true;
      emit RelayedMessage(_xDomainCalldataHash);
    } else {
      unchecked {
        uint256 _failedTimes = messageFailedTimes[_xDomainCalldataHash] + 1;
        require(_failedTimes <= maxFailedExecutionTimes, "Exceed maximum failure");
        messageFailedTimes[_xDomainCalldataHash] = _failedTimes;
      }
      emit FailedRelayedMessage(_xDomainCalldataHash);
    }
  }
}
