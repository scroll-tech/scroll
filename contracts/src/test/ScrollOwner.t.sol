// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {ScrollOwner} from "../misc/ScrollOwner.sol";

contract ScrollOwnerTest is DSTestPlus {
    event GrantAccess(bytes32 indexed role, address indexed target, bytes4[] selectors);
    event RevokeAccess(bytes32 indexed role, address indexed target, bytes4[] selectors);
    event Call();

    ScrollOwner private owner;

    function setUp() public {
        owner = new ScrollOwner();
    }

    function testUpdateAccess() external {
        // not admin, evert
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0x0000000000000000000000000000000000000000000000000000000000000000"
        );
        owner.updateAccess(address(0), new bytes4[](0), bytes32(0), true);
        hevm.stopPrank();

        bytes4[] memory _selectors;
        bytes32[] memory _roles;

        // add access then remove access
        _roles = owner.callableRoles(address(this), ScrollOwnerTest.revertOnCall.selector);
        assertEq(0, _roles.length);
        _selectors = new bytes4[](1);
        _selectors[0] = ScrollOwnerTest.revertOnCall.selector;

        hevm.expectEmit(true, true, false, true);
        emit GrantAccess(bytes32(uint256(1)), address(this), _selectors);

        owner.updateAccess(address(this), _selectors, bytes32(uint256(1)), true);
        _roles = owner.callableRoles(address(this), ScrollOwnerTest.revertOnCall.selector);
        assertEq(1, _roles.length);
        assertEq(_roles[0], bytes32(uint256(1)));

        hevm.expectEmit(true, true, false, true);
        emit RevokeAccess(bytes32(uint256(1)), address(this), _selectors);

        owner.updateAccess(address(this), _selectors, bytes32(uint256(1)), false);
        _roles = owner.callableRoles(address(this), ScrollOwnerTest.revertOnCall.selector);
        assertEq(0, _roles.length);
    }

    function testAdminExecute() external {
        // call with revert
        hevm.expectRevert("Called");
        owner.execute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.revertOnCall.selector), bytes32(0));

        // call with emit
        hevm.expectEmit(false, false, false, true);
        emit Call();
        owner.execute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.emitOnCall.selector), bytes32(0));
    }

    function testExecute(bytes32 _role) external {
        hevm.assume(_role != bytes32(0));

        bytes4[] memory _selectors = new bytes4[](2);
        _selectors[0] = ScrollOwnerTest.revertOnCall.selector;
        _selectors[1] = ScrollOwnerTest.emitOnCall.selector;

        owner.grantRole(_role, address(this));

        // no access, revert
        hevm.expectRevert("no access");
        owner.execute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.revertOnCall.selector), _role);

        owner.updateAccess(address(this), _selectors, _role, true);

        // call with revert
        hevm.expectRevert("Called");
        owner.execute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.revertOnCall.selector), _role);

        // call with emit
        hevm.expectEmit(false, false, false, true);
        emit Call();
        owner.execute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.emitOnCall.selector), _role);
    }

    function revertOnCall() external pure {
        revert("Called");
    }

    function emitOnCall() external {
        emit Call();
    }
}
