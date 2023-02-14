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

  uint256 private constant MIN_GAS_LIMIT = 21000;

  /// @notice The address of L2MessageQueue.
  address public immutable messageQueue;

  /// @notice The contract contains the list of L1 blocks.
  address public immutable blockContainer;

  /*************
   * Variables *
   *************/

  /// @notice Mapping from L2 message hash to sent status.
  mapping(bytes32 => bool) public isL2MessageSent;

  /// @notice Mapping from L1 message hash to a boolean value indicating if the message has been successfully executed.
  mapping(bytes32 => bool) public isL1MessageExecuted;

  /// @notice Mapping from L1 message hash to the number of failed times.
  mapping(bytes32 => uint256) public l1MessageFailedTimes;

  /// @notice The maximum number of times each L1 message can fail in L2.
  uint256 public maxFailedExecutionTimes;

  /***************
   * Constructor *
   ***************/

  constructor(address _blockContainer, address _messageQueue) {
    blockContainer = _blockContainer;
    messageQueue = _messageQueue;
  }

  function initialize(address _counterpart, address _feeVault) external initializer {
    PausableUpgradeable.__Pausable_init();
    ScrollMessengerBase._initialize(_counterpart, _feeVault);

    maxFailedExecutionTimes = 3;

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
    // `mapping(bytes32 => bool) public isL1MessageSent` is the 105-nd slot of contract `L1ScrollMessenger`.
    assembly {
      mstore(0x00, _msgHash)
      mstore(0x20, 105)
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
    // `mapping(bytes32 => bool) public isL2MessageExecuted` is the 106-th slot of contract `L1ScrollMessenger`.
    assembly {
      mstore(0x00, _msgHash)
      mstore(0x20, 106)
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
    uint256 _gasLimit
  ) external payable override whenNotPaused {
    // by pass fee vault relay
    if (feeVault != msg.sender) {
      require(_gasLimit >= MIN_GAS_LIMIT, "gas limit too small");
    }

    // compute and deduct the messaging fee to fee vault.
    uint256 _fee = _gasLimit * IL1BlockContainer(blockContainer).latestBaseFee();
    require(msg.value >= _value + _fee, "insufficient msg.value");
    if (_fee > 0) {
      (bool _success, ) = feeVault.call{ value: msg.value - _value }("");
      require(_success, "failed to deduct fee");
    }

    uint256 _nonce = L2MessageQueue(messageQueue).nextMessageIndex();
    bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(msg.sender, _to, _value, _message, _nonce));

    require(!isL2MessageSent[_xDomainCalldataHash], "duplicated message");
    isL2MessageSent[_xDomainCalldataHash] = true;

    L2MessageQueue(messageQueue).appendMessage(_xDomainCalldataHash);

    emit SentMessage(msg.sender, _to, _value, _message, _nonce, _gasLimit);
  }

  /// @inheritdoc IL2ScrollMessenger
  function relayMessage(
    address _from,
    address _to,
    uint256 _value,
    bytes memory _message,
    uint256 _nonce
  ) external override whenNotPaused onlyWhitelistedSender(msg.sender) {
    // anti reentrance
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "Already in execution");

    // @todo address unalis to check sender is L1ScrollMessenger

    bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(_from, _to, _value, _message, _nonce));

    require(!isL1MessageExecuted[_xDomainCalldataHash], "Message successfully executed");

    _executeMessage(_from, _to, _value, _message, _xDomainCalldataHash);
  }

  /// @inheritdoc IL2ScrollMessenger
  function retryMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    bytes memory _message,
    uint256 _nonce,
    L1MessageProof calldata _proof
  ) external override whenNotPaused {
    // anti reentrance
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "Already in execution");

    // check message status
    bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(_from, _to, _value, _message, _nonce));
    require(!isL1MessageExecuted[_xDomainCalldataHash], "Message successfully executed");
    require(l1MessageFailedTimes[_xDomainCalldataHash] > 0, "Message not relayed before");

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
      isL1MessageExecuted[_xDomainCalldataHash] = true;
      emit RelayedMessage(_xDomainCalldataHash);
    } else {
      unchecked {
        uint256 _failedTimes = l1MessageFailedTimes[_xDomainCalldataHash] + 1;
        require(_failedTimes <= maxFailedExecutionTimes, "Exceed maximum failure");
        l1MessageFailedTimes[_xDomainCalldataHash] = _failedTimes;
      }
      emit FailedRelayedMessage(_xDomainCalldataHash);
    }
  }
}
