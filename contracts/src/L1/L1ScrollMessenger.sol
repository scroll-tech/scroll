// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { PausableUpgradeable } from "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";

import { IZKRollup } from "./rollup/IZKRollup.sol";
import { L1MessageQueue } from "./rollup/L1MessageQueue.sol";
import { IL1ScrollMessenger, IScrollMessenger } from "./IL1ScrollMessenger.sol";
import { Version } from "../libraries/common/Version.sol";
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
contract L1ScrollMessenger is Version, OwnableUpgradeable, PausableUpgradeable, ScrollMessengerBase, IL1ScrollMessenger {
  /*************
   * Variables *
   *************/

  /// @notice Mapping from relay id to relay status.
  mapping(bytes32 => bool) public isMessageRelayed;

  /// @notice Mapping from message hash to drop status.
  mapping(bytes32 => bool) public isMessageDropped;

  /// @notice Mapping from message hash to execution status.
  mapping(bytes32 => bool) public isMessageExecuted;

  /// @notice The address of Rollup contract.
  address public rollup;

  /// @notice The address of L1MessageQueue contract.
  L1MessageQueue public messageQueue;

  /**************************************** Constructor ****************************************/

  function initialize(address _rollup) public initializer {
    OwnableUpgradeable.__Ownable_init();
    PausableUpgradeable.__Pausable_init();
    ScrollMessengerBase._initialize();

    messageQueue = new L1MessageQueue(address(this));
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

    uint256 _nonce = messageQueue.nextMessageIndex();
    bytes32 _msghash = keccak256(abi.encodePacked(msg.sender, _to, _value, _fee, _deadline, _nonce, _message));
    messageQueue.appendMessage(_msghash);

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
  ) external virtual override whenNotPaused {
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "Already in execution");

    // solhint-disable-next-line not-rely-on-time
    // @note disable for now since we cannot generate proof in time.
    // require(_deadline >= block.timestamp, "Message expired");

    bytes32 _msghash = keccak256(abi.encodePacked(_from, _to, _value, _fee, _deadline, _nonce, _message));

    require(!isMessageExecuted[_msghash], "Message successfully executed");

    require(IZKRollup(rollup).isBlockFinalized(_proof.blockHash), "Block not finalized");

    bytes32 _messageRoot = IZKRollup(rollup).getL2MessageRoot(_proof.blockHash);
    require(_messageRoot != bytes32(0), "Invalid L2 message root");

    require(
      ZkTrieVerifier.verifyMerkleProof(_messageRoot, _msghash, _nonce, _proof.messageRootProof),
      "Invalid message proof"
    );

    // @todo check `_to` address to avoid attack.

    // @todo take fee and distribute to relayer later.

    // @note This usually will never happen, just in case.
    require(_from != xDomainMessageSender, "Invalid message sender");

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
    address,
    address,
    uint256,
    uint256,
    uint256,
    uint256,
    bytes memory,
    uint256
  ) external override whenNotPaused {
    // @todo
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
