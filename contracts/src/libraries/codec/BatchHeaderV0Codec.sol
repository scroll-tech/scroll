// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

// solhint-disable no-inline-assembly

/// @dev Below is the encoding for `BatchHeader` V0, total 89 + ceil(l1MessagePopped / 256) * 32 bytes.
/// ```text
///   * Field                   Bytes       Type        Index   Comments
///   * version                 1           uint8       0       The batch version
///   * batchIndex              8           uint64      1       The index of the batch
///   * l1MessagePopped         8           uint64      9       Number of L1 messages popped in the batch
///   * totalL1MessagePopped    8           uint64      17      Number of total L1 message popped after the batch
///   * dataHash                32          bytes32     25      The data hash of the batch
///   * parentBatchHash         32          bytes32     57      The parent batch hash
///   * skippedL1MessageBitmap  dynamic     uint256[]   89      A bitmap to indicate which L1 messages are skipped in the batch
/// ```
library BatchHeaderV0Codec {
    /// @notice Load batch header in calldata to memory.
    /// @param _batchHeader The encoded batch header bytes in calldata.
    /// @return batchPtr The start memory offset of the batch header in memory.
    /// @return length The length in bytes of the batch header.
    function loadAndValidate(bytes calldata _batchHeader) internal pure returns (uint256 batchPtr, uint256 length) {
        length = _batchHeader.length;
        require(length >= 89, "batch header length too small");

        // copy batch header to memory.
        assembly {
            batchPtr := mload(0x40)
            calldatacopy(batchPtr, _batchHeader.offset, length)
            mstore(0x40, add(batchPtr, length))
        }

        // check batch header length
        uint256 _l1MessagePopped = BatchHeaderV0Codec.l1MessagePopped(batchPtr);

        unchecked {
            require(length == 89 + ((_l1MessagePopped + 255) / 256) * 32, "wrong bitmap length");
        }
    }

    /// @notice Get the version of the batch header.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @return _version The version of the batch header.
    function version(uint256 batchPtr) internal pure returns (uint256 _version) {
        assembly {
            _version := shr(248, mload(batchPtr))
        }
    }

    /// @notice Get the batch index of the batch.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @return _batchIndex The batch index of the batch.
    function batchIndex(uint256 batchPtr) internal pure returns (uint256 _batchIndex) {
        assembly {
            _batchIndex := shr(192, mload(add(batchPtr, 1)))
        }
    }

    /// @notice Get the number of L1 messages of the batch.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @return _l1MessagePopped The number of L1 messages of the batch.
    function l1MessagePopped(uint256 batchPtr) internal pure returns (uint256 _l1MessagePopped) {
        assembly {
            _l1MessagePopped := shr(192, mload(add(batchPtr, 9)))
        }
    }

    /// @notice Get the number of L1 messages popped before this batch.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @return _totalL1MessagePopped The the number of L1 messages popped before this batch.
    function totalL1MessagePopped(uint256 batchPtr) internal pure returns (uint256 _totalL1MessagePopped) {
        assembly {
            _totalL1MessagePopped := shr(192, mload(add(batchPtr, 17)))
        }
    }

    /// @notice Get the data hash of the batch header.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @return _dataHash The data hash of the batch header.
    function dataHash(uint256 batchPtr) internal pure returns (bytes32 _dataHash) {
        assembly {
            _dataHash := mload(add(batchPtr, 25))
        }
    }

    /// @notice Get the parent batch hash of the batch header.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @return _parentBatchHash The parent batch hash of the batch header.
    function parentBatchHash(uint256 batchPtr) internal pure returns (bytes32 _parentBatchHash) {
        assembly {
            _parentBatchHash := mload(add(batchPtr, 57))
        }
    }

    /// @notice Get the skipped L1 messages bitmap.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @param index The index of bitmap to load.
    /// @return _bitmap The bitmap from bits `index * 256` to `index * 256 + 255`.
    function skippedBitmap(uint256 batchPtr, uint256 index) internal pure returns (uint256 _bitmap) {
        assembly {
            batchPtr := add(batchPtr, 89)
            _bitmap := mload(add(batchPtr, mul(index, 32)))
        }
    }

    /// @notice Store the version of batch header.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @param _version The version of batch header.
    function storeVersion(uint256 batchPtr, uint256 _version) internal pure {
        assembly {
            mstore8(batchPtr, _version)
        }
    }

    /// @notice Store the batch index of batch header.
    /// @dev Because this function can overwrite the subsequent fields, it must be called before
    /// `storeL1MessagePopped`, `storeTotalL1MessagePopped`, and `storeDataHash`.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @param _batchIndex The batch index.
    function storeBatchIndex(uint256 batchPtr, uint256 _batchIndex) internal pure {
        assembly {
            mstore(add(batchPtr, 1), shl(192, _batchIndex))
        }
    }

    /// @notice Store the number of L1 messages popped in current batch to batch header.
    /// @dev Because this function can overwrite the subsequent fields, it must be called before
    /// `storeTotalL1MessagePopped` and `storeDataHash`.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @param _l1MessagePopped The number of L1 messages popped in current batch.
    function storeL1MessagePopped(uint256 batchPtr, uint256 _l1MessagePopped) internal pure {
        assembly {
            mstore(add(batchPtr, 9), shl(192, _l1MessagePopped))
        }
    }

    /// @notice Store the total number of L1 messages popped after current batch to batch header.
    /// @dev Because this function can overwrite the subsequent fields, it must be called before
    /// `storeDataHash`.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @param _totalL1MessagePopped The total number of L1 messages popped after current batch.
    function storeTotalL1MessagePopped(uint256 batchPtr, uint256 _totalL1MessagePopped) internal pure {
        assembly {
            mstore(add(batchPtr, 17), shl(192, _totalL1MessagePopped))
        }
    }

    /// @notice Store the data hash of batch header.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @param _dataHash The data hash.
    function storeDataHash(uint256 batchPtr, bytes32 _dataHash) internal pure {
        assembly {
            mstore(add(batchPtr, 25), _dataHash)
        }
    }

    /// @notice Store the parent batch hash of batch header.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @param _parentBatchHash The parent batch hash.
    function storeParentBatchHash(uint256 batchPtr, bytes32 _parentBatchHash) internal pure {
        assembly {
            mstore(add(batchPtr, 57), _parentBatchHash)
        }
    }

    /// @notice Store the skipped L1 message bitmap of batch header.
    /// @param batchPtr The start memory offset of the batch header in memory.
    /// @param _skippedL1MessageBitmap The skipped L1 message bitmap.
    function storeSkippedBitmap(uint256 batchPtr, bytes calldata _skippedL1MessageBitmap) internal pure {
        assembly {
            calldatacopy(add(batchPtr, 89), _skippedL1MessageBitmap.offset, _skippedL1MessageBitmap.length)
        }
    }

    /// @notice Compute the batch hash.
    /// @dev Caller should make sure that the encoded batch header is correct.
    ///
    /// @param batchPtr The memory offset of the encoded batch header.
    /// @param length The length of the batch.
    /// @return _batchHash The hash of the corresponding batch.
    function computeBatchHash(uint256 batchPtr, uint256 length) internal pure returns (bytes32 _batchHash) {
        // in the current version, the hash is: keccak(BatchHeader without timestamp)
        assembly {
            _batchHash := keccak256(batchPtr, length)
        }
    }
}
