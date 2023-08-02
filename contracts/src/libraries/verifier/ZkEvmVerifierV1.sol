// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {IZkEvmVerifier} from "./IZkEvmVerifier.sol";

contract ZkEvmVerifierV1 is IZkEvmVerifier {
    /// @notice The address of highly optimized plonk verifier contract.
    address public immutable plonkVerifier;

    constructor(address _verifier) {
        plonkVerifier = _verifier;
    }

    /// @notice Verify aggregate zk proof.
    /// @param aggrProof The aggregated proof.
    /// @param publicInputHash The public input hash.
    function verify(bytes calldata aggrProof, bytes32 publicInputHash) external view override {
        // calldata passed to plonk verifier is `concat(aggrProof, publicInputHash)`.
        (bool success, ) = plonkVerifier.staticcall(abi.encodePacked(aggrProof, publicInputHash));
        require(success, "verification failed");
    }
}
