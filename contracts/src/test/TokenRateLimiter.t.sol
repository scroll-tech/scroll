// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {TokenRateLimiter} from "../rate-limiter/TokenRateLimiter.sol";
import {ITokenRateLimiter} from "../rate-limiter/ITokenRateLimiter.sol";

contract TokenRateLimiterTest is DSTestPlus {
    event UpdateTotalLimit(address indexed token, uint256 oldTotalLimit, uint256 newTotalLimit);

    TokenRateLimiter private limiter;

    function setUp() public {
        hevm.warp(86400);
        limiter = new TokenRateLimiter(86400);
    }

    function testUpdateTotalLimit(address _token, uint104 _newTotalLimit) external {
        hevm.assume(_newTotalLimit > 0);

        // not admin, revert
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0x0000000000000000000000000000000000000000000000000000000000000000"
        );
        limiter.updateTotalLimit(_token, _newTotalLimit);
        hevm.stopPrank();

        // zero revert
        hevm.expectRevert(abi.encodeWithSelector(ITokenRateLimiter.TotalLimitIsZero.selector, _token));
        limiter.updateTotalLimit(_token, 0);

        // success
        hevm.expectEmit(true, false, false, true);
        emit UpdateTotalLimit(_token, 0 ether, _newTotalLimit);
        limiter.updateTotalLimit(_token, _newTotalLimit);
        (, uint104 _totalLimit, ) = limiter.currentPeriod(_token);
        assertEq(_totalLimit, _newTotalLimit);
    }

    function testAddUsedAmount(address _token) external {
        // non-spender, revert
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0x267f05081a073059ae452e6ac77ec189636e43e41051d4c3ec760734b3d173cb"
        );
        limiter.addUsedAmount(_token, 0);
        hevm.stopPrank();

        limiter.grantRole(bytes32(0x267f05081a073059ae452e6ac77ec189636e43e41051d4c3ec760734b3d173cb), address(this));
        limiter.updateTotalLimit(_token, 100 ether);

        // exceed total limit on first call
        hevm.expectRevert(abi.encodeWithSelector(ITokenRateLimiter.ExceedTotalLimit.selector, _token));
        limiter.addUsedAmount(_token, 100 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(_token, 0);

        // exceed total limit on second call
        limiter.addUsedAmount(_token, 50 ether);
        _checkTotalCurrentPeriodAmountAmount(_token, 50 ether);
        hevm.expectRevert(abi.encodeWithSelector(ITokenRateLimiter.ExceedTotalLimit.selector, _token));
        limiter.addUsedAmount(_token, 50 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(_token, 50 ether);

        // one period passed
        hevm.warp(86400 * 2);
        limiter.addUsedAmount(_token, 1 ether);
        _checkTotalCurrentPeriodAmountAmount(_token, 1 ether);

        // exceed
        hevm.expectRevert(abi.encodeWithSelector(ITokenRateLimiter.ExceedTotalLimit.selector, _token));
        limiter.addUsedAmount(_token, 99 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(_token, 1 ether);
    }

    function _checkTotalCurrentPeriodAmountAmount(address token, uint256 expected) internal {
        (, , uint256 totalAmount) = limiter.currentPeriod(token);
        assertEq(totalAmount, expected);
    }
}
