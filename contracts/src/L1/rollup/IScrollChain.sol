// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

interface IScrollChain {
    /**********
     * Events *
     **********/

    /// @notice Emitted when a new batch is committed.
    /// @param batchIndex The index of the batch.
    /// @param batchHash The hash of the batch.
    event CommitBatch(uint256 indexed batchIndex, bytes32 indexed batchHash);

    /// @notice revert a pending batch.
    /// @param batchIndex The index of the batch.
    /// @param batchHash The hash of the batch
    event RevertBatch(uint256 indexed batchIndex, bytes32 indexed batchHash);

    /// @notice Emitted when a batch is finalized.
    /// @param batchIndex The index of the batch.
    /// @param batchHash The hash of the batch
    /// @param stateRoot The state root on layer 2 after this batch.
    /// @param withdrawRoot The merkle root on layer2 after this batch.
    event FinalizeBatch(uint256 indexed batchIndex, bytes32 indexed batchHash, bytes32 stateRoot, bytes32 withdrawRoot);

    /// @notice Emitted when owner updates the status of sequencer.
    /// @param account The address of account updated.
    /// @param status The status of the account updated.
    event UpdateSequencer(address indexed account, bool status);

    /// @notice Emitted when owner updates the status of prover.
    /// @param account The address of account updated.
    /// @param status The status of the account updated.
    event UpdateProver(address indexed account, bool status);

    /// @notice Emitted when the value of `maxNumTxInChunk` is updated.
    /// @param oldMaxNumTxInChunk The old value of `maxNumTxInChunk`.
    /// @param newMaxNumTxInChunk The new value of `maxNumTxInChunk`.
    event UpdateMaxNumTxInChunk(uint256 oldMaxNumTxInChunk, uint256 newMaxNumTxInChunk);

    /*************************
     * Public View Functions *
     *************************/

    /// @notice The latest finalized batch index.
    function lastFinalizedBatchIndex() external view returns (uint256);

    /// @notice Return the batch hash of a committed batch.
    /// @param batchIndex The index of the batch.
    function committedBatches(uint256 batchIndex) external view returns (bytes32);

    /// @notice Return the state root of a committed batch.
    /// @param batchIndex The index of the batch.
    function finalizedStateRoots(uint256 batchIndex) external view returns (bytes32);

    /// @notice Return the message root of a committed batch.
    /// @param batchIndex The index of the batch.
    function withdrawRoots(uint256 batchIndex) external view returns (bytes32);

    /// @notice Return whether the batch is finalized by batch index.
    /// @param batchIndex The index of the batch.
    function isBatchFinalized(uint256 batchIndex) external view returns (bool);

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Commit a batch of transactions on layer 1.
    ///
    /// @param version The version of current batch.
    /// @param parentBatchHeader The header of parent batch, see the comments of `BatchHeaderV0Codec`.
    /// @param chunks The list of encoded chunks, see the comments of `ChunkCodec`.
    /// @param skippedL1MessageBitmap The bitmap indicates whether each L1 message is skipped or not.
    function commitBatch(
        uint8 version,
        bytes calldata parentBatchHeader,
        bytes[] memory chunks,
        bytes calldata skippedL1MessageBitmap
    ) external;

    /// @notice Revert a pending batch.
    /// @dev one can only revert unfinalized batches.
    /// @param batchHeader The header of current batch, see the encoding in comments of `commitBatch`.
    /// @param count The number of subsequent batches to revert, including current batch.
    function revertBatch(bytes calldata batchHeader, uint256 count) external;

    /// @notice Finalize a committed batch on layer 1.
    /// @param batchHeader The header of current batch, see the encoding in comments of `commitBatch.
    /// @param prevStateRoot The state root of parent batch.
    /// @param postStateRoot The state root of current batch.
    /// @param withdrawRoot The withdraw trie root of current batch.
    /// @param aggrProof The aggregation proof for current batch.
    function finalizeBatchWithProof(
        bytes calldata batchHeader,
        bytes32 prevStateRoot,
        bytes32 postStateRoot,
        bytes32 withdrawRoot,
        bytes calldata aggrProof
    ) external;

    /// @notice Finalize a committed batch on layer 1 without providing proof.
    /// @param batchHeader The header of current batch, see the encoding in comments of `commitBatch.
    /// @param prevStateRoot The state root of parent batch.
    /// @param postStateRoot The state root of current batch.
    /// @param withdrawRoot The withdraw trie root of current batch.
    function finalizeBatch(
        bytes calldata batchHeader,
        bytes32 prevStateRoot,
        bytes32 postStateRoot,
        bytes32 withdrawRoot
    ) external;

    /// @notice Finalize a committed batch (with blob) on layer 1.
    ///
    /// @dev Memory layout of `blobDataProof`:
    /// |    z    |    y    | kzg_commitment | kzg_proof |
    /// |---------|---------|----------------|-----------|
    /// | bytes32 | bytes32 |    bytes48     |  bytes48  |
    ///
    /// @param batchHeader The header of current batch, see the encoding in comments of `commitBatch.
    /// @param prevStateRoot The state root of parent batch.
    /// @param postStateRoot The state root of current batch.
    /// @param withdrawRoot The withdraw trie root of current batch.
    /// @param blobDataProof The proof for blob data.
    /// @param aggrProof The aggregation proof for current batch.
    function finalizeBatchWithProof4844(
        bytes calldata batchHeader,
        bytes32 prevStateRoot,
        bytes32 postStateRoot,
        bytes32 withdrawRoot,
        bytes calldata blobDataProof,
        bytes calldata aggrProof
    ) external;

    /// @notice Finalize a committed batch (with blob) on layer 1 without providing proof.
    /// @param batchHeader The header of current batch, see the encoding in comments of `commitBatch`.
    /// @param prevStateRoot The state root of parent batch.
    /// @param postStateRoot The state root of current batch.
    /// @param withdrawRoot The withdraw trie root of current batch.
    /// @param blobDataProof The proof for blob data.
    function finalizeBatch4844(
        bytes calldata batchHeader,
        bytes32 prevStateRoot,
        bytes32 postStateRoot,
        bytes32 withdrawRoot,
        bytes calldata blobDataProof
    ) external;
}
