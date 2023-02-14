// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { IScrollMessenger } from "../libraries/IScrollMessenger.sol";

interface IL1ScrollMessenger is IScrollMessenger {
  /***********
   * Structs *
   ***********/

  struct L2MessageProof {
    // The hash of batch where the message belongs to.
    bytes32 batchHash;
    // Concatenation of merkle proof for withdraw merkle trie.
    bytes merkleProof;
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

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

  /// @notice Relay a L2 => L1 message with message proof.
  /// @param from The address of the sender of the message.
  /// @param to The address of the recipient of the message.
  /// @param value The msg.value passed to the message call.
  /// @param message The content of the message.
  /// @param nonce The nonce of the message to avoid replay attack.
  /// @param proof The proof used to verify the correctness of the transaction.
  function relayMessageWithProof(
    address from,
    address to,
    uint256 value,
    bytes memory message,
    uint256 nonce,
    L2MessageProof memory proof
  ) external;

  /// @notice Replay an exsisting message.
  /// @param from The address of the sender of the message.
  /// @param to The address of the recipient of the message.
  /// @param value The msg.value passed to the message call.
  /// @param message The content of the message.
  /// @param queueIndex The queue index for the message to replay.
  /// @param oldGasLimit Original gas limit used to send the message.
  /// @param newGasLimit New gas limit to be used for this message.
  function replayMessage(
    address from,
    address to,
    uint256 value,
    bytes memory message,
    uint256 queueIndex,
    uint32 oldGasLimit,
    uint32 newGasLimit
  ) external;
}
