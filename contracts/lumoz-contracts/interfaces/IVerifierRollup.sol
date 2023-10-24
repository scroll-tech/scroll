// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

/**
 * @dev Define interface verifier
 */
interface IVerifierRollup {
    function verifyProof(
        bytes memory proof, 
        uint256[1] memory pubSignals
    ) external view returns (bool);
}
