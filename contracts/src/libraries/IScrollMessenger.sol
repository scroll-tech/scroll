// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IScrollMessenger {
  /**********
   * Events *
   **********/

  /// @notice Emitted when a cross domain message is sent
  /// @param sender The address of the sender who initiates the message.
  /// @param target The address of target contract to call.
  /// @param value The amount of value passed to the target contract.
  /// @param message The calldata passed to the target contract.
  /// @param messageNonce The nonce of the message.
  event SentMessage(address indexed sender, address indexed target, uint256 value, bytes message, uint256 messageNonce);

  event MessageDropped(bytes32 indexed msgHash);
  event RelayedMessage(bytes32 indexed msgHash);
  event FailedRelayedMessage(bytes32 indexed msgHash);

  /**************************************** View Functions ****************************************/

  function xDomainMessageSender() external view returns (address);
}
