// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

interface IL1Blocks {
    /**
     * @dev Gets the l1 block hash for a given its block number
     * @param _number The l1 block number
     * @return hash_ The l1 block hash for the provided block number
     */
    function l1Blockhash(uint256 _number) external view returns (bytes32 hash_);

    /**
     * @dev Gets the latest l1 block hash applied by the sequencer
     * @notice This does not mean that this is the latest L1 block number in the
     * L1 blockchain, but rather the last item in the block hashes array
     * @return hash_ The latest l1 block hash from the block hashes array
     */
    function latestBlockhash() external view returns (bytes32 hash_);

		/**
     * @dev Appends an array of block hashes to the block hashes array
     * @param _blocks The array of new block hashes
     */
    function appendBlockhashes(bytes32[] calldata _blocks) external;
}