// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IScrollChain {
    /**********
     * Events *
     **********/

    /// @notice Emitted when a new batch is commited.
    /// @param batchHash The hash of the batch.
    event CommitBatch(bytes32 indexed batchHash);

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
    /// @dev Below is the encoding for `BatchHeader`, total 161 bytes.
    /// ```text
    ///   * Field                   Bytes      Type       Index  Comments
    ///   * version                 1          uint8      0      The batch version
    ///   * batchIndex              8          uint64     1      The index of the batch
    ///   * l1MessagePopped         8          uint64     9      Number of L1 message popped in the batch
    ///   * totalL1MessagePopped    8          uint64     17     Number of total L1 message popped after the batch
    ///   * dataHash                32         bytes32    25     The data hash of the batch
    ///   * lastBlockHash           32         bytes32    57     The block hash of the last block in the batch
    ///   * skippedL1MessageBitmap  32         bytes32    89     A bitmap to indicate if L1 messages are skipped in the batch
    ///   * parentBatchHash         32         bytes32    121    The parent batch hash
    ///   * timestamp               8          uint64     153    The block timestamp when this bath committed.
    /// ```
    ///
    /// @dev Below is the encoding for `Chunk`, total 156*n+1+m bytes.
    /// ```text
    ///   * Field           Bytes       Type            Index       Comments
    ///   * numBlocks       1           uint8           0           The number of blocks in this chunk
    ///   * block[0]        156         BlockContext    1           The first block in this chunk
    ///   * ......
    ///   * block[i]        156         BlockContext    156*i+1     The first block in this chunk
    ///   * ......
    ///   * block[n-1]      156         BlockContext    156*n-155   The last block in this chunk
    ///   * l2Transactions  dynamic     bytes           156*n+1
    /// ```
    ///
    /// @dev Below is the encoding for `BlockContext`, total 156 bytes.
    /// ```text
    ///   * Field                   Bytes      Type         Index  Comments
    ///   * blockHash               32         bytes32      0      The hash of this block.
    ///   * parentHash              32         bytes32      32     The parent hash of this block.
    ///   * blockNumber             8          uint64       64     The height of this block.
    ///   * timestamp               8          uint64       72     The timestamp of this block.
    ///   * baseFee                 32         uint256      80     The base fee of this block. Currently, it is not used, because we disable EIP-1559.
    ///   * gasLimit                8          uint64       112    The gas limit of this block.
    ///   * numTransactions         2          uint16       120    The number of transactions in this block, both L1 & L2 txs.
    ///   * numL1Messages           2          uint16       122    The number of l1 messages in this block.
    ///   * skippedL1MessageBitmap  32         uint256      124    A bitmap to indicate if L1 messages are skipped in the block
    /// ```
    ///
    /// @param version The version of current batch.
    /// @param parentBatchHeader The header of parent batch, see the encoding above.
    /// @param chunks The list of encoded chunks, see the encoding above.
    function commitBatch(
        uint8 version,
        bytes calldata parentBatchHeader,
        bytes[] memory chunks
    ) external;

    /// @notice Finalize commited batch in layer 1
    /// @param batchHeader The header of current batch, see the encoding in comments of `commitBatch.
    /// @param prevStateRoot The state root of parent batch.
    /// @param newStateRoot The state root of current batch.
    /// @param withdrawRoot The withdraw trie root of current batch.
    /// @param aggrProof The aggregated proof for current batch.
    function finalizeBatchWithProof(
        bytes calldata batchHeader,
        bytes32 prevStateRoot,
        bytes32 newStateRoot,
        bytes32 withdrawRoot,
        bytes calldata aggrProof
    ) external;
}
