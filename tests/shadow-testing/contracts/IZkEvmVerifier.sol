// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

interface IZkEvmVerifierV1 {
    /// @notice Verify aggregate zk proof.
    /// @param aggrProof The aggregated proof.
    /// @param publicInputHash The public input hash.
    function verify(bytes calldata aggrProof, bytes32 publicInputHash) external view;
}

interface IZkEvmVerifierV2 {
    /// @notice Verify bundle zk proof.
    /// @param bundleProof The bundle recursion proof.
    /// @param publicInput The public input.
    function verify(bytes calldata bundleProof, bytes calldata publicInput) external view;
}
