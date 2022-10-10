// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { IL2ScrollMessenger, IScrollMessenger } from "./IL2ScrollMessenger.sol";
import { L2ToL1MessagePasser } from "./predeploys/L2ToL1MessagePasser.sol";
import { OwnableBase } from "../libraries/common/OwnableBase.sol";
import { IGasOracle } from "../libraries/oracle/IGasOracle.sol";
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
contract L2ScrollMessenger is ScrollMessengerBase, OwnableBase, IL2ScrollMessenger {
  /**************************************** Variables ****************************************/

  /// @notice Mapping from relay id to relay status.
  mapping(bytes32 => bool) public isMessageRelayed;

  /// @notice Mapping from message hash to execution status.
  mapping(bytes32 => bool) public isMessageExecuted;

  /// @notice Message nonce, used to avoid relay attack.
  uint256 public messageNonce;

  /// @notice Contract to store the sent message.
  L2ToL1MessagePasser public messagePasser;

  constructor(address _owner) {
    ScrollMessengerBase._initialize();
    owner = _owner;

    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

    messagePasser = new L2ToL1MessagePasser(address(this));
  }

  /**************************************** Mutated Functions ****************************************/

  /// @inheritdoc IScrollMessenger
  function sendMessage(
    address _to,
    uint256 _fee,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable override onlyWhitelistedSender(msg.sender) {
    require(msg.value >= _fee, "cannot pay fee");

    // solhint-disable-next-line not-rely-on-time
    uint256 _deadline = block.timestamp + dropDelayDuration;
    // compute fee by GasOracle contract.
    uint256 _minFee = gasOracle == address(0) ? 0 : IGasOracle(gasOracle).estimateMessageFee(msg.sender, _to, _message);
    require(_fee >= _minFee, "fee too small");

    uint256 _nonce = messageNonce;
    uint256 _value;
    unchecked {
      _value = msg.value - _fee;
    }

    bytes32 _msghash = keccak256(abi.encodePacked(msg.sender, _to, _value, _fee, _deadline, _nonce, _message));

    messagePasser.passMessageToL1(_msghash);

    emit SentMessage(_to, msg.sender, _value, _fee, _deadline, _message, _nonce, _gasLimit);

    unchecked {
      messageNonce = _nonce + 1;
    }
  }

  /// @inheritdoc IL2ScrollMessenger
  function relayMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message
  ) external override {
    // anti reentrance
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "already in execution");

    // @todo only privileged accounts can call

    // solhint-disable-next-line not-rely-on-time
    require(_deadline >= block.timestamp, "Message expired");

    bytes32 _msghash = keccak256(abi.encodePacked(_from, _to, _value, _fee, _deadline, _nonce, _message));

    require(!isMessageExecuted[_msghash], "Message successfully executed");

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
  ) external virtual override {
    revert("not supported");
  }

  /**************************************** Restricted Functions ****************************************/

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
