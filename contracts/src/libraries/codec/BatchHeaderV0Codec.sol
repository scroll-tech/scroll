// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

/// @dev Below is the encoding for `BatchHeader` V0, total 89 + ceil(l1MessagePopped / 256) * 32 bytes.
/// ```text
///   * Field                   Bytes       Type        Index   Comments
///   * version                 1           uint8       0       The batch version
///   * batchIndex              8           uint64      1       The index of the batch
///   * l1MessagePopped         8           uint64      9       Number of L1 message popped in the batch
///   * totalL1MessagePopped    8           uint64      17      Number of total L1 message popped after the batch
///   * dataHash                32          bytes32     25      The data hash of the batch
///   * parentBatchHash         32          bytes32     57      The parent batch hash
///   * skippedL1MessageBitmap  dynamic     uint256[]   89     A bitmap to indicate if L1 messages are skipped in the batch
/// ```
library BatchHeaderV0Codec {
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

    function version(uint256 batchPtr) internal pure returns (uint256 _version) {
        assembly {
            _version := shr(248, mload(batchPtr))
        }
    }

    function batchIndex(uint256 batchPtr) internal pure returns (uint256 _batchIndex) {
        assembly {
            _batchIndex := shr(192, mload(add(batchPtr, 1)))
        }
    }

    function l1MessagePopped(uint256 batchPtr) internal pure returns (uint256 _l1MessagePopped) {
        assembly {
            _l1MessagePopped := shr(192, mload(add(batchPtr, 9)))
        }
    }

    function totalL1MessagePopped(uint256 batchPtr) internal pure returns (uint256 _totalL1MessagePopped) {
        assembly {
            _totalL1MessagePopped := shr(192, mload(add(batchPtr, 17)))
        }
    }

    function dataHash(uint256 batchPtr) internal pure returns (bytes32 _dataHash) {
        assembly {
            _dataHash := mload(add(batchPtr, 25))
        }
    }

    function parentBatchHash(uint256 batchPtr) internal pure returns (bytes32 _parentBatchHash) {
        assembly {
            _parentBatchHash := mload(add(batchPtr, 57))
        }
    }

    function storeVersion(uint256 batchPtr, uint256 _version) internal pure {
        assembly {
            mstore(batchPtr, shl(248, _version))
        }
    }

    function storeBatchIndex(uint256 batchPtr, uint256 _batchIndex) internal pure {
        assembly {
            mstore(add(batchPtr, 1), shl(192, _batchIndex))
        }
    }

    function storeL1MessagePopped(uint256 batchPtr, uint256 _l1MessagePopped) internal pure {
        assembly {
            mstore(add(batchPtr, 9), shl(192, _l1MessagePopped))
        }
    }

    function storeTotalL1MessagePopped(uint256 batchPtr, uint256 _totalL1MessagePopped) internal pure {
        assembly {
            mstore(add(batchPtr, 17), shl(192, _totalL1MessagePopped))
        }
    }

    function storeDataHash(uint256 batchPtr, bytes32 _dataHash) internal pure {
        assembly {
            mstore(add(batchPtr, 25), _dataHash)
        }
    }

    function storeParentBatchHash(uint256 batchPtr, bytes32 _parentBatchHash) internal pure {
        assembly {
            mstore(add(batchPtr, 57), _parentBatchHash)
        }
    }

    function storeBitMap(uint256 batchPtr, bytes calldata _skippedL1MessageBitmap) internal pure {
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
        // in current version, the hash is: keccak(BatchHeader without timestamp)
        assembly {
            _batchHash := keccak256(batchPtr, length)
        }
    }
}
