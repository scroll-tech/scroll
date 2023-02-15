// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";

import { L1MessageQueue } from "../L1/rollup/L1MessageQueue.sol";
import { ScrollChain, IScrollChain } from "../L1/rollup/ScrollChain.sol";

import { MockScrollChain } from "./mocks/MockScrollChain.sol";

contract ScrollChainTest is DSTestPlus {
  // from ScrollChain
  event UpdateSequencer(address indexed account, bool status);
  event CommitBatch(bytes32 indexed batchHash);
  event RevertBatch(bytes32 indexed batchHash);
  event FinalizeBatch(bytes32 indexed batchHash);

  ScrollChain private rollup;
  L1MessageQueue internal messageQueue;
  MockScrollChain internal chain;

  function setUp() public {
    messageQueue = new L1MessageQueue();
    rollup = new ScrollChain(233, 4, 0xb5baa665b2664c3bfed7eb46e00ebc110ecf2ebd257854a9bf2b9dbc9b2c08f6);

    rollup.initialize(address(messageQueue));

    chain = new MockScrollChain();
  }

  function testInitialized() public {
    assertEq(address(this), rollup.owner());
    assertEq(rollup.layer2ChainId(), 233);

    hevm.expectRevert("Initializable: contract is already initialized");
    rollup.initialize(address(messageQueue));
  }

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

  function testImportGenesisBlock(IScrollChain.BlockContext memory _genesisBlock) public {
    hevm.assume(_genesisBlock.blockHash != bytes32(0));
    _genesisBlock.blockNumber = 0;
    _genesisBlock.parentHash = bytes32(0);
    _genesisBlock.numTransactions = 0;
    _genesisBlock.numL1Messages = 0;

    IScrollChain.Batch memory _genesisBatch;
    _genesisBatch.blocks = new IScrollChain.BlockContext[](1);
    _genesisBatch.blocks[0] = _genesisBlock;
    _genesisBatch.newStateRoot = bytes32(uint256(2));

    // Not exact one block in genesis, should revert
    _genesisBatch.blocks = new IScrollChain.BlockContext[](2);
    hevm.expectRevert("Not exact one block in genesis");
    rollup.importGenesisBatch(_genesisBatch);
    _genesisBatch.blocks = new IScrollChain.BlockContext[](0);
    hevm.expectRevert("Not exact one block in genesis");
    rollup.importGenesisBatch(_genesisBatch);

    _genesisBatch.blocks = new IScrollChain.BlockContext[](1);
    _genesisBatch.blocks[0] = _genesisBlock;

    // Nonzero prevStateRoot, should revert
    _genesisBatch.prevStateRoot = bytes32(uint256(1));
    hevm.expectRevert("Nonzero prevStateRoot");
    rollup.importGenesisBatch(_genesisBatch);
    _genesisBatch.prevStateRoot = bytes32(0);

    // Block hash is zero, should revert
    bytes32 _originalHash = _genesisBatch.blocks[0].blockHash;
    _genesisBatch.blocks[0].blockHash = bytes32(0);
    hevm.expectRevert("Block hash is zero");
    rollup.importGenesisBatch(_genesisBatch);
    _genesisBatch.blocks[0].blockHash = _originalHash;

    // Block is not genesis, should revert
    _genesisBatch.blocks[0].blockNumber = 1;
    hevm.expectRevert("Block is not genesis");
    rollup.importGenesisBatch(_genesisBatch);
    _genesisBatch.blocks[0].blockNumber = 0;

    // Parent hash not empty, should revert
    _genesisBatch.blocks[0].parentHash = bytes32(uint256(2));
    hevm.expectRevert("Parent hash not empty");
    rollup.importGenesisBatch(_genesisBatch);
    _genesisBatch.blocks[0].parentHash = bytes32(0);

    // import correctly
    assertEq(rollup.finalizedBatches(0), bytes32(0));
    (bytes32 _batchHash, , , ) = chain.computePublicInputHash(0, _genesisBatch);

    hevm.expectEmit(true, false, false, true);
    emit CommitBatch(_batchHash);
    hevm.expectEmit(true, false, false, true);
    emit FinalizeBatch(_batchHash);
    rollup.importGenesisBatch(_genesisBatch);
    {
      assertEq(rollup.finalizedBatches(0), _batchHash);
      assertEq(rollup.lastFinalizedBatchHash(), _batchHash);
      (
        bytes32 currStateRoot,
        bytes32 withdrawTrieRoot,
        bytes32 parentBatchHash,
        uint64 batchIndex,
        uint64 timestamp,
        ,
        ,
        bool finalized
      ) = rollup.batches(_batchHash);
      assertEq(currStateRoot, bytes32(uint256(2)));
      assertEq(withdrawTrieRoot, bytes32(0));
      assertEq(batchIndex, 0);
      assertEq(parentBatchHash, 0);
      assertEq(timestamp, _genesisBlock.timestamp);
      assertBoolEq(finalized, true);
    }

    // genesis block imported
    hevm.expectRevert("Genesis batch imported");
    rollup.importGenesisBatch(_genesisBatch);
  }
}
