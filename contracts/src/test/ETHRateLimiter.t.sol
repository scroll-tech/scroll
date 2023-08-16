// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {ETHRateLimiter} from "../rate-limiter/ETHRateLimiter.sol";
import {IETHRateLimiter} from "../rate-limiter/IETHRateLimiter.sol";

contract ETHRateLimiterTest is DSTestPlus {
    event UpdateTotalLimit(uint256 oldTotalLimit, uint256 newTotalLimit);
    event UpdateDefaultUserLimit(uint256 oldDefaultUserLimit, uint256 newDefaultUserLimit);
    event UpdateCustomUserLimit(address indexed account, uint256 oldUserLimit, uint256 newUserLimit);

    ETHRateLimiter private limiter;

    event Log(address addr);

    function setUp() public {
        hevm.warp(86400);
        limiter = new ETHRateLimiter(86400, address(this), 100 ether, 100 ether);
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

    function testUpdateDefaultUserLimit(uint256 _newDefaultUserLimit) external {
        hevm.assume(_newDefaultUserLimit > 0);

        // not owner, revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        limiter.updateDefaultUserLimit(_newDefaultUserLimit);
        hevm.stopPrank();

        // zero revert
        hevm.expectRevert(IETHRateLimiter.DefaultUserLimitIsZero.selector);
        limiter.updateDefaultUserLimit(0);

        // success
        hevm.expectEmit(false, false, false, true);
        emit UpdateDefaultUserLimit(100 ether, _newDefaultUserLimit);
        limiter.updateDefaultUserLimit(_newDefaultUserLimit);
        assertEq(limiter.defaultUserLimit(), _newDefaultUserLimit);
    }

    function testUpdateCustomUserLimit(address _account, uint104 _newLimit) external {
        // not owner, revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        limiter.updateCustomUserLimit(_account, _newLimit);
        hevm.stopPrank();

        // success
        hevm.expectEmit(true, false, false, true);
        emit UpdateCustomUserLimit(_account, 0, _newLimit);
        limiter.updateCustomUserLimit(_account, _newLimit);
        if (_newLimit == 0) {
            assertEq(limiter.getUserCustomLimit(_account), limiter.defaultUserLimit());
        } else {
            assertEq(limiter.getUserCustomLimit(_account), _newLimit);
        }
    }

    function testAddUsedAmount() external {
        // non-spender, revert
        hevm.startPrank(address(1));
        hevm.expectRevert(IETHRateLimiter.CallerNotSpender.selector);
        limiter.addUsedAmount(address(0), 0);
        hevm.stopPrank();

        // exceed total limit on first call
        hevm.expectRevert(IETHRateLimiter.ExceedTotalLimit.selector);
        limiter.addUsedAmount(address(1), 100 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(0);

        // exceed total limit on second call
        limiter.addUsedAmount(address(1), 50 ether);
        _checkTotalCurrentPeriodAmountAmount(50 ether);
        hevm.expectRevert(IETHRateLimiter.ExceedTotalLimit.selector);
        limiter.addUsedAmount(address(1), 50 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(50 ether);

        // one period passed
        hevm.warp(86400 * 2);
        limiter.addUsedAmount(address(1), 1 ether);
        _checkTotalCurrentPeriodAmountAmount(1 ether);
        _checkUserCurrentPeriodAmountAmount(address(1), 1 ether);

        limiter.updateCustomUserLimit(address(1), 10 ether);

        // user limit exceed
        hevm.warp(86400 * 2 + 10);
        hevm.expectRevert(IETHRateLimiter.ExceedUserLimit.selector);
        limiter.addUsedAmount(address(1), 9 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(1 ether);
        _checkUserCurrentPeriodAmountAmount(address(1), 1 ether);

        // another period passed
        hevm.warp(86400 * 3);

        // user 1 add 1 ether
        hevm.warp(86400 * 3 + 1);
        limiter.addUsedAmount(address(1), 1 ether);
        _checkTotalCurrentPeriodAmountAmount(1 ether);
        _checkUserCurrentPeriodAmountAmount(address(1), 1 ether);

        // user 2 add 1 ether
        hevm.warp(86400 * 3 + 2);
        limiter.addUsedAmount(address(2), 1 ether);
        _checkTotalCurrentPeriodAmountAmount(2 ether);
        _checkUserCurrentPeriodAmountAmount(address(1), 1 ether);
        _checkUserCurrentPeriodAmountAmount(address(2), 1 ether);

        // user 1 exceed
        hevm.warp(86400 * 3 + 3);
        hevm.expectRevert(IETHRateLimiter.ExceedUserLimit.selector);
        limiter.addUsedAmount(address(1), 9 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(2 ether);
        _checkUserCurrentPeriodAmountAmount(address(1), 1 ether);
        _checkUserCurrentPeriodAmountAmount(address(2), 1 ether);

        // user 2 succeed
        hevm.warp(86400 * 3 + 4);
        limiter.addUsedAmount(address(2), 9 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(11 ether + 1);
        _checkUserCurrentPeriodAmountAmount(address(1), 1 ether);
        _checkUserCurrentPeriodAmountAmount(address(2), 10 ether + 1);
    }

    function _checkTotalCurrentPeriodAmountAmount(uint256 expected) internal {
        (, , uint256 totalAmount) = limiter.currentPeriod();
        assertEq(totalAmount, expected);
    }

    function _checkUserCurrentPeriodAmountAmount(address account, uint256 expected) internal {
        (, , uint256 userAmount) = limiter.userCurrentPeriod(account);
        assertEq(userAmount, expected);
    }
}
