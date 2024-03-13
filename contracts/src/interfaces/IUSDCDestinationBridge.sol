// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

/// @title IUSDCDestinationBridge
/// @notice The interface required for USDC bridge in the destination chain (Scroll).
interface IUSDCDestinationBridge {
    /**
     * @notice Called by Circle, this transfers FiatToken roles to the designated owner.
     */
    function transferUSDCRoles(address owner) external;
}
