// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { Version } from "../../libraries/common/Version.sol";
import { IL1MessageQueue } from "./IL1MessageQueue.sol";

/// @title L1MessageQueue
/// @notice This contract will hold all L1 to L2 messages.
/// Each appended message is assigned with a unique and increasing `uint256` index denoting the message nonce.
contract L1MessageQueue is Version, IL1MessageQueue {
  /*************
   * Constants *
   *************/

  /// @notice The address of L1ScrollMessenger contract.
  address public immutable messenger;

  /*************
   * Variables *
   *************/

  /// @inheritdoc IL1MessageQueue
  uint256 public override nextMessageIndex;

  /// @notice Mapping from message hash to message existence.
  mapping(bytes32 => bool) private isMessageSent;

  /***************
   * Constructor *
   ***************/

  constructor(address _messenger) {
    messenger = _messenger;
  }

  /*************************
   * Public View Functions *
   *************************/

  /// @inheritdoc IL1MessageQueue
  function hasMessage(bytes32 _msgHash) external view returns (bool) {
    return isMessageSent[_msgHash];
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @inheritdoc IL1MessageQueue
  function appendMessage(bytes32 _msgHash) external {
    require(msg.sender == messenger, "Only callable by the L1ScrollMessenger");

    require(!isMessageSent[_msgHash], "Message is already appended.");
    isMessageSent[_msgHash] = true;
    emit AppendMessage(_msgHash);

    unchecked {
      nextMessageIndex += 1;
    }
  }
}
