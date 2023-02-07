// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { PausableUpgradeable } from "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";

import { IZKRollup } from "./rollup/IZKRollup.sol";
import { IL1MessageQueue } from "./rollup/IL1MessageQueue.sol";
import { IL1ScrollMessenger, IScrollMessenger } from "./IL1ScrollMessenger.sol";
import { ScrollConstants } from "../libraries/ScrollConstants.sol";
import { ScrollMessengerBase } from "../libraries/ScrollMessengerBase.sol";
import { ZkTrieVerifier } from "../libraries/verifier/ZkTrieVerifier.sol";

// solhint-disable avoid-low-level-calls

/// @title L1ScrollMessenger
/// @notice The `L1ScrollMessenger` contract can:
///
/// 1. send messages from layer 1 to layer 2;
/// 2. relay messages from layer 2 layer 1;
/// 3. replay failed message by replacing the gas limit;
/// 4. drop expired message due to sequencer problems.
///
/// @dev All deposited Ether (including `WETH` deposited throng `L1WETHGateway`) will locked in
/// this contract.
contract L1ScrollMessenger is PausableUpgradeable, ScrollMessengerBase, IL1ScrollMessenger {
  /*************
   * Variables *
   *************/

  /// @notice Mapping from relay id to relay status.
  mapping(bytes32 => bool) public isMessageRelayed;

  /// @notice Mapping from message hash to sent status.
  mapping(bytes32 => bool) public isMessageSent;

  /// @notice Mapping from message hash to execution status.
  mapping(bytes32 => bool) public isMessageExecuted;

  /// @notice The address of L2ScrollMessenger contract in L2.
  address public l2ScrollMessenger;

  /// @notice The address of fee vault, collecting cross domain messaging fee.
  address public feeVault;

  /// @notice The address of Rollup contract.
  address public rollup;

  /// @notice The address of L1MessageQueue contract.
  address public messageQueue;

  /***************
   * Constructor *
   ***************/

  /// @notice Initialize the storage of L1ScrollMessenger.
  /// @param _l2ScrollMessenger The address of L2ScrollMessenger contract in L2.
  /// @param _feeVault The address of fee vault, which will be used to collect relayer fee.
  /// @param _rollup The address of ZKRollup contract.
  /// @param _messageQueue The address of L1MessageQueue contract.
  function initialize(
    address _l2ScrollMessenger,
    address _feeVault,
    address _rollup,
    address _messageQueue
  ) public initializer {
    PausableUpgradeable.__Pausable_init();
    ScrollMessengerBase._initialize();

    l2ScrollMessenger = _l2ScrollMessenger;
    feeVault = _feeVault;
    rollup = _rollup;
    messageQueue = _messageQueue;

    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @inheritdoc IL1ScrollMessenger
  function sendMessage(
    address _to,
    uint256 _value,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable override whenNotPaused {
    address _messageQueue = messageQueue; // gas saving
    address _l2ScrollMessenger = l2ScrollMessenger; // gas saving

    // compute the actual cross domain message calldata.
    uint256 _messageNonce = IL1MessageQueue(_messageQueue).nextCrossDomainMessageIndex();
    bytes memory _xDomainCalldata = _encodeXDomainCalldata(msg.sender, _to, _message, _messageNonce);

    // compute and deduct the messaging fee to fee vault.
    uint256 _fee = IL1MessageQueue(_messageQueue).estimateCrossDomainMessageFee(
      address(this),
      _l2ScrollMessenger,
      _xDomainCalldata,
      _gasLimit
    );
    unchecked {
      require(msg.value >= _fee + _value, "insufficient msg.value");
    }
    if (_fee > 0) {
      (bool _success, ) = feeVault.call{ value: _value }("");
      require(_success, "failed to deduct fee");
    }

    // append message to L2MessageQueue
    IL1MessageQueue(_messageQueue).appendCrossDomainMessage(_l2ScrollMessenger, _gasLimit, _xDomainCalldata);

    // record the message hash for future use.
    bytes32 _xDomainCalldataHash = keccak256(_xDomainCalldata);
    isMessageSent[_xDomainCalldataHash] = true;

    emit SentMessage(msg.sender, _to, _value, _message, _messageNonce);
  }

  /// @inheritdoc IL1ScrollMessenger
  function relayMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _nonce,
    bytes memory _message,
    L2MessageProof memory _proof
  ) external override whenNotPaused onlyWhitelistedSender(msg.sender) {
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "already in execution");

    bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(_from, _to, _message, _nonce));
    require(!isMessageExecuted[_xDomainCalldataHash], "Message successfully executed");

    require(IZKRollup(rollup).isBatchFinalized(_proof.batchIndex), "invalid state proof");
    require(ZkTrieVerifier.verifyMerkleProof(_proof.merkleProof), "invalid proof");

    // @note This usually will never happen, just in case.
    require(_from != xDomainMessageSender, "invalid message sender");

    xDomainMessageSender = _from;
    (bool success, ) = _to.call{ value: _value }(_message);
    // reset value to refund gas.
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

    if (success) {
      isMessageExecuted[_xDomainCalldataHash] = true;
      emit RelayedMessage(_xDomainCalldataHash);
    } else {
      emit FailedRelayedMessage(_xDomainCalldataHash);
    }

    bytes32 _relayId = keccak256(abi.encodePacked(_xDomainCalldataHash, msg.sender, block.number));
    isMessageRelayed[_relayId] = true;
  }

  /// @inheritdoc IL1ScrollMessenger
  function replayMessage(
    address _from,
    address _to,
    uint256 _value,
    bytes memory _message,
    uint256 _queueIndex,
    uint32 _oldGasLimit,
    uint32 _newGasLimit
  ) external override whenNotPaused {
    // @todo
  }

  /************************
   * Restricted Functions *
   ************************/

  /// @notice Pause the contract
  /// @dev This function can only called by contract owner.
  function pause() external onlyOwner {
    _pause();
  }
}
