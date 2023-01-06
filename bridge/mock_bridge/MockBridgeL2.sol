// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

contract MockBridgeL2 {
  /*********************************
   * Events from L2ScrollMessenger *
   *********************************/

  event SentMessage(
    address indexed target,
    address sender,
    uint256 value,
    uint256 fee,
    uint256 deadline,
    bytes message,
    uint256 messageNonce,
    uint256 gasLimit
  );

  event MessageDropped(bytes32 indexed msgHash);
  
  event RelayedMessage(bytes32 indexed msgHash);
  
  event FailedRelayedMessage(bytes32 indexed msgHash);

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
    uint256 _fee,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable {
    // solhint-disable-next-line not-rely-on-time
    uint256 _deadline = block.timestamp + 1 days;
    uint256 _nonce = messageNonce;
    uint256 _value;
    unchecked {
      _value = msg.value - _fee;
    }
    bytes32 _msghash = keccak256(abi.encodePacked(msg.sender, _to, _value, _fee, _deadline, _nonce, _message));
    emit SentMessage(_to, msg.sender, _value, _fee, _deadline, _message, _nonce, _gasLimit);
    messageNonce = _nonce + 1;
  }

  function relayMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message
  ) external {
    bytes32 _msghash = keccak256(abi.encodePacked(_from, _to, _value, _fee, _deadline, _nonce, _message));
    emit RelayedMessage(_msghash);
  }
}
