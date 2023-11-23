// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {IL1ViewOracle} from "./IL1ViewOracle.sol";

contract L1ViewOracle is IL1ViewOracle {
    /**
     * @dev Returns hash of all the blockhashes in the range.
     * @param _from The block number to get the hash of blockhashes after.
     * @param _to The block number to get the hash of blockhashes up to.
     * @return hash_ The keccak hash of all blockhashes in the provided range.
     */
    function blockRangeHash(uint256 _from, uint256 _to) external view returns (bytes32 hash_) {
        require(_from > 0, "Incorrect from/to range");
        require(_to >= _from, "Incorrect from/to range");
        require(_to < block.number, "Incorrect from/to range");

        bytes32[] memory blockHashes = new bytes32[](_to - _from + 1);
        uint256 cnt = 0;

        for (uint256 i = _from; i <= _to; i++) {
            bytes32 blockHash = blockhash(i);
            require(blockHash != 0, "Blockhash not available");
            blockHashes[cnt++] = blockHash;
        }

        hash_ = keccak256(abi.encodePacked(blockHashes));
    }
}
