// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {ZkTrieVerifier} from "../libraries/verifier/ZkTrieVerifier.sol";

contract MockZkTrieVerifier {
    address public immutable poseidon;

    constructor(address _poseidon) {
        poseidon = _poseidon;
    }

    function verifyZkTrieProof(
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
        (stateRoot, storageValue) = ZkTrieVerifier.verifyZkTrieProof(poseidon, account, storageKey, proof);
        gasUsed = start - gasleft();
    }
}
