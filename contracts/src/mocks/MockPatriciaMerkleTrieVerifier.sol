// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {PatriciaMerkleTrieVerifier} from "../libraries/verifier/PatriciaMerkleTrieVerifier.sol";

contract MockPatriciaMerkleTrieVerifier {
    function verifyPatriciaProof(
        address account,
        bytes32 storageKey,
        bytes calldata proof
    )
        external
        view
        returns (
            bytes32 stateRoot,
            bytes32 storageValue,
            uint256 gasUsed
        )
    {
        uint256 start = gasleft();
        (stateRoot, storageValue) = PatriciaMerkleTrieVerifier.verifyPatriciaProof(account, storageKey, proof);
        gasUsed = start - gasleft();
    }
}
