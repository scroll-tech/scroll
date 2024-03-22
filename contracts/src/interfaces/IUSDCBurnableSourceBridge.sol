// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

/// @title IUSDCBurnableSourceBridge
/// @notice The interface of `USDCBurnableSourceBridge` of Circle's upgrader in L1 (Ethereum).
interface IUSDCBurnableSourceBridge {
    /**
     * @notice Called by Circle, this executes a burn on the source
     * chain.
     */
    function burnAllLockedUSDC() external;
}
