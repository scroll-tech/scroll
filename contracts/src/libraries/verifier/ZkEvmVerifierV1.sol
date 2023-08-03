// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {IZkEvmVerifier} from "./IZkEvmVerifier.sol";

// solhint-disable no-inline-assembly

contract ZkEvmVerifierV1 is IZkEvmVerifier {
    error VerificationFailed();

    /// @notice The address of highly optimized plonk verifier contract.
    address public immutable plonkVerifier;

    constructor(address _verifier) {
        plonkVerifier = _verifier;
    }

    /// @notice Verify aggregate zk proof.
    /// @param aggrProof The aggregated proof.
    /// @param publicInputHash The public input hash.
    function verify(bytes calldata aggrProof, bytes32 publicInputHash) external view override {
        address _verifier = plonkVerifier;
        bool success;

        // 1. the first 12 * 32 (0x180) bytes of `aggrProof` is `accumulator`
        // 2. the rest bytes of `aggrProof` if the actual `batch_aggregated_proof`
        // 3. each byte of the `reverse(public_input_hash)` should be converted to a `uint256` and the
        //    1024 (0x400) bytes should inserted between `accumulator` and `batch_aggregated_proof`.
        assembly {
            let p := mload(0x40)
            calldatacopy(p, aggrProof.offset, 0x180)
            for {
                let i := 0
            } lt(i, 0x400) {
                i := add(i, 0x20)
            } {
                mstore(add(p, sub(0x560, i)), and(publicInputHash, 0xff))
                publicInputHash := shr(8, publicInputHash)
            }
            calldatacopy(add(p, 0x580), add(aggrProof.offset, 0x180), sub(aggrProof.length, 0x180))

            success := staticcall(gas(), _verifier, p, add(aggrProof.length, 0x400), 0x00, 0x00)
        }

        if (!success) {
            revert VerificationFailed();
        }
    }
}
