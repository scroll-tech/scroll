// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {L2GasPriceOracle} from "../L1/rollup/L2GasPriceOracle.sol";
import {Whitelist} from "../L2/predeploys/Whitelist.sol";

contract L2GasPriceOracleTest is DSTestPlus {
    L2GasPriceOracle private oracle;
    Whitelist private whitelist;
    uint256 fee;

    event Log(address addr);

    function setUp() public {
        whitelist = new Whitelist(address(this));
        oracle = L2GasPriceOracle(address(new ERC1967Proxy(address(new L2GasPriceOracle()), new bytes(0))));

        oracle.initialize(0, 0, 0, 0);
        oracle.updateWhitelist(address(whitelist));

        address[] memory _accounts = new address[](1);
        _accounts[0] = address(this);
        whitelist.updateWhitelistStatus(_accounts, true);
    }

    function testCalculateIntrinsicGasFee() external {
        uint256 intrinsicGasFee = oracle.calculateIntrinsicGasFee(hex"00");
        assertEq(intrinsicGasFee, 0);
        uint64 zeroGas = 5;
        uint64 nonZeroGas = 10;
        oracle.setIntrinsicParams(20000, 50000, zeroGas, nonZeroGas);

        intrinsicGasFee = oracle.calculateIntrinsicGasFee(hex"001122");
        // 20000 + 1 zero bytes * 5 + 2 nonzero byte * 10 = 20025
        assertEq(intrinsicGasFee, 20025);

        zeroGas = 50;
        nonZeroGas = 100;
        oracle.setIntrinsicParams(10000, 20000, zeroGas, nonZeroGas);

        intrinsicGasFee = oracle.calculateIntrinsicGasFee(hex"0011220033");
        // 10000 + 3 nonzero byte * 100 + 2 zero bytes * 50 = 10000 + 300 + 100 = 10400
        assertEq(intrinsicGasFee, 10400);
    }

    function testSetIntrinsicParamsAccess() external {
        hevm.startPrank(address(4));
        hevm.expectRevert("Not whitelisted sender");
        oracle.setIntrinsicParams(1, 0, 0, 1);
    }

    // forge t --match-contract L2GasPriceOracleTest --match-test testBenchmark --gas-report
    // function testBenchmark() external {
    //     // 50 bytes
    //     fee = oracle.calculateIntrinsicGasFee(hex"11111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111");
    // }
}
