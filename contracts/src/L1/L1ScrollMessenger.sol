// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { PausableUpgradeable } from "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";

import { IZKRollup } from "./rollup/IZKRollup.sol";
import { IL1ScrollMessenger, IScrollMessenger } from "./IL1ScrollMessenger.sol";
import { IGasOracle } from "../libraries/oracle/IGasOracle.sol";
import { ScrollConstants } from "../libraries/ScrollConstants.sol";
import { ScrollMessengerBase } from "../libraries/ScrollMessengerBase.sol";
import { ZkTrieVerifier } from "../libraries/verifier/ZkTrieVerifier.sol";

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
contract L1ScrollMessenger is OwnableUpgradeable, PausableUpgradeable, ScrollMessengerBase, IL1ScrollMessenger {
  /**************************************** Variables ****************************************/

  /// @notice Mapping from relay id to relay status.
  mapping(bytes32 => bool) public isMessageRelayed;

  /// @notice Mapping from message hash to drop status.
  mapping(bytes32 => bool) public isMessageDropped;

  /// @notice Mapping from message hash to execution status.
  mapping(bytes32 => bool) public isMessageExecuted;

  /// @notice The address of Rollup contract.
  address public rollup;

  /**************************************** Constructor ****************************************/

  function initialize(address _rollup) public initializer {
    OwnableUpgradeable.__Ownable_init();
    PausableUpgradeable.__Pausable_init();
    ScrollMessengerBase._initialize();

    rollup = _rollup;
    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;
  }

  /**************************************** Mutated Functions ****************************************/

  /// @inheritdoc IScrollMessenger
  function sendMessage(
    address _to,
    uint256 _fee,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable override whenNotPaused onlyWhitelistedSender(msg.sender) {
    require(msg.value >= _fee, "cannot pay fee");

    // solhint-disable-next-line not-rely-on-time
    uint256 _deadline = block.timestamp + dropDelayDuration;
    // compute minimum fee required by GasOracle contract.
    uint256 _minFee = gasOracle == address(0) ? 0 : IGasOracle(gasOracle).estimateMessageFee(msg.sender, _to, _message);
    require(_fee >= _minFee, "fee too small");
    uint256 _value;
    unchecked {
      _value = msg.value - _fee;
    }

    uint256 _nonce = IZKRollup(rollup).appendMessage(msg.sender, _to, _value, _fee, _deadline, _message, _gasLimit);

    emit SentMessage(_to, msg.sender, _value, _fee, _deadline, _message, _nonce, _gasLimit);
  }

  /// @inheritdoc IL1ScrollMessenger
  function relayMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message,
    L2MessageProof memory _proof
  ) external override whenNotPaused onlyWhitelistedSender(msg.sender) {
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "already in execution");

    // solhint-disable-next-line not-rely-on-time
    // @note disable for now since we cannot generate proof in time.
    // require(_deadline >= block.timestamp, "Message expired");

    bytes32 _msghash = keccak256(abi.encodePacked(_from, _to, _value, _fee, _deadline, _nonce, _message));

    require(!isMessageExecuted[_msghash], "Message successfully executed");

    // @todo check proof
    require(IZKRollup(rollup).verifyMessageStateProof(_proof.batchIndex, _proof.blockHeight), "invalid state proof");
    require(ZkTrieVerifier.verifyMerkleProof(_proof.merkleProof), "invalid proof");

    // @todo check `_to` address to avoid attack.

    // @todo take fee and distribute to relayer later.

    // @note This usually will never happen, just in case.
    require(_from != xDomainMessageSender, "invalid message sender");

    xDomainMessageSender = _from;
    // solhint-disable-next-line avoid-low-level-calls
    (bool success, ) = _to.call{ value: _value }(_message);
    // reset value to refund gas.
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

    if (success) {
      isMessageExecuted[_msghash] = true;
      emit RelayedMessage(_msghash);
    } else {
      emit FailedRelayedMessage(_msghash);
    }

    bytes32 _relayId = keccak256(abi.encodePacked(_msghash, msg.sender, block.number));

    isMessageRelayed[_relayId] = true;
  }

  /// @inheritdoc IL1ScrollMessenger
  function replayMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    bytes memory _message,
    uint256 _queueIndex,
    uint32 _oldGasLimit,
    uint32 _newGasLimit
  ) external override whenNotPaused {
    // @todo
  }

  /// @inheritdoc IScrollMessenger
  function dropMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message,
    uint256 _gasLimit
  ) external override whenNotPaused {
    // solhint-disable-next-line not-rely-on-time
    require(block.timestamp > _deadline, "message not expired");

    // @todo The `queueIndex` is acutally updated asynchronously, it's not a good practice to compare directly.
    address _rollup = rollup; // gas saving
    uint256 _queueIndex = IZKRollup(_rollup).getNextQueueIndex();
    require(_queueIndex <= _nonce, "message already executed");

    bytes32 _expectedMessageHash = IZKRollup(_rollup).getMessageHashByIndex(_nonce);
    bytes32 _messageHash = keccak256(
      abi.encodePacked(_from, _to, _value, _fee, _deadline, _nonce, _message, _gasLimit)
    );
    require(_messageHash == _expectedMessageHash, "message hash mismatched");

    require(!isMessageDropped[_messageHash], "message already dropped");
    isMessageDropped[_messageHash] = true;

    if (_from.code.length > 0) {
      // @todo call finalizeDropMessage of `_from`
    } else {
      // just do simple ether refund
      payable(_from).transfer(_value + _fee);
    }

    emit MessageDropped(_messageHash);
  }

  /**************************************** Restricted Functions ****************************************/

  /// @notice Pause the contract
  /// @dev This function can only called by contract owner.
  function pause() external onlyOwner {
    _pause();
  }

  /// @notice Update whitelist contract.
  /// @dev This function can only called by contract owner.
  /// @param _newWhitelist The address of new whitelist contract.
  function updateWhitelist(address _newWhitelist) external onlyOwner {
    address _oldWhitelist = whitelist;

    whitelist = _newWhitelist;
    emit UpdateWhitelist(_oldWhitelist, _newWhitelist);
  }

  /// @notice Update the address of gas oracle.
  /// @dev This function can only called by contract owner.
  /// @param _newGasOracle The address to update.
  function updateGasOracle(address _newGasOracle) external onlyOwner {
    address _oldGasOracle = gasOracle;
    gasOracle = _newGasOracle;

    emit UpdateGasOracle(_oldGasOracle, _newGasOracle);
  }

  /// @notice Update the drop delay duration.
  /// @dev This function can only called by contract owner.
  /// @param _newDuration The new delay duration to update.
  function updateDropDelayDuration(uint256 _newDuration) external onlyOwner {
    uint256 _oldDuration = dropDelayDuration;
    dropDelayDuration = _newDuration;

    emit UpdateDropDelayDuration(_oldDuration, _newDuration);
  }
}
