// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { IScrollMessenger } from "../libraries/IScrollMessenger.sol";

interface IL1ScrollMessenger is IScrollMessenger {
  struct L2MessageProof {
    // @todo add more fields
    uint256 blockNumber;
    bytes merkleProof;
  }

  /**************************************** Mutated Functions ****************************************/

  /// @notice execute L2 => L1 message
  /// @param _from The address of the sender of the message.
  /// @param _to The address of the recipient of the message.
  /// @param _value The msg.value passed to the message call.
  /// @param _fee The amount of fee in ETH to charge.
  /// @param _deadline The deadline of the message.
  /// @param _nonce The nonce of the message to avoid replay attack.
  /// @param _message The content of the message.
  /// @param _proof The proof used to verify the correctness of the transaction.
  function relayMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message,
    L2MessageProof memory _proof
  ) external;

  /// @notice Replay an exsisting message.
  /// @param _from The address of the sender of the message.
  /// @param _to The address of the recipient of the message.
  /// @param _value The msg.value passed to the message call.
  /// @param _fee The amount of fee in ETH to charge.
  /// @param _deadline The deadline of the message.
  /// @param _message The content of the message.
  /// @param _queueIndex CTC Queue index for the message to replay.
  /// @param _oldGasLimit Original gas limit used to send the message.
  /// @param _newGasLimit New gas limit to be used for this message.
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
  ) external;
}
