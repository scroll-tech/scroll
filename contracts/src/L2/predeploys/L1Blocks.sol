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
    }

    /*************
     * Variables *
     *************/

    address public constant SYSTEM_SENDER = 0xfffffffffffffffffffffffffffffffffffffffe;

    uint32 public constant BLOCK_BUFFER_SIZE = 8192;

    /// @notice Storage slot with the address of the current block hashes offset.
    /// @dev This is the keccak-256 hash of "l1blocks.block_storage_offset".
    bytes32 private constant BLOCK_STORAGE_OFFSET = 0xdb384d0440765c9be19ada21c3d61f9d220d57e5963a1fca370403ac2c4bbbad;

    /// @inheritdoc IL1BlockContainer
    uint64 public latestBlockNumber;

    /***************
     * Constructor *
     ***************/

    constructor() {
        //_transferOwnership(_owner);
    }

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

    modifier onlySystemSender() {
        require(msg.sender == SYSTEM_SENDER, "only system sender allowed");
        _;
    }

    modifier validBlockNumber(uint64 blockNumber) {
        uint64 _latestBlockNumber = latestBlockNumber;
        require(
            blockNumber <= _latestBlockNumber && blockNumber > _latestBlockNumber - BLOCK_BUFFER_SIZE,
            "invalid block number"
        );
    }

    /// @inheritdoc IL1BlockContainer
    function latestBlockHash() external view returns (bytes32) {}

    /// @inheritdoc IL1BlockContainer
    function getBlockHash(uint64 blockNumber) external view validBlockNumber(blockNumber) returns (bytes32) {}

    /// @inheritdoc IL1BlockContainer
    function latestStateRoot() external view returns (bytes32) {}

    /// @inheritdoc IL1BlockContainer
    function getStateRoot(uint64 blockNumber) external view returns (bytes32 stateRoot) {}

    /// @inheritdoc IL1BlockContainer
    function latestBlockTimestamp() external view returns (uint256) {}

    /// @inheritdoc IL1BlockContainer
    function getBlockTimestamp(uint64 blockNumber) external view returns (uint256 timestamp) {}

    /// @inheritdoc IL1BlockContainer
    function latestBaseFee() external view returns (uint256) {}

    /// @inheritdoc IL1BlockContainer
    function getBaseFee(uint64 blockNumber) external view returns (bytes32 baseFee) {}

    /// @inheritdoc IL1BlockContainer
    function latestBlobBaseFee() external view returns (uint256) {}

    /// @inheritdoc IL1BlockContainer
    function getBlobBaseFee(uint64 blockNumber) external view returns (bytes32 blobBaseFee) {}

    /// @inheritdoc IL1BlockContainer
    function latestParentBeaconRoot() external view returns (bytes32) {}

    /// @inheritdoc IL1BlockContainer
    function getParentBeaconRoot(uint64 blockNumber) external view returns (bytes32 parentBeaconRoot) {}

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL1BlockContainer
    function setL1BlockHeader(
        uint64 blockNumber,
        uint64 timestamp,
        uint64 baseFee,
        uint64 blobBaseFee,
        bytes32 blockHash,
        bytes32 stateRoot,
        bytes32 parentBeaconRoot
    ) external onlySystemSender {}
}
