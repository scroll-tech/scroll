// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {L2TxFeeVault} from "../L2/predeploys/L2TxFeeVault.sol";

contract L2TxFeeVaultTest is DSTestPlus {
    MockScrollMessenger private messenger;
    L2TxFeeVault private vault;

    function setUp() public {
        messenger = new MockScrollMessenger();
        vault = new L2TxFeeVault(address(this), address(1));
        vault.updateMessenger(address(messenger));
    }

    function testCantWithdrawBelowMinimum() public {
        hevm.deal(address(vault), 9 ether);
        hevm.expectRevert("FeeVault: withdrawal amount must be greater than minimum withdrawal amount");
        vault.withdraw();
    }

    function testWithdrawOnce() public {
        hevm.deal(address(vault), 11 ether);
        vault.withdraw();
        assertEq(address(messenger).balance, 11 ether);
        assertEq(vault.totalProcessed(), 11 ether);
    }

    function testWithdrawTwice() public {
        hevm.deal(address(vault), 11 ether);
        vault.withdraw();
        assertEq(address(messenger).balance, 11 ether);
        assertEq(vault.totalProcessed(), 11 ether);

        hevm.deal(address(vault), 22 ether);
        vault.withdraw();
        assertEq(address(messenger).balance, 33 ether);
        assertEq(vault.totalProcessed(), 33 ether);
    }
}
