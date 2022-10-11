//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

contract Mock_Bridge {

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

    /// @notice Message nonce, used to avoid relay attack.
  uint256 public messageNonce;

 function sendMessage(
    address _to,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable {
    // solhint-disable-next-line not-rely-on-time
    uint256 _deadline = block.timestamp + 1 days;
    // @todo compute fee
    uint256 _fee = 0;
    uint256 _nonce = messageNonce;
    require(msg.value >= _fee, "cannot pay fee");
    uint256 _value;
    unchecked {
      _value = msg.value - _fee;
    }

    emit SentMessage(_to, msg.sender, _value, _fee, _deadline, _message, _nonce, _gasLimit);

    unchecked {
      messageNonce = _nonce + 1;
    }
  }
}
