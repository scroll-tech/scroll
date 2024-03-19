// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

import {ChunkCodecV0} from "./ChunkCodecV0.sol";

/// @dev Below is the encoding for `Chunk`, total 60*n+1 bytes.
/// The only difference between `ChunkCodecV0` is we remove `l2Transactions` from chunk encoding.
/// ```text
///   * Field           Bytes       Type            Index       Comments
///   * numBlocks       1           uint8           0           The number of blocks in this chunk
///   * block[0]        60          BlockContext    1           The first block in this chunk
///   * ......
///   * block[i]        60          BlockContext    60*i+1      The (i+1)'th block in this chunk
///   * ......
///   * block[n-1]      60          BlockContext    60*n-59     The last block in this chunk
/// ```
///
/// @dev Below is the encoding for `BlockContext`, total 60 bytes.
/// ```text
///   * Field                   Bytes      Type         Index  Comments
///   * blockNumber             8          uint64       0      The height of this block.
///   * timestamp               8          uint64       8      The timestamp of this block.
///   * baseFee                 32         uint256      16     The base fee of this block.
///   * gasLimit                8          uint64       48     The gas limit of this block.
///   * numTransactions         2          uint16       56     The number of transactions in this block, both L1 & L2 txs.
///   * numL1Messages           2          uint16       58     The number of l1 messages in this block.
/// ```
library ChunkCodecV1 {
    /// @dev Thrown when no blocks in chunk.
    error ErrorNoBlockInChunk();

    /// @dev Thrown when the length of chunk is incorrect.
    error ErrorIncorrectChunkLength();

    /// @dev The length of one block context.
    uint256 internal constant BLOCK_CONTEXT_LENGTH = 60;

    /// @notice Validate the length of chunk.
    /// @param chunkPtr The start memory offset of the chunk in memory.
    /// @param _length The length of the chunk.
    /// @return _numBlocks The number of blocks in current chunk.
    function validateChunkLength(uint256 chunkPtr, uint256 _length) internal pure returns (uint256 _numBlocks) {
        _numBlocks = getNumBlocks(chunkPtr);

        // should contain at least one block
        if (_numBlocks == 0) revert ErrorNoBlockInChunk();

        // should contain the number of the blocks and block contexts
        if (_length != 1 + _numBlocks * BLOCK_CONTEXT_LENGTH) revert ErrorIncorrectChunkLength();
    }

    /// @notice Return the number of blocks in current chunk.
    /// @param chunkPtr The start memory offset of the chunk in memory.
    /// @return _numBlocks The number of blocks in current chunk.
    function getNumBlocks(uint256 chunkPtr) internal pure returns (uint256 _numBlocks) {
        return ChunkCodecV0.getNumBlocks(chunkPtr);
    }

    /// @notice Copy the block context to another memory.
    /// @param chunkPtr The start memory offset of the chunk in memory.
    /// @param dstPtr The destination memory offset to store the block context.
    /// @param index The index of block context to copy.
    /// @return uint256 The new destination memory offset after copy.
    function copyBlockContext(
        uint256 chunkPtr,
        uint256 dstPtr,
        uint256 index
    ) internal pure returns (uint256) {
        return ChunkCodecV0.copyBlockContext(chunkPtr, dstPtr, index);
    }

    /// @notice Return the number of transactions in current block.
    /// @param blockPtr The start memory offset of the block context in memory.
    /// @return _numTransactions The number of transactions in current block.
    function getNumTransactions(uint256 blockPtr) internal pure returns (uint256 _numTransactions) {
        return ChunkCodecV0.getNumTransactions(blockPtr);
    }

    /// @notice Return the number of L1 messages in current block.
    /// @param blockPtr The start memory offset of the block context in memory.
    /// @return _numL1Messages The number of L1 messages in current block.
    function getNumL1Messages(uint256 blockPtr) internal pure returns (uint256 _numL1Messages) {
        return ChunkCodecV0.getNumL1Messages(blockPtr);
    }
}
