// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

import {IScrollMessenger} from "../libraries/IScrollMessenger.sol";

interface IL1ScrollMessenger is IScrollMessenger {
    /**********
     * Events *
     **********/

    /// @notice Emitted when the maximum number of times each message can be replayed is updated.
    /// @param oldMaxReplayTimes The old maximum number of times each message can be replayed.
    /// @param newMaxReplayTimes The new maximum number of times each message can be replayed.
    event UpdateMaxReplayTimes(uint256 oldMaxReplayTimes, uint256 newMaxReplayTimes);

    /***********
     * Structs *
     ***********/

    struct L2MessageProof {
        // The index of the batch where the message belongs to.
        uint256 batchIndex;
        // Concatenation of merkle proof for withdraw merkle trie.
        bytes merkleProof;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Relay a L2 => L1 message with message proof.
    /// @param from The address of the sender of the message.
    /// @param to The address of the recipient of the message.
    /// @param value The msg.value passed to the message call.
    /// @param nonce The nonce of the message to avoid replay attack.
    /// @param message The content of the message.
    /// @param proof The proof used to verify the correctness of the transaction.
    function relayMessageWithProof(
        address from,
        address to,
        uint256 value,
        uint256 nonce,
        bytes memory message,
        L2MessageProof memory proof
    ) external;

    /// @notice Replay an existing message.
    /// @param from The address of the sender of the message.
    /// @param to The address of the recipient of the message.
    /// @param value The msg.value passed to the message call.
    /// @param messageNonce The nonce for the message to replay.
    /// @param message The content of the message.
    /// @param newGasLimit New gas limit to be used for this message.
    /// @param refundAddress The address of account who will receive the refunded fee.
    function replayMessage(
        address from,
        address to,
        uint256 value,
        uint256 messageNonce,
        bytes memory message,
        uint32 newGasLimit,
        address refundAddress
    ) external payable;

    /// @notice Drop a skipped message.
    /// @param from The address of the sender of the message.
    /// @param to The address of the recipient of the message.
    /// @param value The msg.value passed to the message call.
    /// @param messageNonce The nonce for the message to drop.
    /// @param message The content of the message.
    function dropMessage(
        address from,
        address to,
        uint256 value,
        uint256 messageNonce,
        bytes memory message
    ) external;
}
