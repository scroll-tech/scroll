// SPDX-License-Identifier: MIT

pragma solidity ^0.8.20;

interface IL1Blocks {
    /**********
     * Errors *
     **********/

    /// @dev Thrown when the given block number is not available in the storage;
    error ErrorBlockUnavailable();

    /**********
     * Events *
     **********/

    /// @notice Emitted when a block is imported.
    /// @param blockHash The hash of the imported block.
    /// @param blockHeight The height of the imported block.
    /// @param blockTimestamp The timestamp of the imported block.
    /// @param baseFee The base fee of the imported block.
    /// @param stateRoot The state root of the imported block.
    event ImportBlock(
        bytes32 indexed blockHash,
        uint256 blockHeight,
        uint256 blockTimestamp,
        uint256 baseFee,
        bytes32 stateRoot
    );

    /*************************
     * Public View Functions *
     *************************/

    // /// @notice Return the latest imported L1 block number
    function latestBlockNumber() external view returns (uint256);

    /// @notice Return the latest imported L1 block hash
    function latestBlockHash() external view returns (bytes32);

    /// @notice Return the block hash of the given block
    /// @param blockNumber The L1 block number
    /// @return blockHash The block hash of the block
    function getBlockHash(uint256 blockNumber) external view returns (bytes32);

    /// @notice Return the latest imported L1 state root
    function latestStateRoot() external view returns (bytes32);

    /// @notice Return the state root of given block
    /// @param blockNumber The L1 block number
    /// @return stateRoot The state root of the block
    function getStateRoot(uint256 blockNumber) external view returns (bytes32);

    /// @notice Return the latest imported block timestamp
    function latestBlockTimestamp() external view returns (uint256);

    /// @notice Return the block timestamp of the given block
    /// @param blockNumber The L1 block number
    /// @return timestamp The block timestamp of the block
    function getBlockTimestamp(uint256 blockNumber) external view returns (uint256);

    /// @notice Return the latest imported L1 base fee
    function latestBaseFee() external view returns (uint256);

    /// @notice Return the base fee of the given block
    /// @param blockNumber The L1 block number
    /// @return baseFee The base fee of the block
    function getBaseFee(uint256 blockNumber) external view returns (uint256);

    /// @notice Return the latest imported L1 blob base fee
    function latestBlobBaseFee() external view returns (uint256);

    /// @notice Return the blob base fee of the given block
    /// @param blockNumber The L1 block number
    /// @return blobBaseFee The blob base fee of the block
    function getBlobBaseFee(uint256 blockNumber) external view returns (uint256);

    /// @notice Return the latest imported parent beacon block root
    function latestParentBeaconRoot() external view returns (bytes32);

    /// @notice Return the state root of given block
    /// @param blockNumber The L1 block number
    /// @return parentBeaconRoot The parent beacon block root of the block
    function getParentBeaconRoot(uint256 blockNumber) external view returns (bytes32);

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Import L1 block header to this contract
    /// @param blockHeaderRlp The RLP encoding of L1 block
    /// @return blockHash The block hash
    function setL1BlockHeader(bytes calldata blockHeaderRlp) external returns (bytes32 blockHash);
}
