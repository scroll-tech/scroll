// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {L1Blocks} from "../L2/L1Blocks.sol";

contract L1BlocksTest is DSTestPlus {
    L1Blocks private l1Blocks;
    uint32 private blockHashesSize;
    uint64 private firstAppliedL1Block = 1;
    address private sequencer = 0x5300000000000000000000000000000000000005;

    function setUp() public {
        l1Blocks = new L1Blocks(firstAppliedL1Block);
        blockHashesSize = l1Blocks.BLOCK_HASHES_SIZE();
    }

    function testFuzzAppendBlockhashesSingleSuccess(bytes32 _hash) external {
        hevm.assume(_hash != bytes32(0));
        bytes32[] memory hashes = new bytes32[](1);
        hashes[0] = _hash;

        hevm.startPrank(address(0));
        l1Blocks.appendBlockhashes(hashes);
        hevm.stopPrank();

        assertEq(l1Blocks.latestBlockhash(), _hash);
        assertEq(l1Blocks.l1Blockhash(l1Blocks.lastAppliedL1Block()), hashes[0]);
    }

    function testFuzzAppendBlockhashesManySuccess(bytes32[] memory hashes) external {
        uint256 size = hashes.length;
        hevm.assume(size > 0);

        for (uint256 i = 0; i < size; i++) {
            if (hashes[i] == bytes32(0)) {
                hashes[i] = keccak256(abi.encodePacked(i));
            }
        }

        hevm.startPrank(sequencer);
        l1Blocks.appendBlockhashes(hashes);
        hevm.stopPrank();

        uint256 lastAppliedL1Block = l1Blocks.lastAppliedL1Block();

        for (uint256 i = 0; i < size; i++) {
            assertEq(l1Blocks.l1Blockhash(lastAppliedL1Block - size + 1 + i), hashes[i]);
        }
        assertEq(l1Blocks.latestBlockhash(), hashes[size - 1]);
    }

    function testFuzzGetL1BlockHashLowerBoundFail(uint256 lowerBound) external {
        lowerBound = lowerBound % uint256(firstAppliedL1Block);

        bytes32[] memory hashes = new bytes32[](1);
        hashes[0] = keccak256(abi.encodePacked(lowerBound));

        hevm.startPrank(sequencer);
        l1Blocks.appendBlockhashes(hashes);
        hevm.stopPrank();

        assertEq(l1Blocks.latestBlockhash(), hashes[0]);

        hevm.expectRevert("L1Blocks: hash number out of bounds");
        l1Blocks.l1Blockhash(lowerBound);
    }

    function testFuzzGetL1BlockHashUpperBoundFail(uint64 upperBound) external {
        uint256 lastAppliedL1Block = l1Blocks.lastAppliedL1Block();
        hevm.assume(upperBound > lastAppliedL1Block + 1);

        bytes32[] memory hashes = new bytes32[](1);
        hashes[0] = keccak256(abi.encodePacked(upperBound));

        hevm.startPrank(sequencer);
        l1Blocks.appendBlockhashes(hashes);
        hevm.stopPrank();

        assertEq(l1Blocks.latestBlockhash(), hashes[0]);

        hevm.expectRevert("L1Blocks: hash number out of bounds");
        l1Blocks.l1Blockhash(upperBound);
    }

    function testFuzzAppendBlockhashesNonSequencerFail(address nonSequencer) external {
        hevm.assume(nonSequencer != address(0));

        bytes32[] memory hashes = new bytes32[](1);
        hashes[0] = keccak256(abi.encodePacked("test"));

        hevm.startPrank(nonSequencer);
        hevm.expectRevert("L1Blocks: caller is not the sequencer");
        l1Blocks.appendBlockhashes(hashes);
        hevm.stopPrank();
    }

    function testGetL1BlockHashOverwrittenRingMapSuccess() external {
        hevm.startPrank(sequencer);

        uint64 lowerBound = 0;
        uint8 times = 3;
        bytes32[] memory hashes = new bytes32[](1);
        bytes32 testHash = keccak256(abi.encodePacked("test"));

        for (uint64 i = 1; i <= uint256(times) * blockHashesSize + (times - 1); i++) {
            hashes[0] = bytes32(uint256(testHash) + i);
            l1Blocks.appendBlockhashes(hashes);

            assertEq(l1Blocks.latestBlockhash(), hashes[0]);

            if (i % blockHashesSize == 0) {
                lowerBound = i - blockHashesSize + 1;

                hevm.expectRevert("L1Blocks: hash number out of bounds");
                l1Blocks.l1Blockhash(lowerBound - 1);

                for (uint64 k = lowerBound; k < i + 1; k++) {
                    assertEq(l1Blocks.l1Blockhash(k), bytes32(uint256(testHash) + k));
                }

                hevm.expectRevert("L1Blocks: hash number out of bounds");
                l1Blocks.l1Blockhash(i + 1);
            }
        }

        hevm.stopPrank();
    }
}
