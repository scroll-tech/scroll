// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IL1MessageQueue {
  /**********
   * Events *
   **********/

  /// @notice Emitted when a L1 to L2 message is appended.
  /// @param msgHash The hash of the appended message.
  event AppendMessage(bytes32 indexed msgHash);

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Return the index of next appended message.
  /// @dev Also the total number of appended messages.
  function nextMessageIndex() external view returns (uint256);

  /// @notice Check whether the message with hash `_msgHash` exists.
  /// @param _msgHash The hash of the message to check.
  function hasMessage(bytes32 _msgHash) external view returns (bool);

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Append a L1 to L2 message into this contract.
  /// @param _msgHash The hash of the appended message.
  function appendMessage(bytes32 _msgHash) external;
}
