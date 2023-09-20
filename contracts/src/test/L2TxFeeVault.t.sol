// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {L2TxFeeVault} from "../L2/predeploys/L2TxFeeVault.sol";

contract L2TxFeeVaultTest is DSTestPlus {
    MockScrollMessenger private messenger;
    L2TxFeeVault private vault;

    function setUp() public {
        messenger = new MockScrollMessenger();
        vault = new L2TxFeeVault(address(this), address(1), 10 ether);
        vault.updateMessenger(address(messenger));
    }

    function testCantWithdrawBelowMinimum() public {
        hevm.deal(address(vault), 9 ether);
        hevm.expectRevert("FeeVault: withdrawal amount must be greater than minimum withdrawal amount");
        vault.withdraw();
    }

    function testCantWithdrawAmountBelowMinimum(uint256 amount) public {
        amount = bound(amount, 0 ether, 10 ether - 1);
        hevm.deal(address(vault), 100 ether);
        hevm.expectRevert("FeeVault: withdrawal amount must be greater than minimum withdrawal amount");
        vault.withdraw(amount);
    }

    function testCantWithdrawMoreThanBalance(uint256 amount) public {
        hevm.assume(amount >= 10 ether);
        hevm.deal(address(vault), amount - 1);
        hevm.expectRevert("FeeVault: insufficient balance to withdraw");
        vault.withdraw(amount);
    }

    function testWithdrawOnce() public {
        hevm.deal(address(vault), 11 ether);
        vault.withdraw();
        assertEq(address(messenger).balance, 11 ether);
        assertEq(vault.totalProcessed(), 11 ether);
    }

    function testWithdrawAmountOnce(uint256 amount) public {
        amount = bound(amount, 10 ether, 100 ether);

        hevm.deal(address(vault), 100 ether);
        vault.withdraw(amount);

        assertEq(address(messenger).balance, amount);
        assertEq(vault.totalProcessed(), amount);
        assertEq(address(vault).balance, 100 ether - amount);
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

    function testWithdrawAmountTwice(uint256 amount1, uint256 amount2) public {
        amount1 = bound(amount1, 10 ether, 100 ether);
        amount2 = bound(amount2, 10 ether, 100 ether);

        hevm.deal(address(vault), 200 ether);

        vault.withdraw(amount1);
        assertEq(address(messenger).balance, amount1);
        assertEq(vault.totalProcessed(), amount1);

        vault.withdraw(amount2);
        assertEq(address(messenger).balance, amount1 + amount2);
        assertEq(vault.totalProcessed(), amount1 + amount2);

        assertEq(address(vault).balance, 200 ether - amount1 - amount2);
    }
}
