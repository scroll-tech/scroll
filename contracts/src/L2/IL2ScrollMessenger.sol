// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {IScrollMessenger} from "../libraries/IScrollMessenger.sol";

interface IL2ScrollMessenger is IScrollMessenger {
    /***********
     * Structs *
     ***********/

    struct L1MessageProof {
        bytes32 blockHash;
        bytes stateRootProof;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice execute L1 => L2 message
    /// @dev Make sure this is only called by privileged accounts.
    /// @param from The address of the sender of the message.
    /// @param to The address of the recipient of the message.
    /// @param value The msg.value passed to the message call.
    /// @param nonce The nonce of the message to avoid replay attack.
    /// @param message The content of the message.
    function relayMessage(
        address from,
        address to,
        uint256 value,
        uint256 nonce,
        bytes calldata message
    ) external;

    /// @notice execute L1 => L2 message with proof
    /// @param from The address of the sender of the message.
    /// @param to The address of the recipient of the message.
    /// @param value The msg.value passed to the message call.
    /// @param nonce The nonce of the message to avoid replay attack.
    /// @param message The content of the message.
    /// @param proof The message proof.
    function retryMessageWithProof(
        address from,
        address to,
        uint256 value,
        uint256 nonce,
        bytes calldata message,
        L1MessageProof calldata proof
    ) external;
}
