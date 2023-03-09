// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";

import { L1BlockContainer } from "../L2/predeploys/L1BlockContainer.sol";
import { L1GasPriceOracle } from "../L2/predeploys/L1GasPriceOracle.sol";
import { L2MessageQueue } from "../L2/predeploys/L2MessageQueue.sol";
import { Whitelist } from "../L2/predeploys/Whitelist.sol";
import { L1ScrollMessenger } from "../L1/L1ScrollMessenger.sol";
import { L2ScrollMessenger } from "../L2/L2ScrollMessenger.sol";

contract L2ScrollMessengerTest is DSTestPlus {
  L1ScrollMessenger internal l1Messenger;

  address internal feeVault;
  Whitelist private whitelist;

  L2ScrollMessenger internal l2Messenger;
  L1BlockContainer internal l1BlockContainer;
  L2MessageQueue internal l2MessageQueue;
  L1GasPriceOracle internal l1GasOracle;

  function setUp() public {
    // Deploy L1 contracts
    l1Messenger = new L1ScrollMessenger();

    // Deploy L2 contracts
    whitelist = new Whitelist(address(this));
    l1BlockContainer = new L1BlockContainer(address(this));
    l2MessageQueue = new L2MessageQueue(address(this));
    l1GasOracle = new L1GasPriceOracle(address(this));
    l2Messenger = new L2ScrollMessenger(address(l1BlockContainer), address(l1GasOracle), address(l2MessageQueue));

    // Initialize L2 contracts
    l2Messenger.initialize(address(l1Messenger), feeVault);
    l2MessageQueue.initialize();
    l2MessageQueue.updateMessenger(address(l2Messenger));
    l1GasOracle.updateWhitelist(address(whitelist));
  }

  function testForbidCallFromL1() external {
    hevm.expectRevert("Forbid to call message queue");
    l2Messenger.relayMessage(address(this), address(l2MessageQueue), 0, 0, new bytes(0));

    hevm.expectRevert("Forbid to call self");
    l2Messenger.relayMessage(address(this), address(l2Messenger), 0, 0, new bytes(0));
  }
}
