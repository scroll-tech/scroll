// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";
import {WETH} from "solmate/tokens/WETH.sol";

import {Forwarder} from "../misc/Forwarder.sol";
import {MockTarget} from "../mocks/MockTarget.sol";
import {IL1ScrollMessenger, L1ScrollMessenger} from "../L1/L1ScrollMessenger.sol";


contract ForwarderTest is DSTestPlus {
    MockTarget public target;
    Forwarder public forwarder;
    L1ScrollMessenger internal l1Messenger;

    address public admin = address(2);
    address public superAdmin = address(3);

    function setUp() public {
        target = new MockTarget();
        forwarder = new Forwarder(admin, superAdmin);

        l1Messenger = new L1ScrollMessenger();
        l1Messenger.initialize(address(0), address(0), address(0), address(0));
        l1Messenger.transferOwnership(address(forwarder));
    }

    function testAdminFail() external {
        hevm.expectRevert("only admin or superAdmin");
        forwarder.forward(address(l1Messenger),hex"00");

        hevm.expectRevert("only superAdmin");
        forwarder.setAdmin(address(0));

        hevm.expectRevert("only superAdmin");
        forwarder.setSuperAdmin(address(0));
    }

    function testAdmin() external {
        // cast calldata "transferOwnership(address)" 0x0000000000000000000000000000000000000005
        // 0xf2fde38b0000000000000000000000000000000000000000000000000000000000000005

        hevm.startPrank(admin);
        forwarder.forward(address(l1Messenger), hex"f2fde38b0000000000000000000000000000000000000000000000000000000000000006");
        assertEq(address(6), l1Messenger.owner());
        hevm.stopPrank();
    }

    function testForwardSuperAdmin() external {
        hevm.startPrank(superAdmin);
        forwarder.forward(address(l1Messenger), hex"f2fde38b0000000000000000000000000000000000000000000000000000000000000006");
        assertEq(address(6), l1Messenger.owner());

        forwarder.setAdmin(address(0));
        assertEq(forwarder.admin(), address(0));
        

        forwarder.setSuperAdmin(address(0));
        assertEq(forwarder.superAdmin(), address(0));
    }
    
    function testNestedRevert() external {
        hevm.startPrank(superAdmin);
        hevm.expectRevert("test error");
        forwarder.forward(address(target), hex"38df7677");
    }
}
