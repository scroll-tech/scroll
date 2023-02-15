// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";

import { ScrollChain, IScrollChain } from "../L1/rollup/ScrollChain.sol";

contract ScrollChainTest is DSTestPlus {
  // from ScrollChain
  event UpdateSequencer(address indexed account, bool status);
  event CommitBatch(bytes32 indexed batchHash);
  event RevertBatch(bytes32 indexed batchHash);
  event FinalizeBatch(bytes32 indexed batchHash);

  ScrollChain private rollup;

  function setUp() public {
    rollup = new ScrollChain(233);
    rollup.initialize();
  }

  function testInitialized() public {
    assertEq(address(this), rollup.owner());
    assertEq(rollup.layer2ChainId(), 233);

    hevm.expectRevert("Initializable: contract is already initialized");
    rollup.initialize();
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
    /* @todo(linxi): fix this
    assertEq(rollup.finalizedBatches(0), bytes32(0));
    bytes32 _batchHash = keccak256(abi.encode(_genesisBatch.blocks[0].blockHash, bytes32(0), 0));

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
        uint64 batchIndex,
        bytes32 parentBatchHash,
        uint64 timestamp,
        ,
        ,
        bool finalized
      ) = rollup.batches(_batchHash);
      assertEq(currStateRoot, bytes32(0));
      assertEq(withdrawTrieRoot, bytes32(0));
      assertEq(batchIndex, 0);
      assertEq(parentBatchHash, 0);
      assertEq(timestamp, _genesisBlock.timestamp);
      assertBoolEq(finalized, true);
    }

    // genesis block imported
    hevm.expectRevert("Genesis batch imported");
    rollup.importGenesisBatch(_genesisBatch);
    */
  }
}
