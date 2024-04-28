// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {IL1Blocks} from "./IL1Blocks.sol";

/// @title L1Blocks
/// @notice This contract will maintain the list of blocks proposed in L1.
contract L1Blocks is IL1Blocks {
    /***********
     * Structs *
     ***********/

    struct BlockFields {
        // The block number
        uint256 number;
        // The block timestamp
        uint256 timestamp;
        // The block hash
        bytes32 blockHash;
        // The state root
        bytes32 stateRoot;
        // The randao value
        bytes32 Randao;
        // The base fee
        uint256 baseFee;
        // The blob base fee
        uint256 blobBaseFee;
        // The parent beacon block root
        bytes32 parentBeaconRoot;
    }

    /*************
     * Variables *
     *************/

    address public constant SYSTEM_SENDER = 0xffffFFFfFFffffffffffffffFfFFFfffFFFfFFfE;

    uint256 private constant LONDON_FORK_NUMBER_MINUS_ONE = 12964999;

    uint256 private constant CANCUN_FORK_NUMBER_MINUS_ONE = 19426586;

    uint256 private constant MIN_BASE_FEE_PER_BLOB_GAS = 1;

    uint256 private constant BLOB_BASE_FEE_UPDATE_FRACTION = 3338477;

    uint256 public constant BLOCK_BUFFER_SIZE = 8192;

    uint256 public constant BLOCK_FIELDS_BYTES = 256;

    /// @notice Storage slot with the address of the current block hashes offset.
    /// @dev This is the keccak-256 hash of "l1blocks.block_storage_offset" with 256-bit alignment
    uint256 private constant BLOCK_STORAGE_OFFSET = 0xdb384d0440765c9be19ada21c3d61f9d220d57e5963a1fca370403ac2c4bbb00;

    /// @inheritdoc IL1Blocks
    uint256 public override latestBlockNumber;

    /*************
     * Modifiers *
     *************/

    modifier validBlockNumber(uint256 blockNumber) {
        uint256 _latestBlockNumber = latestBlockNumber;
        require(
            blockNumber <= _latestBlockNumber && blockNumber > _latestBlockNumber - BLOCK_BUFFER_SIZE,
            "invalid block number"
        );
        _;
    }

    modifier onlySystem() {
        if (msg.sender != SYSTEM_SENDER) revert("only system sender allowed");
        _;
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL1Blocks
    function latestBlockHash() external view returns (bytes32) {
        return _getBlockFields(latestBlockNumber, false).blockHash;
    }

    /// @inheritdoc IL1Blocks
    function getBlockHash(uint256 blockNumber) external view validBlockNumber(blockNumber) returns (bytes32) {
        return _getBlockFields(blockNumber, true).blockHash;
    }

    /// @inheritdoc IL1Blocks
    function latestStateRoot() external view returns (bytes32) {
        return _getBlockFields(latestBlockNumber, false).stateRoot;
    }

    /// @inheritdoc IL1Blocks
    function getStateRoot(uint256 blockNumber) external view validBlockNumber(blockNumber) returns (bytes32) {
        return _getBlockFields(blockNumber, true).stateRoot;
    }

    /// @inheritdoc IL1Blocks
    function latestBlockTimestamp() external view returns (uint256) {
        return _getBlockFields(latestBlockNumber, false).timestamp;
    }

    /// @inheritdoc IL1Blocks
    function getBlockTimestamp(uint256 blockNumber) external view validBlockNumber(blockNumber) returns (uint256) {
        return _getBlockFields(blockNumber, true).timestamp;
    }

    /// @inheritdoc IL1Blocks
    function latestBaseFee() external view returns (uint256) {
        return _getBlockFields(latestBlockNumber, false).baseFee;
    }

    /// @inheritdoc IL1Blocks
    function getBaseFee(uint256 blockNumber) external view validBlockNumber(blockNumber) returns (uint256) {
        return _getBlockFields(blockNumber, true).baseFee;
    }

    /// @inheritdoc IL1Blocks
    function latestBlobBaseFee() external view returns (uint256) {
        return _getBlockFields(latestBlockNumber, false).blobBaseFee;
    }

    /// @inheritdoc IL1Blocks
    function getBlobBaseFee(uint256 blockNumber) external view validBlockNumber(blockNumber) returns (uint256) {
        return _getBlockFields(blockNumber, true).blobBaseFee;
    }

    /// @inheritdoc IL1Blocks
    function latestParentBeaconRoot() external view returns (bytes32) {
        return _getBlockFields(latestBlockNumber, false).parentBeaconRoot;
    }

    /// @inheritdoc IL1Blocks
    function getParentBeaconRoot(uint256 blockNumber) external view validBlockNumber(blockNumber) returns (bytes32) {
        return _getBlockFields(blockNumber, true).parentBeaconRoot;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL1Blocks
    /// @dev The encoding order in block header is
    /// ```text
    /// | Field            |  Bytes  |          Notes          |
    /// | ParentHash       |      32 |                required |
    /// | UncleHash        |      32 |                required |
    /// | Coinbase         |      20 |                required |
    /// | StateRoot        |      32 |                required |
    /// | TransactionsRoot |      32 |                required |
    /// | ReceiptsRoot     |      32 |                required |
    /// | LogsBloom        |     256 |                required |
    /// | Difficulty       |      32 |                required |
    /// | BlockNumber      |      32 |                required |
    /// | GasLimit         |       8 |                required |
    /// | GasUsed          |       8 |                required |
    /// | BlockTimestamp   |       8 |                required |
    /// | ExtraData        | dynamic |                required |
    /// | MixHash/Randao   |      32 |                required |
    /// | BlockNonce       |       8 |                required |
    /// | BaseFee          |      32 | optional after 12965000 |
    /// | WithdrawalsHash  |      32 | optional after 19426587 |
    /// | BlobGasUsed      |       8 | optional after 19426587 |
    /// | ExcessBlobGas    |       8 | optional after 19426587 |
    /// | ParentBeaconRoot |      32 | optional after 19426587 |
    /// ```
    function setL1BlockHeader(bytes calldata blockHeaderRlp) external onlySystem returns (bytes32 blockHash) {
        // Block fields byte
        // |   Bytes   |          Field           |
        // ------------|--------------------------|
        // | [0:31]    | block number             |
        // | [32:63]   | timestamp                |
        // | [64:95]   | block hash               |
        // | [96:127]  | state root               |
        // | [128:150] | randao                   |
        // | [160:191] | base fee                 |
        // | [192:223] | blob base fee            |
        // | [224:255] | parent beacon block root |
        BlockFields memory b;
        bytes32 parentHash;
        assembly {
            // reverts with error `msg`.
            // make sure the length of error string <= 32
            function revertWith(msg) {
                // keccak("Error(string)")
                mstore(0x00, shl(224, 0x08c379a0))
                mstore(0x04, 0x20) // str.offset
                mstore(0x44, msg)
                let msgLen
                for {

                } msg {

                } {
                    msg := shl(8, msg)
                    msgLen := add(msgLen, 1)
                }
                mstore(0x24, msgLen) // str.length
                revert(0x00, 0x64)
            }
            // reverts with `msg` when condition is not matched.
            // make sure the length of error string <= 32
            function require(cond, msg) {
                if iszero(cond) {
                    revertWith(msg)
                }
            }
            // returns the calldata offset of the value and the length in bytes
            // for the RLP encoded data item at `ptr`. used in `decodeFlat`
            function decodeValue(ptr) -> dataLen, valueOffset {
                let b0 := byte(0, calldataload(ptr))
                // 0x00 - 0x7f, single byte
                if lt(b0, 0x80) {
                    // for a single byte whose value is in the [0x00, 0x7f] range,
                    // that byte is its own RLP encoding.
                    dataLen := 1
                    valueOffset := ptr
                    leave
                }
                // 0x80 - 0xb7, short string/bytes, length <= 55
                if lt(b0, 0xb8) {
                    // the RLP encoding consists of a single byte with value 0x80
                    // plus the length of the string followed by the string.
                    dataLen := sub(b0, 0x80)
                    valueOffset := add(ptr, 1)
                    leave
                }
                // 0xb8 - 0xbf, long string/bytes, length > 55
                if lt(b0, 0xc0) {
                    // the RLP encoding consists of a single byte with value 0xb7
                    // plus the length in bytes of the length of the string in binary form,
                    // followed by the length of the string, followed by the string.
                    let lengthBytes := sub(b0, 0xb7)
                    if gt(lengthBytes, 4) {
                        invalid()
                    }

                    // load the extended length
                    valueOffset := add(ptr, 1)
                    let extendedLen := calldataload(valueOffset)
                    let bits := sub(256, mul(lengthBytes, 8))
                    extendedLen := shr(bits, extendedLen)

                    dataLen := extendedLen
                    valueOffset := add(valueOffset, lengthBytes)
                    leave
                }
                revertWith("Not value")
            }
            function loadAndCacheValue(memPtr, _ptr) -> ptr {
                ptr := _ptr
                let len, offset := decodeValue(ptr)
                // the value we care must have at most 32 bytes
                if lt(len, 33) {
                    let bits := mul(sub(32, len), 8)
                    let value := calldataload(offset)
                    value := shr(bits, value)
                    mstore(memPtr, value)
                }
                ptr := add(len, offset)
            }

            let ptr := blockHeaderRlp.offset
            let headerPayloadLength
            {
                let b0 := byte(0, calldataload(ptr))
                // the input should be a long list
                if lt(b0, 0xf8) {
                    invalid()
                }
                let lengthBytes := sub(b0, 0xf7)
                if gt(lengthBytes, 32) {
                    invalid()
                }
                // load the extended length
                ptr := add(ptr, 1)
                headerPayloadLength := calldataload(ptr)
                let bits := sub(256, mul(lengthBytes, 8))
                // compute payload length: extended length + length bytes + 1
                headerPayloadLength := shr(bits, headerPayloadLength)
                headerPayloadLength := add(headerPayloadLength, lengthBytes)
                headerPayloadLength := add(headerPayloadLength, 1)
                ptr := add(ptr, lengthBytes)
            }

            let memPtr := mload(0x40)
            let blockNumber
            calldatacopy(memPtr, blockHeaderRlp.offset, headerPayloadLength)
            blockHash := keccak256(memPtr, headerPayloadLength)

            // load 15 values
            for {
                let i := 0
            } lt(i, 15) {
                i := add(i, 1)
            } {
                ptr := loadAndCacheValue(memPtr, ptr)
                // load BlockNumber, 8-th entry in `blockHeaderRlp`
                if eq(i, 8) {
                    blockNumber := mload(memPtr)
                }
                memPtr := add(memPtr, 0x20)
            }
            // load optional fields after london fork
            if gt(blockNumber, LONDON_FORK_NUMBER_MINUS_ONE) {
                ptr := loadAndCacheValue(memPtr, ptr)
                memPtr := add(memPtr, 0x20)
            }
            // load optional fields after cancun fork
            if gt(blockNumber, CANCUN_FORK_NUMBER_MINUS_ONE) {
                for {
                    let i := 0
                } lt(i, 4) {
                    i := add(i, 1)
                } {
                    ptr := loadAndCacheValue(memPtr, ptr)
                    memPtr := add(memPtr, 0x20)
                }
            }
            require(eq(ptr, add(blockHeaderRlp.offset, blockHeaderRlp.length)), "Header RLP length mismatch")

            memPtr := mload(0x40)
            // load ParentHash, 0-th entry in `blockHeaderRlp`
            parentHash := mload(memPtr)
            mstore(b, blockNumber)
            // load BlockTimestamp, 11-th entry in `blockHeaderRlp`
            mstore(add(b, 0x20), mload(add(memPtr, 0x160))) // 0x20 * 11
            // load StateRoot, 3-th entry in `blockHeaderRlp`
            mstore(add(b, 0x60), mload(add(memPtr, 0x60))) // 0x20 * 3
            // load Randao, 13-th entry in `blockHeaderRlp`
            mstore(add(b, 0x80), mload(add(memPtr, 0x1a0))) // 0x20 * 13
            switch gt(blockNumber, LONDON_FORK_NUMBER_MINUS_ONE)
            case 1 {
                // load BaseFee, 15-th entry in `blockHeaderRlp`
                mstore(add(b, 0xa0), mload(add(memPtr, 0x1e0))) // 0x20 * 15
            }
            default {
                mstore(add(b, 0xa0), 0)
            }
            if gt(blockNumber, CANCUN_FORK_NUMBER_MINUS_ONE) {
                // load ExcessBlobGas, 18-th entry in `blockHeaderRlp`
                mstore(add(b, 0xc0), mload(add(memPtr, 0x240))) // 0x20 * 18
                // load ParentBeaconRoot, 19-th entry in `blockHeaderRlp`
                mstore(add(b, 0xe0), mload(add(memPtr, 0x260))) // 0x20 * 19
            }
        }
        b.blockHash = blockHash;
        if (b.number > CANCUN_FORK_NUMBER_MINUS_ONE) {
            b.blobBaseFee = _exp(MIN_BASE_FEE_PER_BLOB_GAS, b.blobBaseFee, BLOB_BASE_FEE_UPDATE_FRACTION);
        }

        uint256 _latestBlockNumber = latestBlockNumber;
        // validate fields when not first block.
        if (_latestBlockNumber != 0) {
            if (_latestBlockNumber + 1 != b.number) revert();
            BlockFields storage s = _getBlockFields(_latestBlockNumber, false);
            if (s.blockHash != parentHash) revert();
        }

        latestBlockNumber = b.number;
        _setBlockFields(b);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to return the `BlockFields` storage for the given `blockNumber`.
    /// @param blockNumber The block number to load.
    /// @param validateBlockNumber Whether to check the block number.
    function _getBlockFields(
        uint256 blockNumber,
        bool validateBlockNumber
    ) private view returns (BlockFields storage b) {
        uint256 slot = BLOCK_STORAGE_OFFSET + (blockNumber % BLOCK_BUFFER_SIZE) * BLOCK_FIELDS_BYTES;
        assembly {
            b.slot := slot
        }
        if (validateBlockNumber && b.number != blockNumber) {
            revert ErrorBlockUnavailable();
        }
    }

    /// @dev Internal function to update the `BlockFields`.
    function _setBlockFields(BlockFields memory b) private {
        BlockFields storage s = _getBlockFields(b.number, false);
        s.number = b.number;
        s.timestamp = b.timestamp;
        s.blockHash = b.blockHash;
        s.stateRoot = b.stateRoot;
        s.Randao = b.Randao;
        s.baseFee = b.baseFee;
        s.blobBaseFee = b.blobBaseFee;
        s.parentBeaconRoot = b.parentBeaconRoot;
    }

    /// @dev Approximates factor * e ** (numerator / denominator) using Taylor expansion:
    /// based on `fake_exponential` in https://eips.ethereum.org/EIPS/eip-4844
    function _exp(uint256 factor, uint256 numerator, uint256 denominator) private pure returns (uint256) {
        uint256 output;
        uint256 numerator_accum = factor * denominator;
        for (uint256 i = 1; numerator_accum > 0; i++) {
            output += numerator_accum;
            numerator_accum = (numerator_accum * numerator) / (denominator * i);
        }
        return output / denominator;
    }
}
