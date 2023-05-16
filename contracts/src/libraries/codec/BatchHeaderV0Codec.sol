// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

/// @dev Below is the encoding for `BatchHeader` V0, total 121 + ceil(l1MessagePopped / 256) * 32 bytes.
/// ```text
///   * Field                   Bytes       Type        Index   Comments
///   * version                 1           uint8       0       The batch version
///   * batchIndex              8           uint64      1       The index of the batch
///   * l1MessagePopped         8           uint64      9       Number of L1 message popped in the batch
///   * totalL1MessagePopped    8           uint64      17      Number of total L1 message popped after the batch
///   * dataHash                32          bytes32     25      The data hash of the batch
///   * lastBlockHash           32          bytes32     57      The block hash of the last block in the batch
///   * parentBatchHash         32          bytes32     89      The parent batch hash
///   * skippedL1MessageBitmap  dynamic     uint256[]   121     A bitmap to indicate if L1 messages are skipped in the batch
/// ```
library BatchHeaderV0Codec {
    function loadAndValidate(bytes calldata _batchHeader) internal pure returns (uint256 memPtr, uint256 length) {
        length = _batchHeader.length;
        require(length >= 121, "batch header length too small");

        // copy batch header to memory.
        assembly {
            memPtr := mload(0x40)
            calldatacopy(memPtr, _batchHeader.offset, length)
            mstore(0x40, add(memPtr, length))
        }

        // check batch header length
        uint256 _l1MessagePopped = BatchHeaderV0Codec.l1MessagePopped(memPtr);

        unchecked {
            require(length == 121 + ((_l1MessagePopped + 255) / 256) * 32, "wrong bitmap length");
        }
    }

    function version(uint256 memPtr) internal pure returns (uint256 _version) {
        assembly {
            _version := shr(248, mload(memPtr))
        }
    }

    function batchIndex(uint256 memPtr) internal pure returns (uint256 _batchIndex) {
        assembly {
            _batchIndex := shr(192, mload(add(memPtr, 1)))
        }
    }

    function l1MessagePopped(uint256 memPtr) internal pure returns (uint256 _l1MessagePopped) {
        assembly {
            _l1MessagePopped := shr(192, mload(add(memPtr, 9)))
        }
    }

    function totalL1MessagePopped(uint256 memPtr) internal pure returns (uint256 _totalL1MessagePopped) {
        assembly {
            _totalL1MessagePopped := shr(192, mload(add(memPtr, 17)))
        }
    }

    function dataHash(uint256 memPtr) internal pure returns (bytes32 _dataHash) {
        assembly {
            _dataHash := mload(add(memPtr, 25))
        }
    }

    function lastBlockHash(uint256 memPtr) internal pure returns (bytes32 _lastBlockHash) {
        assembly {
            _lastBlockHash := mload(add(memPtr, 57))
        }
    }

    function parentBatchHash(uint256 memPtr) internal pure returns (bytes32 _parentBatchHash) {
        assembly {
            _parentBatchHash := mload(add(memPtr, 89))
        }
    }

    function storeVersion(uint256 memPtr, uint256 _version) internal pure {
        assembly {
            mstore(memPtr, shl(248, _version))
        }
    }

    function storeBatchIndex(uint256 memPtr, uint256 _batchIndex) internal pure {
        assembly {
            mstore(add(memPtr, 1), shl(192, _batchIndex))
        }
    }

    function storeL1MessagePopped(uint256 memPtr, uint256 _l1MessagePopped) internal pure {
        assembly {
            mstore(add(memPtr, 9), shl(192, _l1MessagePopped))
        }
    }

    function storeTotalL1MessagePopped(uint256 memPtr, uint256 _totalL1MessagePopped) internal pure {
        assembly {
            mstore(add(memPtr, 17), shl(192, _totalL1MessagePopped))
        }
    }

    function storeDataHash(uint256 memPtr, bytes32 _dataHash) internal pure {
        assembly {
            mstore(add(memPtr, 25), _dataHash)
        }
    }

    function storeLastBlockHash(uint256 memPtr, bytes32 _lastBlockHash) internal pure {
        assembly {
            mstore(add(memPtr, 57), _lastBlockHash)
        }
    }

    function storeParentBatchHash(uint256 memPtr, bytes32 _parentBatchHash) internal pure {
        assembly {
            mstore(add(memPtr, 89), _parentBatchHash)
        }
    }

    function storeBitMap(uint256 memPtr, uint256 bitmapPtr) internal pure {
        uint256 _l1MessagePopped = l1MessagePopped(memPtr);

        assembly {
            memPtr := add(memPtr, 121)
            for {
                let i := 0
            } lt(i, _l1MessagePopped) {
                i := add(i, 256)
            } {
                mstore(memPtr, bitmapPtr)
                memPtr := add(memPtr, 0x20)
                bitmapPtr := add(bitmapPtr, 0x20)
            }
        }
    }
}
