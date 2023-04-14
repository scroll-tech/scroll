// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IScrollChain {
    /**********
     * Events *
     **********/

    /// @notice Emitted when a new batch is commited.
    /// @param batchHash The hash of the batch
    event CommitBatch(bytes32 indexed batchHash);

    /// @notice Emitted when a batch is reverted.
    /// @param batchHash The identification of the batch.
    event RevertBatch(bytes32 indexed batchHash);

    /// @notice Emitted when a batch is finalized.
    /// @param batchHash The hash of the batch
    event FinalizeBatch(bytes32 indexed batchHash);

    /***********
     * Structs *
     ***********/

    struct BlockContext {
        // The hash of this block.
        bytes32 blockHash;
        // The parent hash of this block.
        bytes32 parentHash;
        // The height of this block.
        uint64 blockNumber;
        // The timestamp of this block.
        uint64 timestamp;
        // The base fee of this block.
        // Currently, it is not used, because we disable EIP-1559.
        // We keep it for future proof.
        uint256 baseFee;
        // The gas limit of this block.
        uint64 gasLimit;
        // The number of transactions in this block, both L1 & L2 txs.
        uint16 numTransactions;
        // The number of l1 messages in this block.
        uint16 numL1Messages;
    }

    struct Batch {
        // The list of blocks in this batch
        BlockContext[] blocks; // MAX_NUM_BLOCKS = 100, about 5 min
        // The state root of previous batch.
        // The first batch will use 0x0 for prevStateRoot
        bytes32 prevStateRoot;
        // The state root of the last block in this batch.
        bytes32 newStateRoot;
        // The withdraw trie root of the last block in this batch.
        bytes32 withdrawTrieRoot;
        // The index of the batch.
        uint64 batchIndex;
        // The parent batch hash.
        bytes32 parentBatchHash;
        // Concatenated raw data of RLP encoded L2 txs
        bytes l2Transactions;
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @notice Return whether the batch is finalized by batch hash.
    /// @param batchHash The hash of the batch to query.
    function isBatchFinalized(bytes32 batchHash) external view returns (bool);

    /// @notice Return the merkle root of L2 message tree.
    /// @param batchHash The hash of the batch to query.
    function getL2MessageRoot(bytes32 batchHash) external view returns (bytes32);

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice commit a batch in layer 1
    /// @param batch The layer2 batch to commit.
    function commitBatch(Batch memory batch) external;

    /// @notice commit a list of batches in layer 1
    /// @param batches The list of layer2 batches to commit.
    function commitBatches(Batch[] memory batches) external;

    /// @notice revert a pending batch.
    /// @dev one can only revert unfinalized batches.
    /// @param batchId The identification of the batch.
    function revertBatch(bytes32 batchId) external;

    /// @notice finalize commited batch in layer 1
    /// @dev will add more parameters if needed.
    /// @param batchId The identification of the commited batch.
    /// @param proof The corresponding proof of the commited batch.
    /// @param instances Instance used to verify, generated from batch.
    function finalizeBatchWithProof(
        bytes32 batchId,
        uint256[] memory proof,
        uint256[] memory instances
    ) external;
}
