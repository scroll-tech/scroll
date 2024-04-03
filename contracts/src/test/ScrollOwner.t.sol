// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";
import { ScrollOwner } from "../misc/ScrollOwner.sol";

contract ScrollOwnerTest is DSTestPlus {
    event GrantAccess(bytes32 indexed role, address indexed target, bytes4[] selectors);
    event RevokeAccess(bytes32 indexed role, address indexed target, bytes4[] selectors);
    event Call();

    ScrollOwner private owner;

    function setUp() public {
        owner = new ScrollOwner();
    }

    function testUpdateAccess() external {
        bytes4[] memory _selectors;
        bytes32[] memory _roles;

        // Add access then remove access
        _roles = owner.callableRoles(address(this), ScrollOwnerTest.revertOnCall.selector);
        assertEq(0, _roles.length);
        _selectors = new bytes4 ;
        _selectors[0] = ScrollOwnerTest.revertOnCall.selector;

        emit GrantAccess(bytes32(uint256(1)), address(this), _selectors);
        owner.updateAccess(address(this), _selectors, bytes32(uint256(1)), true);
        _roles = owner.callableRoles(address(this), ScrollOwnerTest.revertOnCall.selector);
        assertEq(1, _roles.length);
        assertEq(_roles[0], bytes32(uint256(1)));

        emit RevokeAccess(bytes32(uint256(1)), address(this), _selectors);
        owner.updateAccess(address(this), _selectors, bytes32(uint256(1)), false);
        _roles = owner.callableRoles(address(this), ScrollOwnerTest.revertOnCall.selector);
        assertEq(0, _roles.length);
    }

    function testAdminExecute() external {
        // Call with revert
        hevm.expectRevert("Called");
        owner.execute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.revertOnCall.selector), bytes32(0));

        // Call with emit
        emit Call();
        owner.execute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.emitOnCall.selector), bytes32(0));
    }

    function testExecute(bytes32 _role) external {
        hevm.assume(_role != bytes32(0));

        bytes4[] memory _selectors = new bytes4[](2);
        _selectors[0] = ScrollOwnerTest.revertOnCall.selector;
        _selectors[1] = ScrollOwnerTest.emitOnCall.selector;

        owner.grantRole(_role, address(this));

        // No access, revert
        hevm.expectRevert("no access");
        owner.execute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.revertOnCall.selector), _role);

        owner.updateAccess(address(this), _selectors, _role, true);

        // Call with revert
        hevm.expectRevert("Called");
        owner.execute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.revertOnCall.selector), _role);

        // Call with emit
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
