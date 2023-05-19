// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

/// @dev Below is the encoding for `Chunk`, total 60*n+1+m bytes.
/// ```text
///   * Field           Bytes       Type            Index       Comments
///   * numBlocks       1           uint8           0           The number of blocks in this chunk
///   * block[0]        60          BlockContext    1           The first block in this chunk
///   * ......
///   * block[i]        60          BlockContext    60*i+1     The first block in this chunk
///   * ......
///   * block[n-1]      60          BlockContext    60*n-155   The last block in this chunk
///   * l2Transactions  dynamic     bytes           60*n+1
/// ```
///
/// @dev Below is the encoding for `BlockContext`, total 60 bytes.
/// ```text
///   * Field                   Bytes      Type         Index  Comments
///   * blockNumber             8          uint64       0     The height of this block.
///   * timestamp               8          uint64       8     The timestamp of this block.
///   * baseFee                 32         uint256      16     The base fee of this block. Currently, it is always 0, because we disable EIP-1559.
///   * gasLimit                8          uint64       48    The gas limit of this block.
///   * numTransactions         2          uint16       56    The number of transactions in this block, both L1 & L2 txs.
///   * numL1Messages           2          uint16       58    The number of l1 messages in this block.
/// ```
library ChunkCodec {
    uint256 internal constant BLOCK_CONTEXT_LENGTH = 60;

    function validateChunkLength(uint256 chunkPtr, uint256 _length) internal pure returns (uint256 _numBlocks) {
        _numBlocks = numBlocks(chunkPtr);

        // should contain at least one block
        require(_numBlocks > 0, "no block in chunk");

        // should contain at least the number of the blocks and block contexts
        require(_length >= 1 + _numBlocks * BLOCK_CONTEXT_LENGTH, "invalid chunk length");
    }

    function l2TxPtr(uint256 chunkPtr, uint256 _numBlocks) internal pure returns (uint256 _l2TxPtr) {
        unchecked {
            _l2TxPtr = chunkPtr + 1 + _numBlocks * BLOCK_CONTEXT_LENGTH;
        }
    }

    function numBlocks(uint256 chunkPtr) internal pure returns (uint256 _numBlocks) {
        assembly {
            _numBlocks := shr(248, mload(chunkPtr))
        }
    }

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
                and(add(chunkPtr, 0x20), 0xffffffffffffffffffffffffffffffffffffffffffffffffffff000000000000)
            ) // next 26 bytes

            dstPtr := add(dstPtr, 58)
        }

        return dstPtr;
    }

    function numTransactions(uint256 ptr) internal pure returns (uint256 _numTransactions) {
        assembly {
            _numTransactions := shr(240, mload(add(ptr, 56)))
        }
    }

    function numL1Messages(uint256 ptr) internal pure returns (uint256 _numL1Messages) {
        assembly {
            _numL1Messages := shr(240, mload(add(ptr, 58)))
        }
    }

    function loadL2TxHash(uint256 ptr) internal pure returns (bytes32, uint256) {
        bytes32 txHash;
        assembly {
            // first 4 bytes indicate the length
            let txPayloadLength := shr(224, mload(ptr))
            ptr := add(ptr, 4)
            txHash := keccak256(ptr, txPayloadLength)
            ptr := add(ptr, txPayloadLength)
        }

        return (txHash, ptr);
    }
}
