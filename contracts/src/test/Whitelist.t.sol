// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";
import {WETH} from "solmate/tokens/WETH.sol";

import {Whitelist} from "../L2/predeploys/Whitelist.sol";

contract WhitelistTest is DSTestPlus {
    Whitelist private whitelist;

    function setUp() public {
        whitelist = new Whitelist(address(this));
    }

    function testRenounceOwnership() external {
        // call by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("caller is not the owner");
        whitelist.renounceOwnership();
        hevm.stopPrank();

        // call by owner, should succeed
        assertEq(whitelist.owner(), address(this));
        whitelist.renounceOwnership();
        assertEq(whitelist.owner(), address(0));
    }

    function testTransferOwnership(address _to) external {
        // call by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("caller is not the owner");
        whitelist.transferOwnership(_to);
        hevm.stopPrank();

        // call by owner, should succeed
        if (_to == address(0)) {
            hevm.expectRevert("new owner is the zero address");
            whitelist.transferOwnership(_to);
        } else {
            assertEq(whitelist.owner(), address(this));
            whitelist.transferOwnership(_to);
            assertEq(whitelist.owner(), _to);
        }
    }

    function testUpdateWhitelistStatus(address _to) external {
        address[] memory _accounts = new address[](1);
        _accounts[0] = _to;
        // call by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("caller is not the owner");
        whitelist.updateWhitelistStatus(_accounts, true);
        hevm.stopPrank();

        // call by owner, should succeed
        assertBoolEq(whitelist.isSenderAllowed(_to), false);
        whitelist.updateWhitelistStatus(_accounts, true);
        assertBoolEq(whitelist.isSenderAllowed(_to), true);
        whitelist.updateWhitelistStatus(_accounts, false);
        assertBoolEq(whitelist.isSenderAllowed(_to), false);
    }
}
