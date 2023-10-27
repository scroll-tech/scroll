// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;


/// @title L1Blocks

contract L1Blocks {
    /// @notice The max count of block hashes to store.
    uint16 public constant BLOCK_HASHES_SIZE = 65536;

    /// @notice The latest L1 block number known by the L2 system.
    uint64 public lastAppliedL1Block;

    /// @notice Storage slot with the address of the current block hashes offset.
    /// @dev This is the keccak-256 hash of "l1blocks.block_hashes_storage_offset".
    bytes32 private constant BLOCK_HASHES_STORAGE_OFFSET =
        0x46b6ca24459c6768b3d8d5d90e9189b00e3ebb5fe38fb16cb9819816d9fe1c2d;

    modifier onlySequencer() {
        require(msg.sender == address(0), "L1Blocks: caller is not the sequencer");
        _;
    }

    constructor(uint64 _firstAppliedL1Block) {
        // The first applied L1 block number.
        lastAppliedL1Block = _firstAppliedL1Block - 1;
    }

    function l1Blockhash(uint256 _number) external view returns (bytes32 hash_) {
        uint64 lastAppliedL1Block_ = lastAppliedL1Block;

        /// @dev It handles the case where the block is in the future.
        require(_number <= lastAppliedL1Block_, "L1Blocks: hash number out of bounds");

        /// @dev It handles the case where the block is no longer in the ring buffer.
        require(lastAppliedL1Block_ - _number < BLOCK_HASHES_SIZE, "L1Blocks: hash number out of bounds");

        assembly {
            hash_ := sload(add(BLOCK_HASHES_STORAGE_OFFSET, mod(_number, BLOCK_HASHES_SIZE)))
        }

        /// @dev The zero hash means the block hash is not yet set.
        require(hash_ != bytes32(0), "L1Blocks: hash number out of bounds");
    }

    function latestBlockhash() external view returns (bytes32 hash_) {
        return l1Blockhash(lastAppliedL1Block);
    }

    function appendBlockhashes(bytes32[] calldata _blocks) external onlySequencer {
        uint64 lastAppliedL1Block_ = lastAppliedL1Block;
        uint256 length = _blocks.length;

        assembly {
            for {
                let i := 0
            } lt(i, length) {
                i := add(i, 1)
            } {
                lastAppliedL1Block_ := add(lastAppliedL1Block_, 1)
                let offset_ := add(BLOCK_HASHES_STORAGE_OFFSET, mod(lastAppliedL1Block_, BLOCK_HASHES_SIZE))
                let hash_ := calldataload(add(0x44, mul(i, 0x20)))
                sstore(offset_, hash_)
            }
        }

        lastAppliedL1Block = lastAppliedL1Block_;
    }
}
