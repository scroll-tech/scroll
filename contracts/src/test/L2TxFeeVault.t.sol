// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {L2TxFeeVault} from "../L2/predeploys/L2TxFeeVault.sol";

contract L2TxFeeVaultTest is DSTestPlus {
    // events
    event UpdateMessenger(address indexed oldMessenger, address indexed newMessenger);
    event UpdateRecipient(address indexed oldRecipient, address indexed newRecipient);
    event UpdateMinWithdrawAmount(uint256 oldMinWithdrawAmount, uint256 newMinWithdrawAmount);

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

    function testUpdateMinWithdrawAmount(uint256 amount1, uint256 amount2) external {
        // set by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("caller is not the owner");
        vault.updateMinWithdrawAmount(amount1);
        hevm.stopPrank();

        // set by owner, should succeed
        assertEq(10 ether, vault.minWithdrawAmount());

        hevm.expectEmit(false, false, false, true);
        emit UpdateMinWithdrawAmount(10 ether, amount1);
        vault.updateMinWithdrawAmount(amount1);
        assertEq(amount1, vault.minWithdrawAmount());

        hevm.expectEmit(false, false, false, true);
        emit UpdateMinWithdrawAmount(amount1, amount2);
        vault.updateMinWithdrawAmount(amount2);
        assertEq(amount2, vault.minWithdrawAmount());
    }

    function testUpdateRecipient(address recipient1, address recipient2) external {
        // set by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("caller is not the owner");
        vault.updateRecipient(recipient1);
        hevm.stopPrank();

        // set by owner, should succeed
        assertEq(address(1), vault.recipient());

        hevm.expectEmit(true, true, false, true);
        emit UpdateRecipient(address(1), recipient1);
        vault.updateRecipient(recipient1);
        assertEq(recipient1, vault.recipient());

        hevm.expectEmit(true, true, false, true);
        emit UpdateRecipient(recipient1, recipient2);
        vault.updateRecipient(recipient2);
        assertEq(recipient2, vault.recipient());
    }

    function testUpdateMessenger(address messenger1, address messenger2) external {
        // set by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("caller is not the owner");
        vault.updateMessenger(messenger1);
        hevm.stopPrank();

        // set by owner, should succeed
        assertEq(address(messenger), vault.messenger());

        hevm.expectEmit(true, true, false, true);
        emit UpdateMessenger(address(messenger), messenger1);
        vault.updateMessenger(messenger1);
        assertEq(messenger1, vault.messenger());

        hevm.expectEmit(true, true, false, true);
        emit UpdateMessenger(messenger1, messenger2);
        vault.updateMessenger(messenger2);
        assertEq(messenger2, vault.messenger());
    }
}
