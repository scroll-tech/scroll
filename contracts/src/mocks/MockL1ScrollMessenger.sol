// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import "../L1/L1ScrollMessenger.sol";
import "../L1/rollup/ZKRollup.sol";

contract MockL1ScrollMessenger is L1ScrollMessenger {
  /// @inheritdoc IL1ScrollMessenger
  /// @dev Mock function to skip verification logic.
  function relayMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message,
    L2MessageProof memory _proof
  ) external override whenNotPaused onlyWhitelistedSender(msg.sender) {
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "already in execution");

    // solhint-disable-next-line not-rely-on-time
    // @note disable for now since we cannot generate proof in time.
    // require(_deadline >= block.timestamp, "Message expired");

    bytes32 _msghash = keccak256(abi.encodePacked(_from, _to, _value, _fee, _deadline, _nonce, _message));

    require(!isMessageExecuted[_msghash], "Message successfully executed");

    (, , , , bytes32 _messageRoot) = ZKRollup(rollup).blocks(_proof.blockHash);
    require(ZkTrieVerifier.verifyMerkleProof(_messageRoot, _msghash, _nonce, _proof.merkleProof), "invalid proof");

    // @note This usually will never happen, just in case.
    require(_from != xDomainMessageSender, "invalid message sender");

    xDomainMessageSender = _from;
    // solhint-disable-next-line avoid-low-level-calls
    (bool success, ) = _to.call{ value: _value }(_message);
    // reset value to refund gas.
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

    if (success) {
      isMessageExecuted[_msghash] = true;
      emit RelayedMessage(_msghash);
    } else {
      emit FailedRelayedMessage(_msghash);
    }

    bytes32 _relayId = keccak256(abi.encodePacked(_msghash, msg.sender, block.number));

    isMessageRelayed[_relayId] = true;
  }
}
