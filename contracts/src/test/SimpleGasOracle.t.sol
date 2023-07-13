// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";
import {WETH} from "solmate/tokens/WETH.sol";

import {SimpleGasOracle} from "../libraries/oracle/SimpleGasOracle.sol";

contract SimpleGasOracleTest is DSTestPlus {
    SimpleGasOracle private oracle;

    function setUp() public {
        oracle = new SimpleGasOracle();
        oracle.initialize();
    }

    function testUpdateDefaultFeeConfig(uint128 _baseFees, uint128 _feesPerByte) external {
        // call by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        oracle.updateDefaultFeeConfig(_baseFees, _feesPerByte);
        hevm.stopPrank();

        // call by owner, should succeed
        oracle.updateDefaultFeeConfig(111, 3333); // random set some non-zero value first
        (uint128 __baseFees, uint128 __feesPerByte) = oracle.defaultFeeConfig();
        assertEq(__baseFees, 111);
        assertEq(__feesPerByte, 3333);
        oracle.updateDefaultFeeConfig(_baseFees, _feesPerByte);
        (__baseFees, __feesPerByte) = oracle.defaultFeeConfig();
        assertEq(__baseFees, _baseFees);
        assertEq(__feesPerByte, _feesPerByte);
    }

    function testUpdateCustomFeeConfig(
        address _sender,
        uint128 _baseFees,
        uint128 _feesPerByte
    ) external {
        // call by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        oracle.updateCustomFeeConfig(_sender, _baseFees, _feesPerByte);
        hevm.stopPrank();

        // call by owner, should succeed
        (uint128 __baseFees, uint128 __feesPerByte) = oracle.customFeeConfig(_sender);
        assertEq(__baseFees, 0);
        assertEq(__feesPerByte, 0);
        assertBoolEq(oracle.hasCustomConfig(_sender), false);
        oracle.updateCustomFeeConfig(_sender, _baseFees, _feesPerByte);
        (__baseFees, __feesPerByte) = oracle.customFeeConfig(_sender);
        assertEq(__baseFees, _baseFees);
        assertEq(__feesPerByte, _feesPerByte);
        assertBoolEq(oracle.hasCustomConfig(_sender), true);
    }

    function testEstimateMessageFee(
        address _sender,
        address,
        bytes memory _message
    ) external {
        // use default config when no custom config
        oracle.updateDefaultFeeConfig(1, 2);
        assertEq(oracle.estimateMessageFee(_sender, address(0), _message, 0), 1 + 2 * _message.length);

        // use custom config when set
        oracle.updateCustomFeeConfig(_sender, 4, 5);
        assertEq(oracle.estimateMessageFee(_sender, address(0), _message, 0), 4 + 5 * _message.length);
    }
}
