// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

// Implement this on the destination chain (Scroll).
interface IUSDCDestinationBridge {
    /**
     * @notice Called by Circle, this transfers FiatToken roles to the designated owner.
     */
    function transferUSDCRoles(address owner) external;
}
