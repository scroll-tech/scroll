// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {ScrollOwner} from "../misc/ScrollOwner.sol";

contract ScrollOwnerTest is DSTestPlus {
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
        _roles = owner.callable(address(this), ScrollOwnerTest.revertOnCall.selector);
        assertEq(0, _roles.length);
        _selectors = new bytes4[](1);
        _selectors[0] = ScrollOwnerTest.revertOnCall.selector;
        owner.updateAccess(address(this), _selectors, owner.OWNER_ROLE(), true);
        _roles = owner.callable(address(this), ScrollOwnerTest.revertOnCall.selector);
        assertEq(1, _roles.length);
        assertEq(_roles[0], owner.OWNER_ROLE());
        owner.updateAccess(address(this), _selectors, owner.OWNER_ROLE(), false);
        _roles = owner.callable(address(this), ScrollOwnerTest.revertOnCall.selector);
        assertEq(0, _roles.length);
    }

    function testOwnerExecute() external {
        bytes4[] memory _selectors = new bytes4[](2);
        _selectors[0] = ScrollOwnerTest.revertOnCall.selector;
        _selectors[1] = ScrollOwnerTest.emitOnCall.selector;

        // not owner, revert
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0x929f3fd6848015f83b9210c89f7744e3941acae1195c8bf9f5798c090dc8f497"
        );
        owner.ownerExecute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.revertOnCall.selector));
        hevm.stopPrank();

        owner.grantRole(owner.OWNER_ROLE(), address(this));

        // no access, revert
        hevm.expectRevert("no access");
        owner.ownerExecute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.revertOnCall.selector));

        owner.updateAccess(address(this), _selectors, owner.OWNER_ROLE(), true);

        // call with revert
        hevm.expectRevert("Called");
        owner.ownerExecute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.revertOnCall.selector));

        // call with emit
        hevm.expectEmit(false, false, false, true);
        emit Call();
        owner.ownerExecute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.emitOnCall.selector));
    }

    function testGovernaceExecute() external {
        bytes4[] memory _selectors = new bytes4[](2);
        _selectors[0] = ScrollOwnerTest.revertOnCall.selector;
        _selectors[1] = ScrollOwnerTest.emitOnCall.selector;

        // not owner, revert
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0x9409903de1e6fd852dfc61c9dacb48196c48535b60e25abf92acc92dd689078d"
        );
        owner.governaceExecute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.revertOnCall.selector));
        hevm.stopPrank();

        owner.grantRole(owner.GOVERNANCE_ROLE(), address(this));

        // no access, revert
        hevm.expectRevert("no access");
        owner.governaceExecute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.revertOnCall.selector));

        owner.updateAccess(address(this), _selectors, owner.GOVERNANCE_ROLE(), true);

        // call with revert
        hevm.expectRevert("Called");
        owner.governaceExecute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.revertOnCall.selector));

        // call with emit
        hevm.expectEmit(false, false, false, true);
        emit Call();
        owner.governaceExecute(address(this), 0, abi.encodeWithSelector(ScrollOwnerTest.emitOnCall.selector));
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
