// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";

import { L2ToL1MessagePasser } from "../L2/predeploys/L2ToL1MessagePasser.sol";

contract L2ToL1MessagePasserTest is DSTestPlus {
  L2ToL1MessagePasser passer;

  function setUp() public {
    passer = new L2ToL1MessagePasser(address(this));
  }

  function testConstructor() external {
    assertEq(passer.messenger(), address(this));
    assertEq(passer.nextMessageIndex(), 0);
  }

  function testPassMessageFailed() external {
    // not messenger
    hevm.startPrank(address(0));
    hevm.expectRevert("only messenger");
    passer.passMessageToL1(bytes32(0));
    hevm.stopPrank();

    // duplicated message
    passer.passMessageToL1(bytes32(0));
    hevm.expectRevert("duplicated message");
    passer.passMessageToL1(bytes32(0));
  }

  function testPassMessageOnceSuccess(bytes32 _message) external {
    passer.passMessageToL1(_message);
    assertEq(passer.nextMessageIndex(), 1);
    assertEq(passer.branches(0), _message);
    assertEq(passer.messageRoot(), _message);
  }

  function testPassMessageSuccess() external {
    passer.passMessageToL1(bytes32(uint256(1)));
    assertEq(passer.nextMessageIndex(), 1);
    assertEq(passer.branches(0), bytes32(uint256(1)));
    assertEq(passer.messageRoot(), bytes32(uint256(1)));

    passer.passMessageToL1(bytes32(uint256(2)));
    assertEq(passer.nextMessageIndex(), 2);
    assertEq(passer.branches(1), bytes32(uint256(0xe90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0)));
    assertEq(
      passer.messageRoot(),
      bytes32(uint256(0xe90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0))
    );

    passer.passMessageToL1(bytes32(uint256(3)));
    assertEq(passer.nextMessageIndex(), 3);
    assertEq(passer.branches(2), bytes32(uint256(0x222ff5e0b5877792c2bc1670e2ccd0c2c97cd7bb1672a57d598db05092d3d72c)));
    assertEq(
      passer.messageRoot(),
      bytes32(uint256(0x222ff5e0b5877792c2bc1670e2ccd0c2c97cd7bb1672a57d598db05092d3d72c))
    );

    passer.passMessageToL1(bytes32(uint256(4)));
    assertEq(passer.nextMessageIndex(), 4);
    assertEq(passer.branches(2), bytes32(uint256(0xa9bb8c3f1f12e9aa903a50c47f314b57610a3ab32f2d463293f58836def38d36)));
    assertEq(
      passer.messageRoot(),
      bytes32(uint256(0xa9bb8c3f1f12e9aa903a50c47f314b57610a3ab32f2d463293f58836def38d36))
    );
  }
}
