// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {WithdrawTrieVerifier} from "../libraries/verifier/WithdrawTrieVerifier.sol";

contract WithdrawTrieVerifierTest is DSTestPlus {
    function testInvalidProof() public {
        hevm.expectRevert("Invalid proof");
        WithdrawTrieVerifier.verifyMerkleProof(bytes32(uint256(1)), bytes32(uint256(1)), 1, hex"00");
    }

    function testMerkleProof() public {
        bytes32[] memory roots = new bytes32[](4);
        bytes32[] memory hashes = new bytes32[](4);
        uint256[] memory nonces = new uint256[](4);
        bytes[] memory proofs = new bytes[](4);

        // generated from bridge folder
        // adding leaves one at a time (0x0..1, 0x0..2, 0x0..3, 0x0..4)

        roots[0] = hex"0000000000000000000000000000000000000000000000000000000000000001";
        hashes[0] = hex"0000000000000000000000000000000000000000000000000000000000000001";
        nonces[0] = 0;
        proofs[0] = hex"";

        roots[1] = hex"e90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0";
        hashes[1] = hex"0000000000000000000000000000000000000000000000000000000000000002";
        nonces[1] = 1;
        proofs[1] = hex"0000000000000000000000000000000000000000000000000000000000000001";

        roots[2] = hex"222ff5e0b5877792c2bc1670e2ccd0c2c97cd7bb1672a57d598db05092d3d72c";
        hashes[2] = hex"0000000000000000000000000000000000000000000000000000000000000003";
        nonces[2] = 2;
        proofs[
            2
        ] = hex"0000000000000000000000000000000000000000000000000000000000000000e90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0";

        roots[3] = hex"a9bb8c3f1f12e9aa903a50c47f314b57610a3ab32f2d463293f58836def38d36";
        hashes[3] = hex"0000000000000000000000000000000000000000000000000000000000000004";
        nonces[3] = 3;
        proofs[
            3
        ] = hex"0000000000000000000000000000000000000000000000000000000000000003e90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0";

        for (uint256 i = 0; i < 4; i++) {
            require(
                WithdrawTrieVerifier.verifyMerkleProof(roots[i], hashes[i], nonces[i], proofs[i]) == true,
                "WithdrawTrieVerifier: verifyMerkleProof failed"
            );
        }
    }
}
