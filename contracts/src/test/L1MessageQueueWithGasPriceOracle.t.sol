// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {ITransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import {IL1MessageQueueWithGasPriceOracle} from "../L1/rollup/IL1MessageQueueWithGasPriceOracle.sol";
import {L1MessageQueueWithGasPriceOracle} from "../L1/rollup/L1MessageQueueWithGasPriceOracle.sol";
import {L2GasPriceOracle} from "../L1/rollup/L2GasPriceOracle.sol";
import {Whitelist} from "../L2/predeploys/Whitelist.sol";

import {ScrollTestBase} from "./ScrollTestBase.t.sol";

contract L1MessageQueueWithGasPriceOracleTest is ScrollTestBase {
    // events
    event UpdateWhitelist(address indexed _oldWhitelist, address indexed _newWhitelist);
    event UpdateL2BaseFee(uint256 oldL2BaseFee, uint256 newL2BaseFee);

    L1MessageQueueWithGasPriceOracle private queue;
    L2GasPriceOracle internal gasOracle;
    Whitelist private whitelist;

    function setUp() public {
        __ScrollTestBase_setUp();

        queue = L1MessageQueueWithGasPriceOracle(_deployProxy(address(0)));
        gasOracle = L2GasPriceOracle(_deployProxy(address(new L2GasPriceOracle())));
        whitelist = new Whitelist(address(this));

        // initialize L2GasPriceOracle
        gasOracle.initialize(1, 2, 1, 1);
        gasOracle.updateWhitelist(address(whitelist));

        // Setup whitelist
        address[] memory _accounts = new address[](1);
        _accounts[0] = address(this);
        whitelist.updateWhitelistStatus(_accounts, true);

        // Upgrade the L1MessageQueueWithGasPriceOracle implementation and initialize
        admin.upgrade(
            ITransparentUpgradeableProxy(address(queue)),
            address(new L1MessageQueueWithGasPriceOracle(address(1), address(1), address(1)))
        );
        queue.initialize(address(1), address(1), address(1), address(gasOracle), 10000000);
        queue.initializeV2();
    }

    function testUpdateWhitelist(address _newWhitelist) external {
        hevm.assume(_newWhitelist != address(whitelist));

        // call by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        queue.updateWhitelist(_newWhitelist);
        hevm.stopPrank();

        // call by owner, should succeed
        assertEq(address(queue.whitelist()), address(whitelist));
        hevm.expectEmit(true, true, false, true);
        emit UpdateWhitelist(address(whitelist), _newWhitelist);
        queue.updateWhitelist(_newWhitelist);
        assertEq(address(queue.whitelist()), _newWhitelist);
    }

    function testSetL2BaseFee(uint256 _baseFee1, uint256 _baseFee2) external {
        // call by non-whitelister, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert(IL1MessageQueueWithGasPriceOracle.ErrorNotWhitelistedSender.selector);
        queue.setL2BaseFee(_baseFee1);
        hevm.stopPrank();

        // call by owner, should succeed
        assertEq(queue.l2BaseFee(), 0);
        hevm.expectEmit(false, false, false, true);
        emit UpdateL2BaseFee(0, _baseFee1);
        queue.setL2BaseFee(_baseFee1);
        assertEq(queue.l2BaseFee(), _baseFee1);

        hevm.expectEmit(false, false, false, true);
        emit UpdateL2BaseFee(_baseFee1, _baseFee2);
        queue.setL2BaseFee(_baseFee2);
        assertEq(queue.l2BaseFee(), _baseFee2);
    }

    function testEstimateCrossDomainMessageFee(uint256 baseFee, uint256 gasLimit) external {
        gasLimit = bound(gasLimit, 0, 3000000);
        baseFee = bound(baseFee, 0, 1000000000);

        assertEq(queue.estimateCrossDomainMessageFee(gasLimit), 0);

        queue.setL2BaseFee(baseFee);
        assertEq(queue.estimateCrossDomainMessageFee(gasLimit), baseFee * gasLimit);
    }

    function testCalculateIntrinsicGasFee(bytes memory data) external {
        assertEq(queue.calculateIntrinsicGasFee(data), 21000 + data.length * 16);
    }
}
