// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {L2GasPriceOracle} from "../L1/rollup/L2GasPriceOracle.sol";
import {Whitelist} from "../L2/predeploys/Whitelist.sol";

contract L2GasPriceOracleTest is DSTestPlus {
    uint256 private constant PRECISION = 1e9;
    uint256 private constant MAX_OVERHEAD = 30000000 / 16;
    uint256 private constant MAX_SCALE = 1000 * PRECISION;

    L2GasPriceOracle private oracle;
    Whitelist private whitelist;

    event Log(address addr);

    function setUp() public {
        whitelist = new Whitelist(address(this));
        oracle = new L2GasPriceOracle();

        oracle.initialize(0,0,0);
        oracle.updateWhitelist(address(whitelist));

        address[] memory _accounts = new address[](1);
        _accounts[0] = address(this);
        whitelist.updateWhitelistStatus(_accounts, true);
    }

    function testCalculateIntrinsicGasFee() external {
        uint256 fee = oracle.calculateIntrinsicGasFee(hex"00");
        assertEq(fee, 0);
        uint256 zeroGas = 5;
        uint256 nonZeroGas = 10;
        oracle.setIntrinsicParams(20000, zeroGas, nonZeroGas);

        fee = oracle.calculateIntrinsicGasFee(hex"001122");
        // 20000 + 1 zero bytes * 5 +2 nonzero byte * 10 = 20025
        assertEq(fee, 20025);

        zeroGas = 50;
        nonZeroGas = 100;
        oracle.setIntrinsicParams(10000, zeroGas, nonZeroGas);

        fee = oracle.calculateIntrinsicGasFee(hex"0011220033");
        // 10000 + 3 nonzero byte * 100 + 2 zero bytes * 50 = 10000 + 300 + 100 = 10400
        assertEq(fee, 10400);
        

        uint256 MAX_UINT_64 = 2**64-1;

        oracle.setIntrinsicParams(1, 2**63, 0);
        fee = oracle.calculateIntrinsicGasFee(hex"11");

        hevm.expectRevert("Intrinsic gas overflows from zero bytes cost");
        fee = oracle.calculateIntrinsicGasFee(hex"00");
    
        oracle.setIntrinsicParams(1, 0, 2**63);
        fee = oracle.calculateIntrinsicGasFee(hex"00");

        hevm.expectRevert("Intrinsic gas overflows from zero bytes cost");
        fee = oracle.calculateIntrinsicGasFee(hex"11");
    }

}
