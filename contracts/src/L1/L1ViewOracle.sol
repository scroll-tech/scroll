// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {IL1ViewOracle} from "./IL1ViewOracle.sol";

contract L1ViewOracle is IL1ViewOracle {
    /**
     * @dev Returns hash of all the blockhashes in the range
     * @param from The block number to get the hash of blockhashes after.
     * @param to The block number to get the hash of blockhashes up to.
     * @return hash The keccak hash of all blockhashes in the provided range
     */
    function blockRangeHash(uint256 from, uint256 to) external view returns (bytes32 hash) {
        require(to >= from, "End must be greater than or equal to start");
        require(to < block.number, "Block range exceeds current block");

        hash = 0;

        for (uint256 i = from; i <= to; i++) {
            bytes32 blockHash = blockhash(i);

            require(blockHash != 0, "Blockhash not available");

            hash = keccak256(abi.encodePacked(hash, blockHash));
        }
    }
}
