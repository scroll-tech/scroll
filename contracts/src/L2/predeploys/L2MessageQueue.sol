// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {AppendOnlyMerkleTree} from "../../libraries/common/AppendOnlyMerkleTree.sol";
import {OwnableBase} from "../../libraries/common/OwnableBase.sol";

/// @title L2MessageQueue
/// @notice The original idea is from Optimism, see [OVM_L2ToL1MessagePasser](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L2/predeploys/OVM_L2ToL1MessagePasser.sol).
/// The L2 to L1 Message Passer is a utility contract which facilitate an L1 proof of the
/// of a message on L2. The L1 Cross Domain Messenger performs this proof in its
/// _verifyStorageProof function, which verifies the existence of the transaction hash in this
/// contract's `sentMessages` mapping.
contract L2MessageQueue is AppendOnlyMerkleTree, OwnableBase {
    /**********
     * Events *
     **********/

    /// @notice Emitted when a new message is added to the merkle tree.
    /// @param index The index of the corresponding message.
    /// @param messageHash The hash of the corresponding message.
    event AppendMessage(uint256 index, bytes32 messageHash);

    /// @notice Emits each time the owner updates the address of `messenger`.
    /// @param oldMessenger The address of old messenger.
    /// @param newMessenger The address of new messenger.
    event UpdateMessenger(address indexed oldMessenger, address indexed newMessenger);

    /*************
     * Variables *
     *************/

    /// @notice The address of L2ScrollMessenger contract.
    address public messenger;

    /***************
     * Constructor *
     ***************/

    constructor(address _owner) {
        _transferOwnership(_owner);
    }

    function initialize() external {
        _initializeMerkleTree();
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice record the message to merkle tree and compute the new root.
    /// @param _messageHash The hash of the new added message.
    function appendMessage(bytes32 _messageHash) external returns (bytes32) {
        require(msg.sender == messenger, "only messenger");

        (uint256 _currentNonce, bytes32 _currentRoot) = _appendMessageHash(_messageHash);

        // We can use the event to compute the merkle tree locally.
        emit AppendMessage(_currentNonce, _messageHash);

        return _currentRoot;
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the address of messenger.
    /// @dev You are not allowed to update messenger when there are some messages appended.
    /// @param _newMessenger The address of messenger to update.
    function updateMessenger(address _newMessenger) external onlyOwner {
        require(nextMessageIndex == 0, "cannot update messenger");

        address _oldMessenger = messenger;
        messenger = _newMessenger;

        emit UpdateMessenger(_oldMessenger, _newMessenger);
    }
}
