// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {L1MessageQueue} from "../L1/rollup/L1MessageQueue.sol";
import {ScrollChain, IScrollChain} from "../L1/rollup/ScrollChain.sol";

import {MockScrollChain} from "./mocks/MockScrollChain.sol";
import {MockRollupVerifier} from "./mocks/MockRollupVerifier.sol";

// solhint-disable no-inline-assembly

contract ScrollChainTest is DSTestPlus {
    // from ScrollChain
    event UpdateSequencer(address indexed account, bool status);
    event UpdateProver(address indexed account, bool status);
    event UpdateVerifier(address oldVerifier, address newVerifier);
    event UpdateMaxNumL2TxInChunk(uint256 oldMaxNumL2TxInChunk, uint256 newMaxNumL2TxInChunk);

    event CommitBatch(bytes32 indexed batchHash);
    event FinalizeBatch(bytes32 indexed batchHash, bytes32 stateRoot, bytes32 withdrawRoot);

    ScrollChain private rollup;
    L1MessageQueue internal messageQueue;
    MockScrollChain internal chain;
    MockRollupVerifier internal verifier;

    function setUp() public {
        messageQueue = L1MessageQueue(address(new ERC1967Proxy(address(new L1MessageQueue()), new bytes(0))));
        rollup = ScrollChain(address(new ERC1967Proxy(address(new ScrollChain(233)), new bytes(0))));
        verifier = new MockRollupVerifier();

        rollup.initialize(address(messageQueue), address(verifier), 100);
        messageQueue.initialize(address(this), address(rollup), address(0), address(0), 1000000);

        chain = new MockScrollChain();
    }

    function testInitialized() public {
        assertEq(address(this), rollup.owner());
        assertEq(rollup.layer2ChainId(), 233);

        hevm.expectRevert("Initializable: contract is already initialized");
        rollup.initialize(address(messageQueue), address(0), 100);
    }

    function testCommitBatch() public {
        bytes memory batchHeader0 = new bytes(89);

        // import genesis batch first
        assembly {
            mstore(add(batchHeader0, add(0x20, 25)), 1)
        }
        rollup.importGenesisBatch(batchHeader0, bytes32(uint256(1)));

        // caller not sequencer, revert
        hevm.expectRevert("caller not sequencer");
        rollup.commitBatch(0, batchHeader0, new bytes[](0), new bytes(0));

        rollup.updateSequencer(address(this), true);

        // invalid version, revert
        hevm.expectRevert("invalid version");
        rollup.commitBatch(1, batchHeader0, new bytes[](0), new bytes(0));

        // batch is empty, revert
        hevm.expectRevert("batch is empty");
        rollup.commitBatch(0, batchHeader0, new bytes[](0), new bytes(0));

        // batch header length too small, revert
        hevm.expectRevert("batch header length too small");
        rollup.commitBatch(0, new bytes(88), new bytes[](1), new bytes(0));

        // wrong bitmap length, revert
        hevm.expectRevert("wrong bitmap length");
        rollup.commitBatch(0, new bytes(90), new bytes[](1), new bytes(0));

        // incorrect parent batch hash, revert
        assembly {
            mstore(add(batchHeader0, add(0x20, 25)), 2) // change data hash for batch0
        }
        hevm.expectRevert("incorrect parent batch hash");
        rollup.commitBatch(0, batchHeader0, new bytes[](1), new bytes(0));
        assembly {
            mstore(add(batchHeader0, add(0x20, 25)), 1) // change back
        }

        bytes[] memory chunks = new bytes[](1);
        bytes memory chunk0;

        // no block in chunk, revert
        chunk0 = new bytes(1);
        chunks[0] = chunk0;
        hevm.expectRevert("no block in chunk");
        rollup.commitBatch(0, batchHeader0, chunks, new bytes(0));

        // invalid chunk length, revert
        chunk0 = new bytes(1);
        chunk0[0] = bytes1(uint8(1)); // one block in this chunk
        chunks[0] = chunk0;
        hevm.expectRevert("invalid chunk length");
        rollup.commitBatch(0, batchHeader0, chunks, new bytes(0));

        // incomplete l2 transaction data, revert
        chunk0 = new bytes(1 + 60 + 1);
        chunk0[0] = bytes1(uint8(1)); // one block in this chunk
        chunks[0] = chunk0;
        hevm.expectRevert("incomplete l2 transaction data");
        rollup.commitBatch(0, batchHeader0, chunks, new bytes(0));

        // commit batch with one chunk, no tx, correctly
        chunk0 = new bytes(1 + 60);
        chunk0[0] = bytes1(uint8(1)); // one block in this chunk
        chunks[0] = chunk0;
        rollup.commitBatch(0, batchHeader0, chunks, new bytes(0));
        assertGt(uint256(rollup.committedBatches(1)), 0);

        // batch is already committed, revert
        hevm.expectRevert("batch already committed");
        rollup.commitBatch(0, batchHeader0, chunks, new bytes(0));
    }

    function testFinalizeBatchWithProof() public {
        // caller not prover, revert
        hevm.expectRevert("caller not prover");
        rollup.finalizeBatchWithProof(new bytes(0), bytes32(0), bytes32(0), bytes32(0), new bytes(0));

        rollup.updateProver(address(this), true);
        rollup.updateSequencer(address(this), true);

        bytes memory batchHeader0 = new bytes(89);

        // import genesis batch
        assembly {
            mstore(add(batchHeader0, add(0x20, 25)), 1)
        }
        rollup.importGenesisBatch(batchHeader0, bytes32(uint256(1)));
        bytes32 batchHash0 = rollup.committedBatches(0);

        bytes[] memory chunks = new bytes[](1);
        bytes memory chunk0;

        // commit one batch
        chunk0 = new bytes(1 + 60);
        chunk0[0] = bytes1(uint8(1)); // one block in this chunk
        chunks[0] = chunk0;
        rollup.commitBatch(0, batchHeader0, chunks, new bytes(0));
        assertGt(uint256(rollup.committedBatches(1)), 0);

        bytes memory batchHeader1 = new bytes(89);
        assembly {
            mstore(add(batchHeader1, 0x20), 0) // version
            mstore(add(batchHeader1, add(0x20, 1)), shl(192, 1)) // batchIndex
            mstore(add(batchHeader1, add(0x20, 9)), 0) // l1MessagePopped
            mstore(add(batchHeader1, add(0x20, 17)), 0) // totalL1MessagePopped
            mstore(add(batchHeader1, add(0x20, 25)), 0x246394445f4fe64ed5598554d55d1682d6fb3fe04bf58eb54ef81d1189fafb51) // dataHash
            mstore(add(batchHeader1, add(0x20, 57)), batchHash0) // parentBatchHash
        }

        // incorrect batch hash, revert
        hevm.expectRevert("incorrect batch hash");
        batchHeader1[0] = bytes1(uint8(1)); // change version to 1
        rollup.finalizeBatchWithProof(batchHeader1, bytes32(uint256(1)), bytes32(uint256(2)), bytes32(0), new bytes(0));
        batchHeader1[0] = bytes1(uint8(0)); // change back

        // batch header length too small, revert
        hevm.expectRevert("batch header length too small");
        rollup.finalizeBatchWithProof(
            new bytes(88),
            bytes32(uint256(1)),
            bytes32(uint256(2)),
            bytes32(0),
            new bytes(0)
        );

        // wrong bitmap length, revert
        hevm.expectRevert("wrong bitmap length");
        rollup.finalizeBatchWithProof(
            new bytes(90),
            bytes32(uint256(1)),
            bytes32(uint256(2)),
            bytes32(0),
            new bytes(0)
        );

        // incorrect previous state root, revert
        hevm.expectRevert("incorrect previous state root");
        rollup.finalizeBatchWithProof(batchHeader1, bytes32(uint256(2)), bytes32(uint256(2)), bytes32(0), new bytes(0));

        // verify success
        assertBoolEq(rollup.isBatchFinalized(1), false);
        rollup.finalizeBatchWithProof(
            batchHeader1,
            bytes32(uint256(1)),
            bytes32(uint256(2)),
            bytes32(uint256(3)),
            new bytes(0)
        );
        assertBoolEq(rollup.isBatchFinalized(1), true);
        assertEq(rollup.finalizedStateRoots(1), bytes32(uint256(2)));
        assertEq(rollup.withdrawRoots(1), bytes32(uint256(3)));
        assertEq(rollup.lastFinalizedBatchIndex(), 1);

        // batch already verified, revert
        hevm.expectRevert("batch already verified");
        rollup.finalizeBatchWithProof(
            batchHeader1,
            bytes32(uint256(1)),
            bytes32(uint256(2)),
            bytes32(uint256(3)),
            new bytes(0)
        );
    }

    function testCommitAndFinalizeWithL1Messages() public {
        rollup.updateSequencer(address(this), true);
        rollup.updateProver(address(this), true);

        // import 300 L1 messages
        for (uint256 i = 0; i < 300; i++) {
            messageQueue.appendCrossDomainMessage(address(this), 1000000, new bytes(0));
        }

        // import genesis batch first
        bytes memory batchHeader0 = new bytes(89);
        assembly {
            mstore(add(batchHeader0, add(0x20, 25)), 1)
        }
        rollup.importGenesisBatch(batchHeader0, bytes32(uint256(1)));
        bytes32 batchHash0 = rollup.committedBatches(0);

        bytes memory bitmap;
        bytes[] memory chunks;
        bytes memory chunk0;
        bytes memory chunk1;

        // commit batch1, one chunk with one block, 1 tx, 1 L1 message, no skip
        // => payload for data hash of chunk0
        //   0000000000000000
        //   0000000000000000
        //   0000000000000000000000000000000000000000000000000000000000000000
        //   0000000000000000
        //   0001
        //   a2277fd30bbbe74323309023b56035b376d7768ad237ae4fc46ead7dc9591ae1
        // => data hash for chunk0
        //   9ef1e5694bdb014a1eea42be756a8f63bfd8781d6332e9ef3b5126d90c62f110
        // => data hash for all chunks
        //   d9cb6bf9264006fcea490d5c261f7453ab95b1b26033a3805996791b8e3a62f3
        // => payload for batch header
        //   00
        //   0000000000000001
        //   0000000000000001
        //   0000000000000001
        //   d9cb6bf9264006fcea490d5c261f7453ab95b1b26033a3805996791b8e3a62f3
        //   119b828c2a2798d2c957228ebeaff7e10bb099ae0d4e224f3eeb779ff61cba61
        //   0000000000000000000000000000000000000000000000000000000000000000
        // => hash for batch header
        //   00847173b29b238cf319cde79512b7c213e5a8b4138daa7051914c4592b6dfc7
        bytes memory batchHeader1 = new bytes(89 + 32);
        assembly {
            mstore(add(batchHeader1, 0x20), 0) // version
            mstore(add(batchHeader1, add(0x20, 1)), shl(192, 1)) // batchIndex = 1
            mstore(add(batchHeader1, add(0x20, 9)), shl(192, 1)) // l1MessagePopped = 1
            mstore(add(batchHeader1, add(0x20, 17)), shl(192, 1)) // totalL1MessagePopped = 1
            mstore(add(batchHeader1, add(0x20, 25)), 0xd9cb6bf9264006fcea490d5c261f7453ab95b1b26033a3805996791b8e3a62f3) // dataHash
            mstore(add(batchHeader1, add(0x20, 57)), batchHash0) // parentBatchHash
            mstore(add(batchHeader1, add(0x20, 89)), 0) // bitmap0
        }
        chunk0 = new bytes(1 + 60);
        assembly {
            mstore(add(chunk0, 0x20), shl(248, 1)) // numBlocks = 1
            mstore(add(chunk0, add(0x21, 56)), shl(240, 1)) // numTransactions = 1
            mstore(add(chunk0, add(0x21, 58)), shl(240, 1)) // numL1Messages = 1
        }
        chunks = new bytes[](1);
        chunks[0] = chunk0;
        bitmap = new bytes(32);
        rollup.commitBatch(0, batchHeader0, chunks, bitmap);
        assertBoolEq(rollup.isBatchFinalized(1), false);
        bytes32 batchHash1 = rollup.committedBatches(1);
        assertEq(batchHash1, bytes32(0x00847173b29b238cf319cde79512b7c213e5a8b4138daa7051914c4592b6dfc7));

        // finalize batch1
        rollup.finalizeBatchWithProof(
            batchHeader1,
            bytes32(uint256(1)),
            bytes32(uint256(2)),
            bytes32(uint256(3)),
            new bytes(0)
        );
        assertBoolEq(rollup.isBatchFinalized(1), true);
        assertEq(rollup.finalizedStateRoots(1), bytes32(uint256(2)));
        assertEq(rollup.withdrawRoots(1), bytes32(uint256(3)));
        assertEq(rollup.lastFinalizedBatchIndex(), 1);
        assertEq(messageQueue.getCrossDomainMessage(0), bytes32(0));
        assertEq(messageQueue.pendingQueueIndex(), 1);

        // commit batch2 with two chunks, correctly
        // 1. chunk0 has one block, 3 tx, no L1 messages
        //   => payload for chunk0
        //    0000000000000000
        //    0000000000000000
        //    0000000000000000000000000000000000000000000000000000000000000000
        //    0000000000000000
        //    0003
        //    ... (some tx hashes)
        //   => data hash for chunk0
        //    2ac1dad3f3696e5581dfc10f2c7a7a8fc5b344285f7d332c7895a8825fca609a
        // 2. chunk1 has three blocks
        //   2.1 block0 has 5 tx, 3 L1 messages, no skips
        //   2.2 block1 has 10 tx, 5 L1 messages, even is skipped.
        //   2.2 block1 has 300 tx, 256 L1 messages, odd position is skipped.
        //   => payload for chunk1
        //    0000000000000000
        //    0000000000000000
        //    0000000000000000000000000000000000000000000000000000000000000000
        //    0000000000000000
        //    0005
        //    0000000000000000
        //    0000000000000000
        //    0000000000000000000000000000000000000000000000000000000000000000
        //    0000000000000000
        //    000a
        //    0000000000000000
        //    0000000000000000
        //    0000000000000000000000000000000000000000000000000000000000000000
        //    0000000000000000
        //    012c
        //    ... (some tx hashes)
        //   => data hash for chunk2
        //    0520f1fbe159af97fdf1d6cfcfe7605f99f7bfe3ed876e87b64250b1810df00b
        // => data hash for all chunks
        //   f52343299f6379fd15b20b23d51fc61b9b357b124be112686626b6278bcffa83
        // => payload for batch header
        //  00
        //  0000000000000002
        //  0000000000000108
        //  0000000000000109
        //  f52343299f6379fd15b20b23d51fc61b9b357b124be112686626b6278bcffa83
        //  cef70bf80683c4d9b8b2813e90c314e8c56648e231300b8cfed9d666b0caf14e
        //  aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa800000000000000000000000000000000000000000000000000000000000000aa
        // => hash for batch header
        //  2231cf185a5c07931584f970738b4cd2ae4fb39e2d90853b26746c7616ea71b9
        bytes memory batchHeader2 = new bytes(89 + 32 + 32);
        assembly {
            mstore(add(batchHeader2, 0x20), 0) // version
            mstore(add(batchHeader2, add(0x20, 1)), shl(192, 2)) // batchIndex = 2
            mstore(add(batchHeader2, add(0x20, 9)), shl(192, 264)) // l1MessagePopped = 264
            mstore(add(batchHeader2, add(0x20, 17)), shl(192, 265)) // totalL1MessagePopped = 265
            mstore(add(batchHeader2, add(0x20, 25)), 0xf52343299f6379fd15b20b23d51fc61b9b357b124be112686626b6278bcffa83) // dataHash
            mstore(add(batchHeader2, add(0x20, 57)), batchHash1) // parentBatchHash
            mstore(
                add(batchHeader2, add(0x20, 89)),
                77194726158210796949047323339125271902179989777093709359638389338608753093288
            ) // bitmap0
            mstore(add(batchHeader2, add(0x20, 121)), 170) // bitmap1
        }
        chunk0 = new bytes(1 + 60 + 3 * 5);
        assembly {
            mstore(add(chunk0, 0x20), shl(248, 1)) // numBlocks = 1
            mstore(add(chunk0, add(0x21, 56)), shl(240, 3)) // numTransactions = 3
            mstore(add(chunk0, add(0x21, 58)), shl(240, 0)) // numL1Messages = 0
        }
        for (uint256 i = 0; i < 3; i++) {
            assembly {
                mstore(add(chunk0, add(93, mul(i, 5))), shl(224, 1)) // tx = "0x00"
            }
        }
        chunk1 = new bytes(1 + 60 * 3 + 51 * 5);
        assembly {
            mstore(add(chunk1, 0x20), shl(248, 3)) // numBlocks = 3
            mstore(add(chunk1, add(33, 56)), shl(240, 5)) // block0.numTransactions = 5
            mstore(add(chunk1, add(33, 58)), shl(240, 3)) // block0.numL1Messages = 3
            mstore(add(chunk1, add(93, 56)), shl(240, 10)) // block1.numTransactions = 10
            mstore(add(chunk1, add(93, 58)), shl(240, 5)) // block1.numL1Messages = 5
            mstore(add(chunk1, add(153, 56)), shl(240, 300)) // block1.numTransactions = 300
            mstore(add(chunk1, add(153, 58)), shl(240, 256)) // block1.numL1Messages = 256
        }
        for (uint256 i = 0; i < 51; i++) {
            assembly {
                mstore(add(chunk1, add(213, mul(i, 5))), shl(224, 1)) // tx = "0x00"
            }
        }
        chunks = new bytes[](2);
        chunks[0] = chunk0;
        chunks[1] = chunk1;
        bitmap = new bytes(64);
        assembly {
            mstore(
                add(bitmap, add(0x20, 0)),
                77194726158210796949047323339125271902179989777093709359638389338608753093288
            ) // bitmap0
            mstore(add(bitmap, add(0x20, 32)), 170) // bitmap1
        }

        rollup.commitBatch(0, batchHeader1, chunks, bitmap);
        assertBoolEq(rollup.isBatchFinalized(2), false);
        bytes32 batchHash2 = rollup.committedBatches(2);
        assertEq(batchHash2, bytes32(0x2231cf185a5c07931584f970738b4cd2ae4fb39e2d90853b26746c7616ea71b9));

        // verify committed batch correctly
        rollup.finalizeBatchWithProof(
            batchHeader2,
            bytes32(uint256(2)),
            bytes32(uint256(4)),
            bytes32(uint256(5)),
            new bytes(0)
        );
        assertBoolEq(rollup.isBatchFinalized(2), true);
        assertEq(rollup.finalizedStateRoots(2), bytes32(uint256(4)));
        assertEq(rollup.withdrawRoots(2), bytes32(uint256(5)));
        assertEq(rollup.lastFinalizedBatchIndex(), 2);
        assertEq(messageQueue.pendingQueueIndex(), 265);
        // 1 ~ 4, zero
        for (uint256 i = 1; i < 4; i++) {
            assertEq(messageQueue.getCrossDomainMessage(i), bytes32(0));
        }
        // 4 ~ 9, even is nonzero, odd is zero
        for (uint256 i = 4; i < 9; i++) {
            if (i % 2 == 1) {
                assertEq(messageQueue.getCrossDomainMessage(i), bytes32(0));
            } else {
                assertGt(uint256(messageQueue.getCrossDomainMessage(i)), 0);
            }
        }
        // 9 ~ 265, even is nonzero, odd is zero
        for (uint256 i = 9; i < 265; i++) {
            if (i % 2 == 1) {
                assertEq(messageQueue.getCrossDomainMessage(i), bytes32(0));
            } else {
                assertGt(uint256(messageQueue.getCrossDomainMessage(i)), 0);
            }
        }
    }

    function testRevertBatch() public {
        // caller not owner, revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        rollup.revertBatch(new bytes(89), 1);
        hevm.stopPrank();

        rollup.updateSequencer(address(this), true);

        bytes memory batchHeader0 = new bytes(89);

        // import genesis batch
        assembly {
            mstore(add(batchHeader0, add(0x20, 25)), 1)
        }
        rollup.importGenesisBatch(batchHeader0, bytes32(uint256(1)));
        bytes32 batchHash0 = rollup.committedBatches(0);

        bytes[] memory chunks = new bytes[](1);
        bytes memory chunk0;

        // commit one batch
        chunk0 = new bytes(1 + 60);
        chunk0[0] = bytes1(uint8(1)); // one block in this chunk
        chunks[0] = chunk0;
        rollup.commitBatch(0, batchHeader0, chunks, new bytes(0));

        bytes memory batchHeader1 = new bytes(89);
        assembly {
            mstore(add(batchHeader1, 0x20), 0) // version
            mstore(add(batchHeader1, add(0x20, 1)), shl(192, 1)) // batchIndex
            mstore(add(batchHeader1, add(0x20, 9)), 0) // l1MessagePopped
            mstore(add(batchHeader1, add(0x20, 17)), 0) // totalL1MessagePopped
            mstore(add(batchHeader1, add(0x20, 25)), 0x246394445f4fe64ed5598554d55d1682d6fb3fe04bf58eb54ef81d1189fafb51) // dataHash
            mstore(add(batchHeader1, add(0x20, 57)), batchHash0) // parentBatchHash
        }

        // commit another batch
        rollup.commitBatch(0, batchHeader1, chunks, new bytes(0));

        // count must be nonzero, revert
        hevm.expectRevert("count must be nonzero");
        rollup.revertBatch(batchHeader0, 0);

        // incorrect batch hash, revert
        hevm.expectRevert("incorrect batch hash");
        batchHeader1[0] = bytes1(uint8(1)); // change version to 1
        rollup.revertBatch(batchHeader1, 1);
        batchHeader1[0] = bytes1(uint8(0)); // change back

        // can only revert unfinalized batch, revert
        hevm.expectRevert("can only revert unfinalized batch");
        rollup.revertBatch(batchHeader0, 1);

        // succeed to revert next two pending batches.
        assertGt(uint256(rollup.committedBatches(1)), 0);
        assertGt(uint256(rollup.committedBatches(2)), 0);
        rollup.revertBatch(batchHeader1, 2);
        assertEq(uint256(rollup.committedBatches(1)), 0);
        assertEq(uint256(rollup.committedBatches(2)), 0);
    }

    function testUpdateSequencer(address _sequencer) public {
        // set by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        rollup.updateSequencer(_sequencer, true);
        hevm.stopPrank();

        // change to random operator
        hevm.expectEmit(true, false, false, true);
        emit UpdateSequencer(_sequencer, true);

        assertBoolEq(rollup.isSequencer(_sequencer), false);
        rollup.updateSequencer(_sequencer, true);
        assertBoolEq(rollup.isSequencer(_sequencer), true);

        hevm.expectEmit(true, false, false, true);
        emit UpdateSequencer(_sequencer, false);
        rollup.updateSequencer(_sequencer, false);
        assertBoolEq(rollup.isSequencer(_sequencer), false);
    }

    function testUpdateProver(address _prover) public {
        // set by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        rollup.updateProver(_prover, true);
        hevm.stopPrank();

        // change to random operator
        hevm.expectEmit(true, false, false, true);
        emit UpdateProver(_prover, true);

        assertBoolEq(rollup.isProver(_prover), false);
        rollup.updateProver(_prover, true);
        assertBoolEq(rollup.isProver(_prover), true);

        hevm.expectEmit(true, false, false, true);
        emit UpdateProver(_prover, false);
        rollup.updateProver(_prover, false);
        assertBoolEq(rollup.isProver(_prover), false);
    }

    function testUpdateVerifier(address _newVerifier) public {
        // set by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        rollup.updateVerifier(_newVerifier);
        hevm.stopPrank();

        // change to random operator
        hevm.expectEmit(false, false, false, true);
        emit UpdateVerifier(address(verifier), _newVerifier);

        assertEq(rollup.verifier(), address(verifier));
        rollup.updateVerifier(_newVerifier);
        assertEq(rollup.verifier(), _newVerifier);
    }

    function testUpdateMaxNumL2TxInChunk(uint256 _maxNumL2TxInChunk) public {
        // set by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        rollup.updateMaxNumL2TxInChunk(_maxNumL2TxInChunk);
        hevm.stopPrank();

        // change to random operator
        hevm.expectEmit(false, false, false, true);
        emit UpdateMaxNumL2TxInChunk(100, _maxNumL2TxInChunk);

        assertEq(rollup.maxNumL2TxInChunk(), 100);
        rollup.updateMaxNumL2TxInChunk(_maxNumL2TxInChunk);
        assertEq(rollup.maxNumL2TxInChunk(), _maxNumL2TxInChunk);
    }

    function testImportGenesisBlock() public {
        bytes memory batchHeader;

        // zero state root, revert
        batchHeader = new bytes(89);
        hevm.expectRevert("zero state root");
        rollup.importGenesisBatch(batchHeader, bytes32(0));

        // batch header length too small, revert
        batchHeader = new bytes(88);
        hevm.expectRevert("batch header length too small");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)));

        // wrong bitmap length, revert
        batchHeader = new bytes(90);
        hevm.expectRevert("wrong bitmap length");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)));

        // not all fields are zero, revert
        batchHeader = new bytes(89);
        batchHeader[0] = bytes1(uint8(1)); // version not zero
        hevm.expectRevert("not all fields are zero");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)));

        batchHeader = new bytes(89);
        batchHeader[1] = bytes1(uint8(1)); // batchIndex not zero
        hevm.expectRevert("not all fields are zero");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)));

        batchHeader = new bytes(89 + 32);
        assembly {
            mstore(add(batchHeader, add(0x20, 9)), shl(192, 1)) // l1MessagePopped not zero
        }
        hevm.expectRevert("not all fields are zero");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)));

        batchHeader = new bytes(89);
        batchHeader[17] = bytes1(uint8(1)); // totalL1MessagePopped not zero
        hevm.expectRevert("not all fields are zero");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)));

        // zero data hash, revert
        batchHeader = new bytes(89);
        hevm.expectRevert("zero data hash");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)));

        // nonzero parent batch hash, revert
        batchHeader = new bytes(89);
        batchHeader[25] = bytes1(uint8(1)); // dataHash not zero
        batchHeader[57] = bytes1(uint8(1)); // parentBatchHash not zero
        hevm.expectRevert("nonzero parent batch hash");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)));

        // import correctly
        batchHeader = new bytes(89);
        batchHeader[25] = bytes1(uint8(1)); // dataHash not zero
        assertEq(rollup.finalizedStateRoots(0), bytes32(0));
        assertEq(rollup.withdrawRoots(0), bytes32(0));
        assertEq(rollup.committedBatches(0), bytes32(0));
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)));
        assertEq(rollup.finalizedStateRoots(0), bytes32(uint256(1)));
        assertEq(rollup.withdrawRoots(0), bytes32(0));
        assertGt(uint256(rollup.committedBatches(0)), 0);

        // Genesis batch imported, revert
        hevm.expectRevert("Genesis batch imported");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)));
    }
}
