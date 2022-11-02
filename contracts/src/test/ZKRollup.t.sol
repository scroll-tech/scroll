// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";

import { ZKRollup, IZKRollup } from "../L1/rollup/ZKRollup.sol";

contract ZKRollupTest is DSTestPlus {
  ZKRollup private rollup;

  function setUp() public {
    rollup = new ZKRollup();
    rollup.initialize(233);
  }

  function testInitialization() public {
    assertEq(address(this), rollup.owner());
    assertEq(rollup.layer2ChainId(), 233);
    assertEq(rollup.operator(), address(0));
    assertEq(rollup.messenger(), address(0));

    hevm.expectRevert("Initializable: contract is already initialized");
    rollup.initialize(555);
  }

  function testUpdateOperator(address _operator) public {
    if (_operator == address(0)) return;

    // set by non-owner, should revert
    hevm.startPrank(address(1));
    hevm.expectRevert("Ownable: caller is not the owner");
    rollup.updateOperator(_operator);
    hevm.stopPrank();

    // change to random operator
    rollup.updateOperator(_operator);
    assertEq(rollup.operator(), _operator);

    // set to same operator, should revert
    hevm.expectRevert("change to same operator");
    rollup.updateOperator(_operator);
  }

  function testUpdateMessenger(address _messenger) public {
    if (_messenger == address(0)) return;

    // set by non-owner, should revert
    hevm.startPrank(address(1));
    hevm.expectRevert("Ownable: caller is not the owner");
    rollup.updateMessenger(_messenger);
    hevm.stopPrank();

    // change to random messenger
    rollup.updateMessenger(_messenger);
    assertEq(rollup.messenger(), _messenger);

    // set to same messenger, should revert
    hevm.expectRevert("change to same messenger");
    rollup.updateMessenger(_messenger);
  }

  function testImportGenesisBlock(IZKRollup.Layer2BlockHeader memory _genesis) public {
    if (_genesis.blockHash == bytes32(0)) {
      _genesis.blockHash = bytes32(uint256(1));
    }
    _genesis.parentHash = bytes32(0);
    _genesis.blockHeight = 0;

    // set by non-owner, should revert
    hevm.startPrank(address(1));
    hevm.expectRevert("Ownable: caller is not the owner");
    rollup.importGenesisBlock(_genesis);
    hevm.stopPrank();

    // not genesis block, should revert
    _genesis.blockHeight = 1;
    hevm.expectRevert("Block is not genesis");
    rollup.importGenesisBlock(_genesis);
    _genesis.blockHeight = 0;

    // parent hash not empty, should revert
    _genesis.parentHash = bytes32(uint256(2));
    hevm.expectRevert("Parent hash not empty");
    rollup.importGenesisBlock(_genesis);
    _genesis.parentHash = bytes32(0);

    // invalid block hash, should revert
    bytes32 _originalHash = _genesis.blockHash;
    _genesis.blockHash = bytes32(0);
    hevm.expectRevert("Block hash is zero");
    rollup.importGenesisBlock(_genesis);
    _genesis.blockHash = _originalHash;

    // TODO: add Block hash verification failed

    // import correctly
    assertEq(rollup.finalizedBatches(0), bytes32(0));
    rollup.importGenesisBlock(_genesis);
    {
      (bytes32 parentHash, , uint64 blockHeight, uint64 batchIndex) = rollup.blocks(_genesis.blockHash);
      assertEq(_genesis.parentHash, parentHash);
      assertEq(_genesis.blockHeight, blockHeight);
      assertEq(batchIndex, 0);
    }
    {
      bytes32 _batchId = keccak256(abi.encode(_genesis.blockHash, bytes32(0), 0));
      assertEq(rollup.finalizedBatches(0), _batchId);
      assertEq(rollup.lastFinalizedBatchID(), _batchId);
      (bytes32 batchHash, bytes32 parentHash, uint64 batchIndex, bool verified) = rollup.batches(_batchId);
      assertEq(batchHash, _genesis.blockHash);
      assertEq(parentHash, bytes32(0));
      assertEq(batchIndex, 0);
      assertBoolEq(verified, true);
    }

    // genesis block imported
    hevm.expectRevert("Genesis block imported");
    rollup.importGenesisBlock(_genesis);
  }

  function testCommitBatchFailed() public {
    rollup.updateOperator(address(1));

    IZKRollup.Layer2BlockHeader memory _header;
    IZKRollup.Layer2Batch memory _batch;

    // not operator call, should revert
    hevm.expectRevert("caller not operator");
    rollup.commitBatch(_batch);

    // import fake genesis
    _header.blockHash = bytes32(uint256(1));
    rollup.importGenesisBlock(_header);

    hevm.startPrank(address(1));
    // batch is empty
    hevm.expectRevert("Batch is empty");
    rollup.commitBatch(_batch);

    // block submitted, should revert
    _header.blockHash = bytes32(uint256(1));
    _batch.blocks = new IZKRollup.Layer2BlockHeader[](1);
    _batch.blocks[0] = _header;
    _batch.batchIndex = 0;
    _batch.parentHash = bytes32(0);
    hevm.expectRevert("Batch has been committed before");
    rollup.commitBatch(_batch);

    // no parent batch, should revert
    _header.blockHash = bytes32(uint256(2));
    _batch.blocks = new IZKRollup.Layer2BlockHeader[](1);
    _batch.blocks[0] = _header;
    _batch.batchIndex = 0;
    _batch.parentHash = bytes32(0);
    hevm.expectRevert("Parent batch hasn't been committed");
    rollup.commitBatch(_batch);

    // Batch index and parent batch index mismatch
    _header.blockHash = bytes32(uint256(2));
    _batch.blocks = new IZKRollup.Layer2BlockHeader[](1);
    _batch.blocks[0] = _header;
    _batch.batchIndex = 2;
    _batch.parentHash = bytes32(uint256(1));
    hevm.expectRevert("Batch index and parent batch index mismatch");
    rollup.commitBatch(_batch);

    // BLock parent hash mismatch
    _header.blockHash = bytes32(uint256(2));
    _header.parentHash = bytes32(0);
    _batch.blocks = new IZKRollup.Layer2BlockHeader[](1);
    _batch.blocks[0] = _header;
    _batch.batchIndex = 1;
    _batch.parentHash = bytes32(uint256(1));
    hevm.expectRevert("Block parent hash mismatch");
    rollup.commitBatch(_batch);

    // Block height mismatch
    _header.blockHash = bytes32(uint256(2));
    _header.parentHash = bytes32(uint256(1));
    _header.blockHeight = 2;
    _batch.blocks = new IZKRollup.Layer2BlockHeader[](1);
    _batch.blocks[0] = _header;
    _batch.batchIndex = 1;
    _batch.parentHash = bytes32(uint256(1));
    hevm.expectRevert("Block height mismatch");
    rollup.commitBatch(_batch);

    _header.blockHash = bytes32(uint256(2));
    _header.parentHash = bytes32(uint256(1));
    _header.blockHeight = 0;
    _batch.blocks = new IZKRollup.Layer2BlockHeader[](1);
    _batch.blocks[0] = _header;
    _batch.batchIndex = 1;
    _batch.parentHash = bytes32(uint256(1));
    hevm.expectRevert("Block height mismatch");
    rollup.commitBatch(_batch);

    // Block has been commited before
    _header.blockHash = bytes32(uint256(1));
    _header.parentHash = bytes32(uint256(1));
    _header.blockHeight = 1;
    _batch.blocks = new IZKRollup.Layer2BlockHeader[](1);
    _batch.blocks[0] = _header;
    _batch.batchIndex = 1;
    _batch.parentHash = bytes32(uint256(1));
    hevm.expectRevert("Block has been commited before");
    rollup.commitBatch(_batch);

    hevm.stopPrank();
  }

  function testCommitBatch(IZKRollup.Layer2BlockHeader memory _header) public {
    if (_header.parentHash == bytes32(0)) {
      _header.parentHash = bytes32(uint256(1));
    }
    if (_header.blockHash == _header.parentHash) {
      return;
    }
    rollup.updateOperator(address(1));

    // import fake genesis
    IZKRollup.Layer2BlockHeader memory _genesis;
    _genesis.blockHash = _header.parentHash;
    rollup.importGenesisBlock(_genesis);
    _header.blockHeight = 1;

    IZKRollup.Layer2Batch memory _batch;
    _batch.blocks = new IZKRollup.Layer2BlockHeader[](1);
    _batch.blocks[0] = _header;
    _batch.batchIndex = 1;
    _batch.parentHash = _header.parentHash;

    // mock caller as operator
    assertEq(rollup.finalizedBatches(1), bytes32(0));
    hevm.startPrank(address(1));
    rollup.commitBatch(_batch);
    hevm.stopPrank();

    // verify block
    {
      (bytes32 parentHash, , uint64 blockHeight, uint64 batchIndex) = rollup.blocks(_header.blockHash);
      assertEq(parentHash, _header.parentHash);
      assertEq(blockHeight, _header.blockHeight);
      assertEq(batchIndex, _batch.batchIndex);
    }
    // verify batch
    {
      bytes32 _batchId = keccak256(abi.encode(_header.blockHash, _header.parentHash, 1));
      (bytes32 batchHash, bytes32 parentHash, uint64 batchIndex, bool verified) = rollup.batches(_batchId);
      assertEq(batchHash, _header.blockHash);
      assertEq(parentHash, _batch.parentHash);
      assertEq(batchIndex, _batch.batchIndex);
      assertBoolEq(verified, false);
      assertEq(rollup.finalizedBatches(1), bytes32(0));
    }
  }
}
