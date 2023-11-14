// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

interface IL1ViewOracle {
    /**
     * @dev Returns hash of all the blockhashes in the range
     * @param from The block number to get the hash of blockhashes after.
     * @param to The block number to get the hash of blockhashes up to.
     * @return hash The keccak hash of all blockhashes in the provided range
     */
    function blockRangeHash(uint256 from, uint256 to) external view returns (bytes32 hash);
}
