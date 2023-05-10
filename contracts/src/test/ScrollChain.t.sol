// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {L1MessageQueue} from "../L1/rollup/L1MessageQueue.sol";
import {ScrollChain, IScrollChain} from "../L1/rollup/ScrollChain.sol";

import {MockScrollChain} from "./mocks/MockScrollChain.sol";
import {MockRollupVerifier} from "./mocks/MockRollupVerifier.sol";

contract ScrollChainTest is DSTestPlus {
    // from ScrollChain
    event UpdateSequencer(address indexed account, bool status);
    event UpdateVerifier(address oldVerifier, address newVerifier);

    event CommitBatch(bytes32 indexed batchHash);
    event FinalizeBatch(bytes32 indexed batchHash, bytes32 stateRoot, bytes32 withdrawRoot);

    ScrollChain private rollup;
    L1MessageQueue internal messageQueue;
    MockScrollChain internal chain;
    MockRollupVerifier internal verifier;

    function setUp() public {
        messageQueue = new L1MessageQueue();
        rollup = new ScrollChain(233);
        verifier = new MockRollupVerifier();

        rollup.initialize(address(messageQueue), address(verifier));

        chain = new MockScrollChain();
    }

    function testInitialized() public {
        assertEq(address(this), rollup.owner());
        assertEq(rollup.layer2ChainId(), 233);

        hevm.expectRevert("Initializable: contract is already initialized");
        rollup.initialize(address(messageQueue), address(0));
    }

    /*
    function testPublicInputHash() public {
        IScrollChain.Batch memory batch;
        batch.prevStateRoot = bytes32(0x000000000000000000000000000000000000000000000000000000000000cafe);
        batch.newStateRoot = bytes32(0);
        batch.withdrawTrieRoot = bytes32(0);
        batch
            .l2Transactions = hex"0000007402f8710582fd14808506e38dccc9825208944d496ccc28058b1d74b7a19541663e21154f9c848801561db11e24a43380c080a0d890606d7a35b2ab0f9b866d62c092d5b163f3e6a55537ae1485aac08c3f8ff7a023997be2d32f53e146b160fff0ba81e81dbb4491c865ab174d15c5b3d28c41ae";

        batch.blocks = new IScrollChain.BlockContext[](1);
        batch.blocks[0].blockHash = bytes32(0);
        batch.blocks[0].parentHash = bytes32(0);
        batch.blocks[0].blockNumber = 51966;
        batch.blocks[0].timestamp = 123456789;
        batch.blocks[0].baseFee = 0;
        batch.blocks[0].gasLimit = 10000000000000000;
        batch.blocks[0].numTransactions = 1;
        batch.blocks[0].numL1Messages = 0;

        (bytes32 hash, , , ) = chain.computePublicInputHash(0, batch);
        assertEq(hash, bytes32(0xa9f2ca3175794f91226a410ba1e60fff07a405c957562675c4149b77e659d805));

        batch
            .l2Transactions = hex"00000064f8628001830f424094000000000000000000000000000000000000bbbb8080820a97a064e07cd8f939e2117724bdcbadc80dda421381cbc2a1f4e0d093d9cc5c5cf68ea03e264227f80852d88743cd9e43998f2746b619180366a87e4531debf9c3fa5dc";
        (hash, , , ) = chain.computePublicInputHash(0, batch);
        assertEq(hash, bytes32(0x398cb22bbfa1665c1b342b813267538a4c933d7f92d8bd9184aba0dd1122987b));
    }
    */

    function testCommitBatch() public {
        bytes memory batchHeader0 = new bytes(161);

        // import genesis batch first
        assembly {
            mstore(add(batchHeader0, add(0x20, 25)), 1)
            mstore(add(batchHeader0, add(0x20, 57)), 1)
        }
        rollup.importGenesisBatch(batchHeader0, bytes32(uint256(1)), bytes32(uint256(0)));
        assembly {
            mstore(add(batchHeader0, 185), shl(192, timestamp()))
        }

        // caller not sequencer, revert
        hevm.expectRevert("caller not sequencer");
        rollup.commitBatch(0, batchHeader0, new bytes[](0));

        rollup.updateSequencer(address(this), true);

        // batch is empty, revert
        hevm.expectRevert("batch is empty");
        rollup.commitBatch(0, batchHeader0, new bytes[](0));

        // invalid batch header length, revert
        hevm.expectRevert("invalid batch header length");
        rollup.commitBatch(0, new bytes(160), new bytes[](1));
        hevm.expectRevert("invalid batch header length");
        rollup.commitBatch(0, new bytes(162), new bytes[](1));

        // incorrect parent batch bash, revert
        assembly {
            mstore(add(batchHeader0, add(0x20, 25)), 2) // change data hash for batch0
        }
        hevm.expectRevert("incorrect parent batch bash");
        rollup.commitBatch(0, batchHeader0, new bytes[](1));
        assembly {
            mstore(add(batchHeader0, add(0x20, 25)), 1) // change back
        }

        // chunk length too small, revert
        hevm.expectRevert("chunk length too small");
        rollup.commitBatch(0, batchHeader0, new bytes[](1));

        bytes[] memory chunks = new bytes[](1);
        bytes memory chunk0;

        // no block in chunk, revert
        chunk0 = new bytes(1);
        chunks[0] = chunk0;
        hevm.expectRevert("no block in chunk");
        rollup.commitBatch(0, batchHeader0, chunks);

        // invalid chunk length, revert
        chunk0 = new bytes(1);
        chunk0[0] = bytes1(uint8(1)); // one block in this chunk
        chunks[0] = chunk0;
        hevm.expectRevert("invalid chunk length");
        rollup.commitBatch(0, batchHeader0, chunks);

        // incorrect parent hash, revert
        chunk0 = new bytes(1 + 156);
        chunk0[0] = bytes1(uint8(1)); // one block in this chunk
        assembly {
            mstore(add(chunk0, 33), 2) // blockHash = 2
            mstore(add(chunk0, 65), 2) // parentHash = 2
        }
        chunks[0] = chunk0;
        hevm.expectRevert("incorrect parent hash");
        rollup.commitBatch(0, batchHeader0, chunks);

        chunk0 = new bytes(1 + 156 * 2);
        chunk0[0] = bytes1(uint8(2)); // two blocks in this chunk
        assembly {
            mstore(add(chunk0, 33), 2) // blockHash = 2
            mstore(add(chunk0, 65), 1) // parentHash = 1

            mstore(add(chunk0, 189), 3) // blockHash = 3
            mstore(add(chunk0, 221), 1) // parentHash = 1
        }
        chunks[0] = chunk0;
        hevm.expectRevert("incorrect parent hash");
        rollup.commitBatch(0, batchHeader0, chunks);

        // chunk length mismatch, revert
        chunk0 = new bytes(1 + 156 + 1);
        chunk0[0] = bytes1(uint8(1)); // one block in this chunk
        assembly {
            mstore(add(chunk0, 33), 2) // blockHash = 2
            mstore(add(chunk0, 65), 1) // parentHash = 1
        }
        chunks[0] = chunk0;
        hevm.expectRevert("chunk length mismatch");
        rollup.commitBatch(0, batchHeader0, chunks);

        // commit batch with one chunk, no tx, correctly
        chunk0 = new bytes(1 + 156);
        chunk0[0] = bytes1(uint8(1)); // one block in this chunk
        assembly {
            mstore(add(chunk0, 33), 2) // blockHash = 2
            mstore(add(chunk0, 65), 1) // parentHash = 1
        }
        chunks[0] = chunk0;
        rollup.commitBatch(0, batchHeader0, chunks);
        assertGt(uint256(rollup.committedBatches(1)), 0);
    }

    function testFinalizeBatchWithProof() public {
        // caller not sequencer, revert
        hevm.expectRevert("caller not sequencer");
        rollup.finalizeBatchWithProof(new bytes(0), bytes32(0), bytes32(0), bytes32(0), new bytes(0));

        rollup.updateSequencer(address(this), true);

        bytes memory batchHeader0 = new bytes(161);

        // import genesis batch
        assembly {
            mstore(add(batchHeader0, add(0x20, 25)), 1)
            mstore(add(batchHeader0, add(0x20, 57)), 1)
        }
        rollup.importGenesisBatch(batchHeader0, bytes32(uint256(1)), bytes32(uint256(0)));
        assembly {
            mstore(add(batchHeader0, 185), shl(192, timestamp()))
        }
        bytes32 batchHash0 = rollup.committedBatches(0);

        bytes[] memory chunks = new bytes[](1);
        bytes memory chunk0;

        // commit one batch
        chunk0 = new bytes(1 + 156);
        chunk0[0] = bytes1(uint8(1)); // one block in this chunk
        assembly {
            mstore(add(chunk0, 33), 2) // blockHash = 2
            mstore(add(chunk0, 65), 1) // parentHash = 1
        }
        chunks[0] = chunk0;
        rollup.commitBatch(0, batchHeader0, chunks);
        assertGt(uint256(rollup.committedBatches(1)), 0);

        bytes memory batchHeader1 = new bytes(161);
        assembly {
            mstore(add(batchHeader1, 0x20), 0) // version
            mstore(add(batchHeader1, add(0x20, 1)), shl(192, 1)) // batchIndex
            mstore(add(batchHeader1, add(0x20, 9)), 0) // l1MessagePopped
            mstore(add(batchHeader1, add(0x20, 17)), 0) // totalL1MessagePopped
            mstore(add(batchHeader1, add(0x20, 25)), 0x657b9e1f6ba0dfc7f88bd8eaedb28b65d347e30236e8e114035499177418b318) // dataHash
            mstore(add(batchHeader1, add(0x20, 57)), 2) // lastBlockHash
            mstore(add(batchHeader1, add(0x20, 89)), 0) // skippedL1MessageBitmap
            mstore(add(batchHeader1, add(0x20, 121)), batchHash0) // parentBatchHash
            mstore(add(batchHeader1, add(0x20, 153)), shl(192, timestamp())) // timestamp
        }

        // invalid batch header length, revert
        hevm.expectRevert("invalid batch header length");
        rollup.finalizeBatchWithProof(
            new bytes(160),
            bytes32(uint256(1)),
            bytes32(uint256(2)),
            bytes32(0),
            new bytes(0)
        );
        hevm.expectRevert("invalid batch header length");
        rollup.finalizeBatchWithProof(
            new bytes(162),
            bytes32(uint256(1)),
            bytes32(uint256(2)),
            bytes32(0),
            new bytes(0)
        );

        // incorrect batch bash, revert
        hevm.expectRevert("incorrect batch bash");
        batchHeader1[0] = bytes1(uint8(1)); // change version to 1
        rollup.finalizeBatchWithProof(batchHeader1, bytes32(uint256(1)), bytes32(uint256(2)), bytes32(0), new bytes(0));
        batchHeader1[0] = bytes1(uint8(0)); // change back

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

    function testImportGenesisBlock() public {
        bytes memory batchHeader;

        // invalid batch header length, revert
        batchHeader = new bytes(160);
        hevm.expectRevert("invalid batch header length");
        rollup.importGenesisBatch(batchHeader, bytes32(0), bytes32(0));

        // zero state root, revert
        batchHeader = new bytes(161);
        hevm.expectRevert("zero state root");
        rollup.importGenesisBatch(batchHeader, bytes32(0), bytes32(0));

        // not all fields are zero, revert
        batchHeader = new bytes(161);
        batchHeader[0] = bytes1(uint8(1)); // version not zero
        hevm.expectRevert("not all fields are zero");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)), bytes32(0));

        batchHeader = new bytes(161);
        batchHeader[1] = bytes1(uint8(1)); // batchIndex not zero
        hevm.expectRevert("not all fields are zero");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)), bytes32(0));

        batchHeader = new bytes(161);
        batchHeader[9] = bytes1(uint8(1)); // l1MessagePopped not zero
        hevm.expectRevert("not all fields are zero");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)), bytes32(0));

        batchHeader = new bytes(161);
        batchHeader[17] = bytes1(uint8(1)); // totalL1MessagePopped not zero
        hevm.expectRevert("not all fields are zero");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)), bytes32(0));

        // zero data hash, revert
        batchHeader = new bytes(161);
        hevm.expectRevert("zero data hash");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)), bytes32(0));

        // nonzero last block hash, revert
        batchHeader = new bytes(161);
        batchHeader[25] = bytes1(uint8(1)); // dataHash not zero
        hevm.expectRevert("zero last block hash");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)), bytes32(0));

        // import correctly
        batchHeader = new bytes(161);
        batchHeader[25] = bytes1(uint8(1)); // dataHash not zero
        batchHeader[57] = bytes1(uint8(1)); // lastBlockHash not zero
        assertEq(rollup.finalizedStateRoots(0), bytes32(0));
        assertEq(rollup.withdrawRoots(0), bytes32(0));
        assertEq(rollup.committedBatches(0), bytes32(0));
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)), bytes32(uint256(2)));
        assertEq(rollup.finalizedStateRoots(0), bytes32(uint256(1)));
        assertEq(rollup.withdrawRoots(0), bytes32(uint256(2)));
        assertGt(uint256(rollup.committedBatches(0)), 0);

        // Genesis batch imported, revert
        hevm.expectRevert("Genesis batch imported");
        rollup.importGenesisBatch(batchHeader, bytes32(uint256(1)), bytes32(uint256(2)));
    }
}
