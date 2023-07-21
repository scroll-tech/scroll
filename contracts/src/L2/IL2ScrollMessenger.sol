// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {IScrollMessenger} from "../libraries/IScrollMessenger.sol";

interface IL2ScrollMessenger is IScrollMessenger {
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
}
