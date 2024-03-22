// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {ETHRateLimiter} from "../rate-limiter/ETHRateLimiter.sol";
import {IETHRateLimiter} from "../rate-limiter/IETHRateLimiter.sol";

contract ETHRateLimiterTest is DSTestPlus {
    event UpdateTotalLimit(uint256 oldTotalLimit, uint256 newTotalLimit);

    ETHRateLimiter private limiter;

    function setUp() public {
        hevm.warp(86400);
        limiter = new ETHRateLimiter(86400, address(this), 100 ether);
    }

    function testUpdateTotalLimit(uint104 _newTotalLimit) external {
        hevm.assume(_newTotalLimit > 0);

        // not owner, revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        limiter.updateTotalLimit(_newTotalLimit);
        hevm.stopPrank();

        // zero revert
        hevm.expectRevert(IETHRateLimiter.TotalLimitIsZero.selector);
        limiter.updateTotalLimit(0);

        // success
        hevm.expectEmit(false, false, false, true);
        emit UpdateTotalLimit(100 ether, _newTotalLimit);
        limiter.updateTotalLimit(_newTotalLimit);
        (, uint104 _totalLimit, ) = limiter.currentPeriod();
        assertEq(_totalLimit, _newTotalLimit);
    }

    function testAddUsedAmount() external {
        // non-spender, revert
        hevm.startPrank(address(1));
        hevm.expectRevert(IETHRateLimiter.CallerNotSpender.selector);
        limiter.addUsedAmount(0);
        hevm.stopPrank();

        // exceed total limit on first call
        hevm.expectRevert(IETHRateLimiter.ExceedTotalLimit.selector);
        limiter.addUsedAmount(100 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(0);

        // exceed total limit on second call
        limiter.addUsedAmount(50 ether);
        _checkTotalCurrentPeriodAmountAmount(50 ether);
        hevm.expectRevert(IETHRateLimiter.ExceedTotalLimit.selector);
        limiter.addUsedAmount(50 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(50 ether);

        // one period passed
        hevm.warp(86400 * 2);
        limiter.addUsedAmount(1 ether);
        _checkTotalCurrentPeriodAmountAmount(1 ether);

        // exceed
        hevm.expectRevert(IETHRateLimiter.ExceedTotalLimit.selector);
        limiter.addUsedAmount(99 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(1 ether);
    }

    function _checkTotalCurrentPeriodAmountAmount(uint256 expected) internal {
        (, , uint256 totalAmount) = limiter.currentPeriod();
        assertEq(totalAmount, expected);
    }
}
