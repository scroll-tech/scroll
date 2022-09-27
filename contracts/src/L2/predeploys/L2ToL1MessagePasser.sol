// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

/// @title L2ToL1MessagePasser
/// @notice The original idea is from Optimism, see [OVM_L2ToL1MessagePasser](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L2/predeploys/OVM_L2ToL1MessagePasser.sol).
/// The L2 to L1 Message Passer is a utility contract which facilitate an L1 proof of the
/// of a message on L2. The L1 Cross Domain Messenger performs this proof in its
/// _verifyStorageProof function, which verifies the existence of the transaction hash in this
/// contract's `sentMessages` mapping.
contract L2ToL1MessagePasser {
  address public immutable messenger;

  /// @notice Mapping from message hash to sent messages.
  mapping(bytes32 => bool) public sentMessages;

  constructor(address _messenger) {
    messenger = _messenger;
  }

  function passMessageToL1(bytes32 _messageHash) public {
    require(msg.sender == messenger, "only messenger");

    require(!sentMessages[_messageHash], "duplicated message");

    sentMessages[_messageHash] = true;
  }
}
