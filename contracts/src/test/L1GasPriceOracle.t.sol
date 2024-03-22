// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {L1BlockContainer} from "../L2/predeploys/L1BlockContainer.sol";
import {L1GasPriceOracle} from "../L2/predeploys/L1GasPriceOracle.sol";
import {Whitelist} from "../L2/predeploys/Whitelist.sol";

contract L1GasPriceOracleTest is DSTestPlus {
    uint256 private constant PRECISION = 1e9;
    uint256 private constant MAX_OVERHEAD = 30000000 / 16;
    uint256 private constant MAX_SCALE = 1000 * PRECISION;

    L1GasPriceOracle private oracle;
    Whitelist private whitelist;

    function setUp() public {
        whitelist = new Whitelist(address(this));
        oracle = new L1GasPriceOracle(address(this));
        oracle.updateWhitelist(address(whitelist));

        address[] memory _accounts = new address[](1);
        _accounts[0] = address(this);
        whitelist.updateWhitelistStatus(_accounts, true);
    }

    function testSetOverhead(uint256 _overhead) external {
        _overhead = bound(_overhead, 0, MAX_OVERHEAD);

        // call by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("caller is not the owner");
        oracle.setOverhead(_overhead);
        hevm.stopPrank();

        // overhead is too large
        hevm.expectRevert("exceed maximum overhead");
        oracle.setOverhead(MAX_OVERHEAD + 1);

        // call by owner, should succeed
        assertEq(oracle.overhead(), 0);
        oracle.setOverhead(_overhead);
        assertEq(oracle.overhead(), _overhead);
    }

    function testSetScalar(uint256 _scalar) external {
        _scalar = bound(_scalar, 0, MAX_SCALE);

        // call by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("caller is not the owner");
        oracle.setScalar(_scalar);
        hevm.stopPrank();

        // scale is too large
        hevm.expectRevert("exceed maximum scale");
        oracle.setScalar(MAX_SCALE + 1);

        // call by owner, should succeed
        assertEq(oracle.scalar(), 0);
        oracle.setScalar(_scalar);
        assertEq(oracle.scalar(), _scalar);
    }

    function testUpdateWhitelist(address _newWhitelist) external {
        hevm.assume(_newWhitelist != address(whitelist));

        // call by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("caller is not the owner");
        oracle.updateWhitelist(_newWhitelist);
        hevm.stopPrank();

        // call by owner, should succeed
        assertEq(address(oracle.whitelist()), address(whitelist));
        oracle.updateWhitelist(_newWhitelist);
        assertEq(address(oracle.whitelist()), _newWhitelist);
    }

    function testSetL1BaseFee(uint256 _baseFee) external {
        _baseFee = bound(_baseFee, 0, 1e9 * 20000); // max 20k gwei

        // call by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Not whitelisted sender");
        oracle.setL1BaseFee(_baseFee);
        hevm.stopPrank();

        // call by owner, should succeed
        assertEq(oracle.l1BaseFee(), 0);
        oracle.setL1BaseFee(_baseFee);
        assertEq(oracle.l1BaseFee(), _baseFee);
    }

    function testGetL1GasUsed(uint256 _overhead, bytes memory _data) external {
        _overhead = bound(_overhead, 0, MAX_OVERHEAD);

        oracle.setOverhead(_overhead);

        uint256 _gasUsed = _overhead + 4 * 16;
        for (uint256 i = 0; i < _data.length; i++) {
            if (_data[i] == 0) _gasUsed += 4;
            else _gasUsed += 16;
        }

        assertEq(oracle.getL1GasUsed(_data), _gasUsed);
    }

    function testGetL1Fee(
        uint256 _baseFee,
        uint256 _overhead,
        uint256 _scalar,
        bytes memory _data
    ) external {
        _overhead = bound(_overhead, 0, MAX_OVERHEAD);
        _scalar = bound(_scalar, 0, MAX_SCALE);
        _baseFee = bound(_baseFee, 0, 1e9 * 20000); // max 20k gwei

        oracle.setOverhead(_overhead);
        oracle.setScalar(_scalar);
        oracle.setL1BaseFee(_baseFee);

        uint256 _gasUsed = _overhead + 4 * 16;
        for (uint256 i = 0; i < _data.length; i++) {
            if (_data[i] == 0) _gasUsed += 4;
            else _gasUsed += 16;
        }

        assertEq(oracle.getL1Fee(_data), (_gasUsed * _baseFee * _scalar) / PRECISION);
    }
}
