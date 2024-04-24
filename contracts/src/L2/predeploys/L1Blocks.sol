// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {IL1Blocks} from "./IL1Blocks.sol";
import {IL1GasPriceOracle} from "./IL1GasPriceOracle.sol";

import {OwnableBase} from "../../libraries/common/OwnableBase.sol";
import {IWhitelist} from "../../libraries/common/IWhitelist.sol";
import {ScrollPredeploy} from "../../libraries/constants/ScrollPredeploy.sol";

/// @title L1Blocks
/// @notice This contract will maintain the list of blocks proposed in L1.
contract L1Blocks is OwnableBase, IL1Blocks {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates whitelist contract.
    /// @param _oldWhitelist The address of old whitelist contract.
    /// @param _newWhitelist The address of new whitelist contract.
    event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

    /***********
     * Structs *
     ***********/

    /// @dev Compiler will pack this into single `uint256`.
    struct BlockFields {
        // The block number
        uint64 number;
        // The block timestamp
        uint64 timestamp;
        // The base fee
        uint64 baseFee;
        // The blob base fee
        uint64 blobBaseFee;
        // The block hash
        bytes32 blockHash;
        // The state root
        bytes32 stateRoot;
        // The parent beacon block root
        bytes32 parentBeaconRoot;
        // The randao value
        bytes32 Randao;
    }

    // Block fields byte
    // |   Bytes   |          Field           |
    // ------------|--------------------------|
    // | [0:7]     | block number             |
    // | [8:15]    | timestamp                |
    // | [16:23]   | base fee                 |
    // | [24:31]   | blob base fee            |
    // | [32:63]   | block hash               |
    // | [64:95]   | state root               |
    // | [96:127]  | parent beacon block root |
    // | [128:160] | randao                   |

    /*************
     * Variables *
     *************/

    address public constant SYSTEM_SENDER = 0xffffFFFfFFffffffffffffffFfFFFfffFFFfFFfE;

    uint32 public constant BLOCK_BUFFER_SIZE = 8192;

    uint32 public constant BLOCK_FIELDS_BYTES = 160;

    /// @notice Storage slot with the address of the current block hashes offset.
    /// @dev This is the keccak-256 hash of "l1blocks.block_storage_offset" with 256-bit alignment
    uint256 private constant BLOCK_STORAGE_OFFSET = 0xdb384d0440765c9be19ada21c3d61f9d220d57e5963a1fca370403ac2c4bbb00;

    /// @inheritdoc IL1Blocks
    uint64 public override latestBlockNumber;

    /***************
     * Constructor *
     ***************/

    // function initialize(
    //     bytes32 _startBlockHash,
    //     uint64 _startBlockHeight,
    //     uint64 _startBlockTimestamp,
    //     uint128 _startBlockBaseFee,
    //     bytes32 _startStateRoot
    // ) external onlyOwner {
    //     require(latestBlockHash == bytes32(0), "already initialized");

    //     latestBlockHash = _startBlockHash;
    //     stateRoot[_startBlockHash] = _startStateRoot;
    //     metadata[_startBlockHash] = BlockMetadata(_startBlockHeight, _startBlockTimestamp, _startBlockBaseFee);

    //     emit ImportBlock(_startBlockHash, _startBlockHeight, _startBlockTimestamp, _startBlockBaseFee, _startStateRoot);
    // }

    /*************************
     * Public View Functions *
     *************************/
    modifier validBlockNumber(uint64 blockNumber) {
        uint64 _latestBlockNumber = latestBlockNumber;
        require(
            blockNumber <= _latestBlockNumber && blockNumber > _latestBlockNumber - BLOCK_BUFFER_SIZE,
            "invalid block number"
        );
        _;
    }

    /// @inheritdoc IL1Blocks
    function latestBlockHash() external view returns (bytes32) {
        uint256 blockPtr = BLOCK_STORAGE_OFFSET + (latestBlockNumber % BLOCK_BUFFER_SIZE) * BLOCK_FIELDS_BYTES;
        return _loadBlockHash(blockPtr);
    }

    /// @inheritdoc IL1Blocks
    function getBlockHash(uint64 blockNumber) external view validBlockNumber(blockNumber) returns (bytes32) {
        uint256 blockPtr = BLOCK_STORAGE_OFFSET + (blockNumber % BLOCK_BUFFER_SIZE) * BLOCK_FIELDS_BYTES;
        uint64 _blockNumber = _loadBlockNumber(blockPtr);
        if (_blockNumber != blockNumber) {
            revert ErrorBlockUnavailable();
        }
        return _loadBlockHash(blockPtr);
    }

    /// @inheritdoc IL1Blocks
    function latestStateRoot() external view returns (bytes32) {
        uint256 blockPtr = BLOCK_STORAGE_OFFSET + (latestBlockNumber % BLOCK_BUFFER_SIZE) * BLOCK_FIELDS_BYTES;
        return _loadStateRoot(blockPtr);
    }

    /// @inheritdoc IL1Blocks
    function getStateRoot(uint64 blockNumber) external view validBlockNumber(blockNumber) returns (bytes32) {
        uint256 blockPtr = BLOCK_STORAGE_OFFSET + (blockNumber % BLOCK_BUFFER_SIZE) * BLOCK_FIELDS_BYTES;
        uint64 _blockNumber = _loadBlockNumber(blockPtr);
        if (_blockNumber != blockNumber) {
            revert ErrorBlockUnavailable();
        }
        return _loadStateRoot(blockPtr);
    }

    /// @inheritdoc IL1Blocks
    function latestBlockTimestamp() external view returns (uint256) {
        uint256 blockPtr = BLOCK_STORAGE_OFFSET + (latestBlockNumber % BLOCK_BUFFER_SIZE) * BLOCK_FIELDS_BYTES;
        return _loadTimestamp(blockPtr);
    }

    /// @inheritdoc IL1Blocks
    function getBlockTimestamp(uint64 blockNumber) external view validBlockNumber(blockNumber) returns (uint256) {
        uint256 blockPtr = BLOCK_STORAGE_OFFSET + (blockNumber % BLOCK_BUFFER_SIZE) * BLOCK_FIELDS_BYTES;
        uint64 _blockNumber = _loadBlockNumber(blockPtr);
        if (_blockNumber != blockNumber) {
            revert ErrorBlockUnavailable();
        }
        return _loadTimestamp(blockPtr);
    }

    /// @inheritdoc IL1Blocks
    function latestBaseFee() external view returns (uint256) {
        uint256 blockPtr = BLOCK_STORAGE_OFFSET + (latestBlockNumber % BLOCK_BUFFER_SIZE) * BLOCK_FIELDS_BYTES;
        return _loadBaseFee(blockPtr);
    }

    /// @inheritdoc IL1Blocks
    function getBaseFee(uint64 blockNumber) external view validBlockNumber(blockNumber) returns (uint256) {
        uint256 blockPtr = BLOCK_STORAGE_OFFSET + (blockNumber % BLOCK_BUFFER_SIZE) * BLOCK_FIELDS_BYTES;
        uint64 _blockNumber = _loadBlockNumber(blockPtr);
        if (_blockNumber != blockNumber) {
            revert ErrorBlockUnavailable();
        }
        return _loadBaseFee(blockPtr);
    }

    /// @inheritdoc IL1Blocks
    function latestBlobBaseFee() external view returns (uint256) {
        uint256 blockPtr = BLOCK_STORAGE_OFFSET + (latestBlockNumber % BLOCK_BUFFER_SIZE) * BLOCK_FIELDS_BYTES;
        return _loadBlobBaseFee(blockPtr);
    }

    /// @inheritdoc IL1Blocks
    function getBlobBaseFee(uint64 blockNumber) external view validBlockNumber(blockNumber) returns (uint256) {
        uint256 blockPtr = BLOCK_STORAGE_OFFSET + (blockNumber % BLOCK_BUFFER_SIZE) * BLOCK_FIELDS_BYTES;
        uint64 _blockNumber = _loadBlockNumber(blockPtr);
        if (_blockNumber != blockNumber) {
            revert ErrorBlockUnavailable();
        }
        return _loadBlobBaseFee(blockPtr);
    }

    /// @inheritdoc IL1Blocks
    function latestParentBeaconRoot() external view returns (bytes32) {
        uint256 blockPtr = BLOCK_STORAGE_OFFSET + (latestBlockNumber % BLOCK_BUFFER_SIZE) * BLOCK_FIELDS_BYTES;
        return _loadParentBeaconRoot(blockPtr);
    }

    /// @inheritdoc IL1Blocks
    function getParentBeaconRoot(uint64 blockNumber) external view validBlockNumber(blockNumber) returns (bytes32) {
        uint256 blockPtr = BLOCK_STORAGE_OFFSET + (blockNumber % BLOCK_BUFFER_SIZE) * BLOCK_FIELDS_BYTES;
        uint64 _blockNumber = _loadBlockNumber(blockPtr);
        if (_blockNumber != blockNumber) {
            revert ErrorBlockUnavailable();
        }
        return _loadParentBeaconRoot(blockPtr);
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL1Blocks
    function setL1BlockHeader(bytes32 blockHash, bytes calldata blockHeaderRlp) external {
        require(msg.sender == SYSTEM_SENDER, "only system sender allowed");

        // The encoding order in block header is
        // 1. ParentHash: 32 bytes
        // 2. UncleHash: 32 bytes
        // 3. Coinbase: 20 bytes
        // 4. StateRoot: 32 bytes
        // 5. TransactionsRoot: 32 bytes
        // 6. ReceiptsRoot: 32 bytes
        // 7. LogsBloom: 256 bytes
        // 8. Difficulty: uint256
        // 9. BlockNumber: uint256
        // 10. GasLimit: uint64
        // 11. GasUsed: uint64
        // 12. BlockTimestamp: uint64
        // 13. ExtraData: variable bytes
        // 14. MixHash/Randao: 32 bytes
        // 15. BlockNonce: 8 bytes
        // 16. BaseFee: uint256 // optional
        // 17. WithdrawalsHash: 32 bytes // optional
        // 18. BlobGasUsed: uint64 // optional
        // 19. ExcessBlobGas: uint64 // optional
        // 20. ParentBeaconRoot: 32 bytes // optional
    }

    /**********************
     * Internal Functions *
     **********************/
    function _loadBlockNumber(uint256 blockPtr) internal pure returns (uint64 _blockNumber) {
        assembly {
            _blockNumber := shr(192, mload(blockPtr))
        }
    }

    function _loadTimestamp(uint256 blockPtr) internal pure returns (uint256 _timestamp) {
        assembly {
            _timestamp := and(shr(128, mload(blockPtr)), 0xffffffffffffffff)
        }
    }

    function _loadBaseFee(uint256 blockPtr) internal pure returns (uint256 _basefee) {
        assembly {
            _basefee := and(shr(64, mload(blockPtr)), 0xffffffffffffffff)
        }
    }

    function _loadBlobBaseFee(uint256 blockPtr) internal pure returns (uint256 _blobbasefee) {
        assembly {
            _blobbasefee := and(mload(blockPtr), 0xffffffffffffffff)
        }
    }

    function _loadBlockHash(uint256 blockPtr) internal pure returns (bytes32 _blockhash) {
        assembly {
            _blockhash := mload(add(blockPtr, 32))
        }
    }

    function _loadStateRoot(uint256 blockPtr) internal pure returns (bytes32 _stateroot) {
        assembly {
            _stateroot := mload(add(blockPtr, 64))
        }
    }

    function _loadParentBeaconRoot(uint256 blockPtr) internal pure returns (bytes32 _beaconRoot) {
        assembly {
            _beaconRoot := mload(add(blockPtr, 96))
        }
    }

    function _loadRandao(uint256 blockPtr) internal pure returns (bytes32 _randao) {
        assembly {
            _randao := mload(add(blockPtr, 128))
        }
    }
}
