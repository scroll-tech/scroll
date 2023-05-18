// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IScrollChain {
    /**********
     * Events *
     **********/

    /// @notice Emitted when a new batch is commited.
    /// @param batchHash The hash of the batch.
    event CommitBatch(bytes32 indexed batchHash);

    /// @notice revert a pending batch.
    /// @param batchHash The hash of the batch
    event RevertBatch(bytes32 indexed batchHash);

    /// @notice Emitted when a batch is finalized.
    /// @param batchHash The hash of the batch
    /// @param stateRoot The state root in layer 2 after this batch.
    /// @param withdrawRoot The merkle root in layer2 after this batch.
    event FinalizeBatch(bytes32 indexed batchHash, bytes32 stateRoot, bytes32 withdrawRoot);

    /*************************
     * Public View Functions *
     *************************/

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

    /// @notice commit a batch in layer 1
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
    /// @param batchHeader The header of current batch, see the encoding in comments of `commitBatch.
    function revertBatch(bytes calldata batchHeader) external;

    /// @notice Finalize commited batch in layer 1
    /// @param batchHeader The header of current batch, see the encoding in comments of `commitBatch.
    /// @param prevStateRoot The state root of parent batch.
    /// @param newStateRoot The state root of current batch.
    /// @param withdrawRoot The withdraw trie root of current batch.
    /// @param aggrProof The aggregation proof for current batch.
    function finalizeBatchWithProof(
        bytes calldata batchHeader,
        bytes32 prevStateRoot,
        bytes32 newStateRoot,
        bytes32 withdrawRoot,
        bytes calldata aggrProof
    ) external;
}
