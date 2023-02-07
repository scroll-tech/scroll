// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { IScrollMessenger } from "../libraries/IScrollMessenger.sol";

interface IL2ScrollMessenger is IScrollMessenger {
  /***********
   * Structs *
   ***********/

  struct L1MessageProof {
    bytes32 blockHash;
    bytes stateRootProof;
  }

  /**************************************** Mutate Functions ****************************************/

    /// @notice Send cross chain message from L1 to L2.
  /// @param target The address of account who recieve the message.
  /// @param value The amount of ether passed when call target contract.
  /// @param message The content of the message.
  /// @param gasLimit Gas limit required to complete the message relay on L2.
  function sendMessage(
    address target,
    uint256 value,
    bytes calldata message,
    uint256 gasLimit
  ) external payable;

  /// @notice execute L1 => L2 message
  /// @dev Make sure this is only called by privileged accounts.
  /// @param _from The address of the sender of the message.
  /// @param _to The address of the recipient of the message.
  /// @param _value The msg.value passed to the message call.
  /// @param _nonce The nonce of the message to avoid replay attack.
  /// @param _message The content of the message.
  function relayMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _nonce,
    bytes memory _message
  ) external;

  /// @notice execute L1 => L2 message with proof
  /// @param _from The address of the sender of the message.
  /// @param _to The address of the recipient of the message.
  /// @param _value The msg.value passed to the message call.
  /// @param _nonce The nonce of the message to avoid replay attack.
  /// @param _message The content of the message.
  /// @param _proof The message proof.
  function relayMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _nonce,
    bytes memory _message,
    L1MessageProof calldata _proof
  ) external;
}
