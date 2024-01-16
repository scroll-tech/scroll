// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

interface IZkTrieVerifier {
    /// @notice Internal function to validates a proof from eth_getProof.
    /// @param account The address of the contract.
    /// @param storageKey The storage slot to verify.
    /// @param proof The rlp encoding result of eth_getProof.
    /// @return stateRoot The computed state root. Must be checked by the caller.
    /// @return storageValue The value of `storageKey`.
    ///
    /// The encoding order of `proof` is
    /// ```text
    /// |        1 byte        |      ...      |        1 byte        |      ...      |
    /// | account proof length | account proof | storage proof length | storage proof |
    /// ```
    function verifyZkTrieProof(
        address account,
        bytes32 storageKey,
        bytes calldata proof
    ) external view returns (bytes32 stateRoot, bytes32 storageValue);
}
