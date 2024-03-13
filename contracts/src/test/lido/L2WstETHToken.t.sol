// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {IERC1271Upgradeable} from "@openzeppelin/contracts-upgradeable/interfaces/IERC1271Upgradeable.sol";
import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {L2WstETHToken} from "../../lido/L2WstETHToken.sol";

contract L2WstETHTokenTest is DSTestPlus {
    L2WstETHToken private counterpart;
    L2WstETHToken private token;

    bytes4 private _magicValue;
    bool private revertOnSignature;

    function setUp() public {
        hevm.warp(1000); // make block timestamp nonzero

        counterpart = new L2WstETHToken();
        token = L2WstETHToken(address(new ERC1967Proxy(address(new L2WstETHToken()), new bytes(0))));

        token.initialize("Wrapped liquid staked Ether 2.0", "wstETH", 18, address(this), address(counterpart));
    }

    function testInitialize() external {
        assertEq(token.name(), "Wrapped liquid staked Ether 2.0");
        assertEq(token.symbol(), "wstETH");
        assertEq(token.decimals(), 18);
        assertEq(token.gateway(), address(this));
        assertEq(token.counterpart(), address(counterpart));
    }

    function testPermit(uint256 amount) external {
        uint256 timestamp = block.timestamp;
        // revert when expire
        hevm.expectRevert("ERC20Permit: expired deadline");
        token.permit(address(this), address(counterpart), 1, timestamp - 1, 0, 0, 0);

        // revert when invalid contract signature
        hevm.expectRevert("ERC20Permit: invalid signature");
        _magicValue = bytes4(0);
        revertOnSignature = false;
        token.permit(address(this), address(counterpart), 1, timestamp, 0, 0, 0);

        // revert when invalid contract signature
        hevm.expectRevert("ERC20Permit: invalid signature");
        _magicValue = IERC1271Upgradeable.isValidSignature.selector;
        revertOnSignature = true;
        token.permit(address(this), address(counterpart), 1, timestamp, 0, 0, 0);

        // succeed on contract signer
        _magicValue = IERC1271Upgradeable.isValidSignature.selector;
        revertOnSignature = false;
        assertEq(token.allowance(address(this), address(counterpart)), 0);
        token.permit(address(this), address(counterpart), amount, timestamp, 0, 0, 0);
        assertEq(token.allowance(address(this), address(counterpart)), amount);
    }

    function isValidSignature(bytes32 hash, bytes memory signature) external view returns (bytes4 magicValue) {
        if (revertOnSignature) {
            revert("revert");
        }

        magicValue = _magicValue;
    }
}
