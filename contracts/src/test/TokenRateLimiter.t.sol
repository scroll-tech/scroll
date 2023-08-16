// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {TokenRateLimiter} from "../rate-limiter/TokenRateLimiter.sol";
import {ITokenRateLimiter} from "../rate-limiter/ITokenRateLimiter.sol";

contract TokenRateLimiterTest is DSTestPlus {
    event UpdateTotalLimit(address indexed token, uint256 oldTotalLimit, uint256 newTotalLimit);
    event UpdateDefaultUserLimit(address indexed token, uint256 oldDefaultUserLimit, uint256 newDefaultUserLimit);
    event UpdateCustomUserLimit(
        address indexed token,
        address indexed account,
        uint256 oldUserLimit,
        uint256 newUserLimit
    );

    TokenRateLimiter private limiter;

    event Log(address addr);

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

    function testUpdateDefaultUserLimit(address _token, uint256 _newDefaultUserLimit) external {
        hevm.assume(_newDefaultUserLimit > 0);

        // not owner, revert
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0x0000000000000000000000000000000000000000000000000000000000000000"
        );
        limiter.updateDefaultUserLimit(_token, _newDefaultUserLimit);
        hevm.stopPrank();

        // zero revert
        hevm.expectRevert(abi.encodeWithSelector(ITokenRateLimiter.DefaultUserLimitIsZero.selector, _token));
        limiter.updateDefaultUserLimit(_token, 0);

        // success
        hevm.expectEmit(true, false, false, true);
        emit UpdateDefaultUserLimit(_token, 0 ether, _newDefaultUserLimit);
        limiter.updateDefaultUserLimit(_token, _newDefaultUserLimit);
        assertEq(limiter.defaultUserLimit(_token), _newDefaultUserLimit);
    }

    function testUpdateCustomUserLimit(
        address _token,
        address _account,
        uint104 _newLimit
    ) external {
        // not owner, revert
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0x0000000000000000000000000000000000000000000000000000000000000000"
        );
        limiter.updateCustomUserLimit(_token, _account, _newLimit);
        hevm.stopPrank();

        // success
        hevm.expectEmit(true, true, false, true);
        emit UpdateCustomUserLimit(_token, _account, 0, _newLimit);
        limiter.updateCustomUserLimit(_token, _account, _newLimit);
        if (_newLimit == 0) {
            assertEq(limiter.getUserCustomLimit(_token, _account), limiter.defaultUserLimit(_token));
        } else {
            assertEq(limiter.getUserCustomLimit(_token, _account), _newLimit);
        }
    }

    function testAddUsedAmount(address _token) external {
        // non-spender, revert
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0x267f05081a073059ae452e6ac77ec189636e43e41051d4c3ec760734b3d173cb"
        );
        limiter.addUsedAmount(_token, address(0), 0);
        hevm.stopPrank();

        limiter.grantRole(bytes32(0x267f05081a073059ae452e6ac77ec189636e43e41051d4c3ec760734b3d173cb), address(this));
        limiter.updateTotalLimit(_token, 100 ether);

        // exceed total limit on first call
        hevm.expectRevert(abi.encodeWithSelector(ITokenRateLimiter.ExceedTotalLimit.selector, _token));
        limiter.addUsedAmount(_token, address(1), 100 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(_token, 0);

        // exceed total limit on second call
        limiter.addUsedAmount(_token, address(1), 50 ether);
        _checkTotalCurrentPeriodAmountAmount(_token, 50 ether);
        hevm.expectRevert(abi.encodeWithSelector(ITokenRateLimiter.ExceedTotalLimit.selector, _token));
        limiter.addUsedAmount(_token, address(1), 50 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(_token, 50 ether);

        // one period passed
        hevm.warp(86400 * 2);
        limiter.addUsedAmount(_token, address(1), 1 ether);
        _checkTotalCurrentPeriodAmountAmount(_token, 1 ether);
        _checkUserCurrentPeriodAmountAmount(_token, address(1), 1 ether);

        limiter.updateCustomUserLimit(_token, address(1), 10 ether);

        // user limit exceed
        hevm.warp(86400 * 2 + 10);
        hevm.expectRevert(abi.encodeWithSelector(ITokenRateLimiter.ExceedUserLimit.selector, _token));
        limiter.addUsedAmount(_token, address(1), 9 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(_token, 1 ether);
        _checkUserCurrentPeriodAmountAmount(_token, address(1), 1 ether);

        // another period passed
        hevm.warp(86400 * 3);

        // user 1 add 1 ether
        hevm.warp(86400 * 3 + 1);
        limiter.addUsedAmount(_token, address(1), 1 ether);
        _checkTotalCurrentPeriodAmountAmount(_token, 1 ether);
        _checkUserCurrentPeriodAmountAmount(_token, address(1), 1 ether);

        // user 2 add 1 ether
        hevm.warp(86400 * 3 + 2);
        limiter.addUsedAmount(_token, address(2), 1 ether);
        _checkTotalCurrentPeriodAmountAmount(_token, 2 ether);
        _checkUserCurrentPeriodAmountAmount(_token, address(1), 1 ether);
        _checkUserCurrentPeriodAmountAmount(_token, address(2), 1 ether);

        // user 1 exceed
        hevm.warp(86400 * 3 + 3);
        hevm.expectRevert(abi.encodeWithSelector(ITokenRateLimiter.ExceedUserLimit.selector, _token));
        limiter.addUsedAmount(_token, address(1), 9 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(_token, 2 ether);
        _checkUserCurrentPeriodAmountAmount(_token, address(1), 1 ether);
        _checkUserCurrentPeriodAmountAmount(_token, address(2), 1 ether);

        // user 2 succeed
        hevm.warp(86400 * 3 + 4);
        limiter.addUsedAmount(_token, address(2), 9 ether + 1);
        _checkTotalCurrentPeriodAmountAmount(_token, 11 ether + 1);
        _checkUserCurrentPeriodAmountAmount(_token, address(1), 1 ether);
        _checkUserCurrentPeriodAmountAmount(_token, address(2), 10 ether + 1);
    }

    function _checkTotalCurrentPeriodAmountAmount(address token, uint256 expected) internal {
        (, , uint256 totalAmount) = limiter.currentPeriod(token);
        assertEq(totalAmount, expected);
    }

    function _checkUserCurrentPeriodAmountAmount(
        address token,
        address account,
        uint256 expected
    ) internal {
        (, , uint256 userAmount) = limiter.userCurrentPeriod(token, account);
        assertEq(userAmount, expected);
    }
}
