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

  function testImportGenesisBlock(IZKRollup.BlockHeader memory _genesis) public {
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
    hevm.expectRevert("not genesis block");
    rollup.importGenesisBlock(_genesis);
    _genesis.blockHeight = 0;

    // parent hash not empty, should revert
    _genesis.parentHash = bytes32(uint256(2));
    hevm.expectRevert("parent hash not empty");
    rollup.importGenesisBlock(_genesis);
    _genesis.parentHash = bytes32(0);

    // invalid block hash, should revert
    bytes32 _originalHash = _genesis.blockHash;
    _genesis.blockHash = bytes32(0);
    hevm.expectRevert("invalid block hash");
    rollup.importGenesisBlock(_genesis);
    _genesis.blockHash = _originalHash;

    // import correctly
    assertEq(rollup.finalizedBlocks(0), bytes32(0));
    rollup.importGenesisBlock(_genesis);
    (IZKRollup.BlockHeader memory _header, , bool _verified) = rollup.blocks(_genesis.blockHash);
    assertEq(_genesis.blockHash, rollup.lastFinalizedBlockHash());
    assertEq(_genesis.blockHash, _header.blockHash);
    assertEq(_genesis.parentHash, _header.parentHash);
    assertEq(_genesis.baseFee, _header.baseFee);
    assertEq(_genesis.stateRoot, _header.stateRoot);
    assertEq(_genesis.blockHeight, _header.blockHeight);
    assertEq(_genesis.gasUsed, _header.gasUsed);
    assertEq(_genesis.timestamp, _header.timestamp);
    assertBytesEq(_genesis.extraData, _header.extraData);
    assertBoolEq(_verified, true);
    assertEq(rollup.finalizedBlocks(0), _genesis.blockHash);

    // genesis block imported
    hevm.expectRevert("genesis block imported");
    rollup.importGenesisBlock(_genesis);
  }

  function testCommitBlockFailed() public {
    rollup.updateOperator(address(1));

    IZKRollup.BlockHeader memory _header;
    IZKRollup.Layer2Transaction[] memory _txns = new IZKRollup.Layer2Transaction[](0);

    // not operator call, should revert
    hevm.expectRevert("caller not operator");
    rollup.commitBlock(_header, _txns);

    // import fake genesis
    _header.blockHash = bytes32(uint256(1));
    rollup.importGenesisBlock(_header);

    hevm.startPrank(address(1));
    // block submitted, should revert
    _header.blockHash = bytes32(uint256(1));
    hevm.expectRevert("Block has been committed before");
    rollup.commitBlock(_header, _txns);

    // no parent block, should revert
    _header.blockHash = bytes32(uint256(2));
    hevm.expectRevert("Parent hasn't been committed");
    rollup.commitBlock(_header, _txns);

    // block height mismatch, should revert
    _header.blockHash = bytes32(uint256(2));
    _header.parentHash = bytes32(uint256(1));
    hevm.expectRevert("Block height and parent block height mismatch");
    rollup.commitBlock(_header, _txns);

    _header.blockHeight = 2;
    hevm.expectRevert("Block height and parent block height mismatch");
    rollup.commitBlock(_header, _txns);
    hevm.stopPrank();
  }

  function testCommitBlock(IZKRollup.BlockHeader memory _header) public {
    if (_header.parentHash == bytes32(0)) {
      _header.parentHash = bytes32(uint256(1));
    }
    if (_header.blockHash == _header.parentHash) {
      return;
    }
    rollup.updateOperator(address(1));

    IZKRollup.Layer2Transaction[] memory _txns = new IZKRollup.Layer2Transaction[](0);

    // import fake genesis
    IZKRollup.BlockHeader memory _genesis;
    _genesis.blockHash = _header.parentHash;
    rollup.importGenesisBlock(_genesis);
    _header.blockHeight = 1;

    // mock caller as operator
    assertEq(rollup.finalizedBlocks(1), bytes32(0));
    hevm.startPrank(address(1));
    rollup.commitBlock(_header, _txns);
    hevm.stopPrank();

    (IZKRollup.BlockHeader memory _storedHeader, , bool _verified) = rollup.blocks(_header.blockHash);
    assertEq(_header.blockHash, _storedHeader.blockHash);
    assertEq(_header.parentHash, _storedHeader.parentHash);
    assertEq(_header.baseFee, _storedHeader.baseFee);
    assertEq(_header.stateRoot, _storedHeader.stateRoot);
    assertEq(_header.blockHeight, _storedHeader.blockHeight);
    assertEq(_header.gasUsed, _storedHeader.gasUsed);
    assertEq(_header.timestamp, _storedHeader.timestamp);
    assertBytesEq(_header.extraData, _storedHeader.extraData);
    assertBoolEq(_verified, false);
    assertEq(rollup.finalizedBlocks(1), bytes32(0));
  }
}
