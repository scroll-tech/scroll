// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

/**
 * @dev A library that provides the necessary calculations to calculate the global exit root
 */
library GlobalExitRootLib {
    function calculateGlobalExitRoot(
        bytes32 mainnetExitRoot,
        bytes32 rollupExitRoot
    ) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(mainnetExitRoot, rollupExitRoot));
    }
}
