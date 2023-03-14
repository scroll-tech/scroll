// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { ScrollChain } from "./ScrollChain.sol";
import { ZkTrieVerifier } from "../../libraries/verifier/ZkTrieVerifier.sol";

contract ScrollChainCommitmentVerifier {
  /// @notice The address of poseidon hash contract
  address public immutable poseidon;

  /// @notice The address of ScrollChain contract.
  address public immutable rollup;

  constructor(address _poseidon, address _rollup) {
    poseidon = _poseidon;
    rollup = _rollup;
  }

  /// @notice Validates a proof from eth_getProof in l2geth.
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
  ) public view returns (bytes32 stateRoot, bytes32 storageValue) {
    return ZkTrieVerifier.verifyZkTrieProof(poseidon, account, storageKey, proof);
  }

  /// @notice Verifies a batch inclusion proof.
  /// @param batchHash The hash of the batch.
  /// @param account The address of the contract in L2.
  /// @param storageKey The storage key inside the contract in L2.
  /// @param proof The rlp encoding result of eth_getProof.
  /// @return storageValue The value of `storageKey`.
  function verifyStateCommitment(
    bytes32 batchHash,
    address account,
    bytes32 storageKey,
    bytes calldata proof
  ) external view returns (bytes32 storageValue) {
    require(ScrollChain(rollup).isBatchFinalized(batchHash), "Batch not finalized");

    bytes32 computedStateRoot;
    (computedStateRoot, storageValue) = ZkTrieVerifier.verifyZkTrieProof(poseidon, account, storageKey, proof);
    (bytes32 expectedStateRoot, , , , , , , ) = ScrollChain(rollup).batches(batchHash);
    require(computedStateRoot == expectedStateRoot, "Invalid inclusion proof");
  }
}
