// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {L1ViewOracle} from "../L1/L1ViewOracle.sol";

contract L1ViewOracleTest is DSTestPlus {
    L1ViewOracle private oracle;

    function setUp() public {
        oracle = new L1ViewOracle();
    }

    function testTooOldBlocks() external {
        hevm.expectRevert("Blockhash not available");

        hevm.roll(300);

        uint256 from = block.number - 260;
        uint256 to = from + 5;

        bytes32 hash = oracle.blockRangeHash(from, to);
    }

    function testTooNewBlocks() external {
        hevm.expectRevert("Block range exceeds current block");

        hevm.roll(10);

        uint256 from = block.number - 5;
        uint256 to = block.number + 5;

        bytes32 hash = oracle.blockRangeHash(from, to);
    }

    function testInvalidRange() external {
        hevm.expectRevert("End must be greater than or equal to start");

        uint256 from = 200;
        uint256 to = 100;

        bytes32 hash = oracle.blockRangeHash(from, to);
    }

    function testCorrectness() external {
        hevm.roll(150);

        uint256 from = 15;
        uint256 to = 48;

        bytes32 expectedHash = 0;

        for (uint256 i = from; i <= to; i++) {
            bytes32 blockHash = blockhash(i);

            require(blockHash != 0, "Blockhash not available");

            expectedHash = keccak256(abi.encodePacked(expectedHash, blockHash));
        }

        bytes32 gotHash = oracle.blockRangeHash(from, to);

        assertEq(expectedHash, gotHash);
    }
}
