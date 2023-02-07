// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { PausableUpgradeable } from "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";

import { IL2ScrollMessenger, IScrollMessenger } from "./IL2ScrollMessenger.sol";
import { L2MessageQueue } from "./predeploys/L2MessageQueue.sol";
import { IL1BlockContainer } from "./predeploys/IL1BlockContainer.sol";
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
contract L2ScrollMessenger is ScrollMessengerBase, PausableUpgradeable, IL2ScrollMessenger {
  address public immutable messageQueue;

  /// @notice The contract contains the list of L1 blocks.
  address public immutable blockContainer;

  /**************************************** Variables ****************************************/

  /// @notice Mapping from relay id to relay status.
  mapping(bytes32 => bool) public isMessageRelayed;

  /// @notice Mapping from message hash to execution status.
  mapping(bytes32 => bool) public isMessageExecuted;

  constructor(address _blockContainer, address _messageQueue) {
    blockContainer = _blockContainer;
    messageQueue = _messageQueue;
  }

  function initialize() external initializer {
    PausableUpgradeable.__Pausable_init();
    ScrollMessengerBase._initialize();

    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;
  }

  /**************************************** Mutated Functions ****************************************/

  /// @inheritdoc IL2ScrollMessenger
  function sendMessage(
    address _to,
    uint256 _value,
    bytes memory _message,
    uint256
  ) external payable override whenNotPaused {
    require(msg.value >= _value, "value not enough");

    // @todo deduct fee

    uint256 _nonce = L2MessageQueue(messageQueue).nextMessageIndex();
    bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(msg.sender, _to, _message, _nonce));

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

    bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(_from, _to, _message, _nonce));

    require(!isMessageExecuted[_xDomainCalldataHash], "Message successfully executed");

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
      isMessageExecuted[_xDomainCalldataHash] = true;
      emit RelayedMessage(_xDomainCalldataHash);
    } else {
      emit FailedRelayedMessage(_xDomainCalldataHash);
    }

    bytes32 _relayId = keccak256(abi.encodePacked(_xDomainCalldataHash, msg.sender, block.number));

    isMessageRelayed[_relayId] = true;
  }

  /// @inheritdoc IL2ScrollMessenger
  function relayMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _nonce,
    bytes memory _message,
    L1MessageProof calldata _proof
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
