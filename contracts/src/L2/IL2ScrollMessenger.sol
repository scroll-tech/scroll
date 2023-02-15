// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { IScrollMessenger } from "../libraries/IScrollMessenger.sol";

interface IL2ScrollMessenger is IScrollMessenger {
  /**********
   * Events *
   **********/

  /// @notice Emitted when a cross domain message is sent.
  /// @param sender The address of the sender who initiates the message.
  /// @param target The address of target contract to call.
  /// @param value The amount of value passed to the target contract.
  /// @param messageNonce The nonce of the message.
  /// @param gasLimit The optional gas limit passed to L1.
  /// @param message The calldata passed to the target contract.
  event SentMessage(
    address indexed sender,
    address indexed target,
    uint256 value,
    uint256 messageNonce,
    uint256 gasLimit,
    bytes message
  );

  /***********
   * Structs *
   ***********/

  struct L1MessageProof {
    bytes32 blockHash;
    bytes stateRootProof;
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Send cross chain message from L2 to L1.
  /// @param target The address of account who recieve the message.
  /// @param value The amount of ether passed when call target contract.
  /// @param message The content of the message.
  /// @param gasLimit Gas limit required to complete the message relay on L1.
  function sendMessage(
    address target,
    uint256 value,
    bytes calldata message,
    uint256 gasLimit
  ) external payable;

  /// @notice execute L1 => L2 message
  /// @dev Make sure this is only called by privileged accounts.
  /// @param from The address of the sender of the message.
  /// @param to The address of the recipient of the message.
  /// @param value The msg.value passed to the message call.
  /// @param nonce The nonce of the message to avoid replay attack.
  /// @param message The content of the message.
  function relayMessage(
    address from,
    address to,
    uint256 value,
    uint256 nonce,
    bytes calldata message
  ) external;

  /// @notice execute L1 => L2 message with proof
  /// @param from The address of the sender of the message.
  /// @param to The address of the recipient of the message.
  /// @param value The msg.value passed to the message call.
  /// @param nonce The nonce of the message to avoid replay attack.
  /// @param message The content of the message.
  /// @param proof The message proof.
  function retryMessageWithProof(
    address from,
    address to,
    uint256 value,
    uint256 nonce,
    bytes calldata message,
    L1MessageProof calldata proof
  ) external;
}
