// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

/// @dev Below is the encoding for `Chunk`, total 60*n+1+m bytes.
/// ```text
///   * Field           Bytes       Type            Index       Comments
///   * numBlocks       1           uint8           0           The number of blocks in this chunk
///   * block[0]        60          BlockContext    1           The first block in this chunk
///   * ......
///   * block[i]        60          BlockContext    60*i+1      The (i+1)'th block in this chunk
///   * ......
///   * block[n-1]      60          BlockContext    60*n-59     The last block in this chunk
///   * l2Transactions  dynamic     bytes           60*n+1
/// ```
///
/// @dev Below is the encoding for `BlockContext`, total 60 bytes.
/// ```text
///   * Field                   Bytes      Type         Index  Comments
///   * blockNumber             8          uint64       0      The height of this block.
///   * timestamp               8          uint64       8      The timestamp of this block.
///   * baseFee                 32         uint256      16     The base fee of this block. Currently, it is always 0, because we disable EIP-1559.
///   * gasLimit                8          uint64       48     The gas limit of this block.
///   * numTransactions         2          uint16       56     The number of transactions in this block, both L1 & L2 txs.
///   * numL1Messages           2          uint16       58     The number of l1 messages in this block.
/// ```
library ChunkCodec {
    uint256 internal constant BLOCK_CONTEXT_LENGTH = 60;

    /// @notice Validate the length of chunk.
    /// @param chunkPtr The start memory offset of the chunk in memory.
    /// @param _length The length of the chunk.
    /// @return _numBlocks The number of blocks in current chunk.
    function validateChunkLength(uint256 chunkPtr, uint256 _length) internal pure returns (uint256 _numBlocks) {
        _numBlocks = numBlocks(chunkPtr);

        // should contain at least one block
        require(_numBlocks > 0, "no block in chunk");

        // should contain at least the number of the blocks and block contexts
        require(_length >= 1 + _numBlocks * BLOCK_CONTEXT_LENGTH, "invalid chunk length");
    }

    /// @notice Return the start memory offset of `l2Transactions`.
    /// @dev The caller should make sure `_numBlocks` is correct.
    /// @param chunkPtr The start memory offset of the chunk in memory.
    /// @param _numBlocks The number of blocks in current chunk.
    /// @return _l2TxPtr the start memory offset of `l2Transactions`.
    function l2TxPtr(uint256 chunkPtr, uint256 _numBlocks) internal pure returns (uint256 _l2TxPtr) {
        unchecked {
            _l2TxPtr = chunkPtr + 1 + _numBlocks * BLOCK_CONTEXT_LENGTH;
        }
    }

    /// @notice Return the number of blocks in current chunk.
    /// @param chunkPtr The start memory offset of the chunk in memory.
    /// @return _numBlocks The number of blocks in current chunk.
    function numBlocks(uint256 chunkPtr) internal pure returns (uint256 _numBlocks) {
        assembly {
            _numBlocks := shr(248, mload(chunkPtr))
        }
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
        // only first 58 bytes is needed.
        assembly {
            chunkPtr := add(chunkPtr, add(1, mul(BLOCK_CONTEXT_LENGTH, index)))
            mstore(dstPtr, mload(chunkPtr)) // first 32 bytes
            mstore(
                add(dstPtr, 0x20),
                and(mload(add(chunkPtr, 0x20)), 0xffffffffffffffffffffffffffffffffffffffffffffffffffff000000000000)
            ) // next 26 bytes

            dstPtr := add(dstPtr, 58)
        }

        return dstPtr;
    }

    /// @notice Return the number of transactions in current block.
    /// @param blockPtr The start memory offset of the block context in memory.
    /// @return _numTransactions The number of transactions in current block.
    function numTransactions(uint256 blockPtr) internal pure returns (uint256 _numTransactions) {
        assembly {
            _numTransactions := shr(240, mload(add(blockPtr, 56)))
        }
    }

    /// @notice Return the number of L1 messages in current block.
    /// @param blockPtr The start memory offset of the block context in memory.
    /// @return _numL1Messages The number of L1 messages in current block.
    function numL1Messages(uint256 blockPtr) internal pure returns (uint256 _numL1Messages) {
        assembly {
            _numL1Messages := shr(240, mload(add(blockPtr, 58)))
        }
    }

    /// @notice Compute and load the transaction hash.
    /// @param _l2TxPtr The start memory offset of the transaction in memory.
    /// @return bytes32 The transaction hash of the transaction.
    /// @return uint256 The start memory offset of the next transaction in memory.
    function loadL2TxHash(uint256 _l2TxPtr) internal pure returns (bytes32, uint256) {
        bytes32 txHash;
        assembly {
            // first 4 bytes indicate the length
            let txPayloadLength := shr(224, mload(_l2TxPtr))
            _l2TxPtr := add(_l2TxPtr, 4)
            txHash := keccak256(_l2TxPtr, txPayloadLength)
            _l2TxPtr := add(_l2TxPtr, txPayloadLength)
        }

        return (txHash, _l2TxPtr);
    }
}
