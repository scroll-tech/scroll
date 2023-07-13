// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

interface IL1BlockContainer {
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

    /// @notice Return the latest imported block hash
    function latestBlockHash() external view returns (bytes32);

    /// @notice Return the latest imported L1 base fee
    function latestBaseFee() external view returns (uint256);

    /// @notice Return the latest imported block number
    function latestBlockNumber() external view returns (uint256);

    /// @notice Return the latest imported block timestamp
    function latestBlockTimestamp() external view returns (uint256);

    /// @notice Return the state root of given block.
    /// @param blockHash The block hash to query.
    /// @return stateRoot The state root of the block.
    function getStateRoot(bytes32 blockHash) external view returns (bytes32 stateRoot);

    /// @notice Return the block timestamp of given block.
    /// @param blockHash The block hash to query.
    /// @return timestamp The corresponding block timestamp.
    function getBlockTimestamp(bytes32 blockHash) external view returns (uint256 timestamp);

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Import L1 block header to this contract.
    /// @param blockHash The hash of block.
    /// @param blockHeaderRLP The RLP encoding of L1 block.
    /// @param updateGasPriceOracle Whether to update gas price oracle.
    function importBlockHeader(
        bytes32 blockHash,
        bytes calldata blockHeaderRLP,
        bool updateGasPriceOracle
    ) external;
}
