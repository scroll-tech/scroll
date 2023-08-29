// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

// Implement this on the source chain (Ethereum).
interface IUSDCBurnableSourceBridge {
    /**
     * @notice Called by Circle, this executes a burn on the source
     * chain.
     */
    function burnAllLockedUSDC() external;
}
