// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

contract MockBridgeL2 {
  /*********************************
   * Events from L2ScrollMessenger *
   *********************************/

  event SentMessage(
    address indexed sender,
    address indexed target,
    uint256 value,
    uint256 messageNonce,
    uint256 gasLimit,
    bytes message
  );
  
  event RelayedMessage(bytes32 indexed messageHash);

  /*************
   * Variables *
   *************/

  /// @notice Message nonce, used to avoid relay attack.
  uint256 public messageNonce;

  /************************************
   * Functions from L2ScrollMessenger *
   ************************************/

  function sendMessage(
    address _to,
    uint256 _value,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable {
    emit SentMessage(msg.sender, _to, _value, messageNonce, _gasLimit, _message);
    messageNonce += 1;
  }
}
