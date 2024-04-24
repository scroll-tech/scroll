// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {IL1Blocks} from "./IL1Blocks.sol";
import {IL1GasPriceOracle} from "./IL1GasPriceOracle.sol";

import {OwnableBase} from "../../libraries/common/OwnableBase.sol";
import {IWhitelist} from "../../libraries/common/IWhitelist.sol";
import {ScrollPredeploy} from "../../libraries/constants/ScrollPredeploy.sol";

/// @title L1BlockContainer
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

    /// @notice Storage slot with the address of the current block hashes offset.
    /// @dev This is the keccak-256 hash of "l1blocks.block_storage_offset".
    bytes32 private constant BLOCK_STORAGE_OFFSET = 0xdb384d0440765c9be19ada21c3d61f9d220d57e5963a1fca370403ac2c4bbbad;

    uint32 public constant BLOCK_FIELD_BYTES = 160;
    uint32 public constant TIMESTAMP_OFFSET = 8;

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
    function latestBlockHash() external view returns (bytes32) {}

    /// @inheritdoc IL1Blocks
    function getBlockHash(uint64 blockNumber) external view validBlockNumber(blockNumber) returns (bytes32) {}

    /// @inheritdoc IL1Blocks
    function latestStateRoot() external view returns (bytes32) {}

    /// @inheritdoc IL1Blocks
    function getStateRoot(uint64 blockNumber) external view returns (bytes32 stateRoot) {}

    /// @inheritdoc IL1Blocks
    function latestBlockTimestamp() external view returns (uint256) {}

    /// @inheritdoc IL1Blocks
    function getBlockTimestamp(uint64 blockNumber) external view returns (uint256 timestamp) {}

    /// @inheritdoc IL1Blocks
    function latestBaseFee() external view returns (uint256) {}

    /// @inheritdoc IL1Blocks
    function getBaseFee(uint64 blockNumber) external view returns (bytes32 baseFee) {}

    /// @inheritdoc IL1Blocks
    function latestBlobBaseFee() external view returns (uint256) {}

    /// @inheritdoc IL1Blocks
    function getBlobBaseFee(uint64 blockNumber) external view returns (bytes32 blobBaseFee) {}

    /// @inheritdoc IL1Blocks
    function latestParentBeaconRoot() external view returns (bytes32) {}

    /// @inheritdoc IL1Blocks
    function getParentBeaconRoot(uint64 blockNumber) external view returns (bytes32 parentBeaconRoot) {}

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
}
