// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

interface IZkEvmVerifier {
    /// @notice Verify aggregate zk proof.
    /// @param aggrProof The aggregated proof.
    /// @param publicInputHash The public input hash.
    function verify(bytes calldata aggrProof, bytes32 publicInputHash) external view;
}
