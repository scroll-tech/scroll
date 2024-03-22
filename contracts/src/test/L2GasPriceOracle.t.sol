// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {L2GasPriceOracle} from "../L1/rollup/L2GasPriceOracle.sol";
import {Whitelist} from "../L2/predeploys/Whitelist.sol";

contract L2GasPriceOracleTest is DSTestPlus {
    // events
    event L2BaseFeeUpdated(uint256 oldL2BaseFee, uint256 newL2BaseFee);
    event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

    L2GasPriceOracle private oracle;
    Whitelist private whitelist;
    uint256 fee;

    event Log(address addr);

    function setUp() public {
        whitelist = new Whitelist(address(this));
        oracle = L2GasPriceOracle(address(new ERC1967Proxy(address(new L2GasPriceOracle()), new bytes(0))));

        oracle.initialize(1, 2, 1, 1);
        oracle.updateWhitelist(address(whitelist));

        address[] memory _accounts = new address[](1);
        _accounts[0] = address(this);
        whitelist.updateWhitelistStatus(_accounts, true);
    }

    function testCalculateIntrinsicGasFee() external {
        uint256 intrinsicGasFee = oracle.calculateIntrinsicGasFee(hex"00");
        assertEq(intrinsicGasFee, 2);
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
        hevm.expectRevert("Ownable: caller is not the owner");
        oracle.setIntrinsicParams(1, 0, 0, 1);
    }

    function testSetL2BaseFee(uint256 _baseFee1, uint256 _baseFee2) external {
        // call by non-whitelister, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Not whitelisted sender");
        oracle.setL2BaseFee(_baseFee1);
        hevm.stopPrank();

        // call by owner, should succeed
        assertEq(oracle.l2BaseFee(), 0);
        hevm.expectEmit(false, false, false, true);
        emit L2BaseFeeUpdated(0, _baseFee1);
        oracle.setL2BaseFee(_baseFee1);
        assertEq(oracle.l2BaseFee(), _baseFee1);

        hevm.expectEmit(false, false, false, true);
        emit L2BaseFeeUpdated(_baseFee1, _baseFee2);
        oracle.setL2BaseFee(_baseFee2);
        assertEq(oracle.l2BaseFee(), _baseFee2);
    }

    function testEstimateCrossDomainMessageFee(uint256 baseFee, uint256 gasLimit) external {
        gasLimit = bound(gasLimit, 0, 3000000);
        baseFee = bound(baseFee, 0, 1000000000);

        assertEq(oracle.estimateCrossDomainMessageFee(gasLimit), 0);

        oracle.setL2BaseFee(baseFee);
        assertEq(oracle.estimateCrossDomainMessageFee(gasLimit), baseFee * gasLimit);
    }
}
