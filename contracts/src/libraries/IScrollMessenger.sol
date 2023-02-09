// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IScrollMessenger {
  /**********
   * Events *
   **********/

  /// @notice Emitted when a cross domain message is sent.
  /// @param sender The address of the sender who initiates the message.
  /// @param target The address of target contract to call.
  /// @param value The amount of value passed to the target contract.
  /// @param message The calldata passed to the target contract.
  /// @param messageNonce The nonce of the message.
  event SentMessage(address indexed sender, address indexed target, uint256 value, bytes message, uint256 messageNonce);

  /// @notice Emitted when a cross domain message is relayed successfully.
  /// @param messageHash The hash of the message.
  event RelayedMessage(bytes32 indexed messageHash);

  /// @notice Emitted when a cross domain message is failed to relay.
  /// @param messageHash The hash of the message.
  event FailedRelayedMessage(bytes32 indexed messageHash);

  /**************************************** View Functions ****************************************/

  /// @notice Return the sender of a cross domain message.
  function xDomainMessageSender() external view returns (address);
}
