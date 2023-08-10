// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {MockERC20} from "solmate/test/utils/mocks/MockERC20.sol";

import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {L1ETHGateway} from "../L1/gateways/L1ETHGateway.sol";
import {L1StandardERC20Gateway} from "../L1/gateways/L1StandardERC20Gateway.sol";
import {L1ScrollMessenger} from "../L1/L1ScrollMessenger.sol";
import {ScrollChain} from "../L1/rollup/ScrollChain.sol";
import {L2ETHGateway} from "../L2/gateways/L2ETHGateway.sol";
import {L2GatewayRouter} from "../L2/gateways/L2GatewayRouter.sol";
import {L2StandardERC20Gateway} from "../L2/gateways/L2StandardERC20Gateway.sol";
import {ScrollStandardERC20} from "../libraries/token/ScrollStandardERC20.sol";
import {ScrollStandardERC20Factory} from "../libraries/token/ScrollStandardERC20Factory.sol";

import {L2GatewayTestBase} from "./L2GatewayTestBase.t.sol";

contract L2GatewayRouterTest is L2GatewayTestBase {
    // from L2GatewayRouter
    event SetETHGateway(address indexed oldETHGateway, address indexed newEthGateway);
    event SetDefaultERC20Gateway(address indexed oldDefaultERC20Gateway, address indexed newDefaultERC20Gateway);
    event SetERC20Gateway(address indexed token, address indexed oldGateway, address indexed newGateway);

    ScrollStandardERC20 private template;
    ScrollStandardERC20Factory private factory;

    L1StandardERC20Gateway private l1StandardERC20Gateway;
    L2StandardERC20Gateway private l2StandardERC20Gateway;

    L1ETHGateway private l1ETHGateway;
    L2ETHGateway private l2ETHGateway;

    L2GatewayRouter private router;
    MockERC20 private l1Token;

    function setUp() public {
        setUpBase();

        // Deploy tokens
        l1Token = new MockERC20("Mock", "M", 18);

        // Deploy L1 contracts
        l1StandardERC20Gateway = new L1StandardERC20Gateway();
        l1ETHGateway = new L1ETHGateway();

        // Deploy L2 contracts
        l2StandardERC20Gateway = L2StandardERC20Gateway(
            address(new ERC1967Proxy(address(new L2StandardERC20Gateway()), new bytes(0)))
        );
        l2ETHGateway = L2ETHGateway(address(new ERC1967Proxy(address(new L2ETHGateway()), new bytes(0))));
        router = L2GatewayRouter(address(new ERC1967Proxy(address(new L2GatewayRouter()), new bytes(0))));
        template = new ScrollStandardERC20();
        factory = new ScrollStandardERC20Factory(address(template));

        // Initialize L2 contracts
        l2StandardERC20Gateway.initialize(
            address(l1StandardERC20Gateway),
            address(router),
            address(l1Messenger),
            address(factory)
        );
        l2ETHGateway.initialize(address(l1ETHGateway), address(router), address(l2Messenger));
        router.initialize(address(l2ETHGateway), address(l2StandardERC20Gateway));
    }

    function testOwnership() public {
        assertEq(address(this), router.owner());
    }

    function testInitialized() public {
        assertEq(address(l2ETHGateway), router.ethGateway());
        assertEq(address(l2StandardERC20Gateway), router.defaultERC20Gateway());
        assertEq(address(l2StandardERC20Gateway), router.getERC20Gateway(address(l1Token)));

        hevm.expectRevert("Initializable: contract is already initialized");
        router.initialize(address(l2ETHGateway), address(l2StandardERC20Gateway));
    }

    function testSetDefaultERC20Gateway() public {
        router.setDefaultERC20Gateway(address(0));

        // set by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        router.setDefaultERC20Gateway(address(l2StandardERC20Gateway));
        hevm.stopPrank();

        // set by owner, should succeed
        hevm.expectEmit(true, true, false, true);
        emit SetDefaultERC20Gateway(address(0), address(l2StandardERC20Gateway));

        assertEq(address(0), router.getERC20Gateway(address(l1Token)));
        assertEq(address(0), router.defaultERC20Gateway());
        router.setDefaultERC20Gateway(address(l2StandardERC20Gateway));
        assertEq(address(l2StandardERC20Gateway), router.getERC20Gateway(address(l1Token)));
        assertEq(address(l2StandardERC20Gateway), router.defaultERC20Gateway());
    }

    function testSetERC20Gateway() public {
        router.setDefaultERC20Gateway(address(0));

        // length mismatch, should revert
        address[] memory empty = new address[](0);
        address[] memory single = new address[](1);
        hevm.expectRevert("length mismatch");
        router.setERC20Gateway(empty, single);
        hevm.expectRevert("length mismatch");
        router.setERC20Gateway(single, empty);

        // set by owner, should succeed
        address[] memory _tokens = new address[](1);
        address[] memory _gateways = new address[](1);
        _tokens[0] = address(l1Token);
        _gateways[0] = address(l2StandardERC20Gateway);

        hevm.expectEmit(true, true, true, true);
        emit SetERC20Gateway(address(l1Token), address(0), address(l2StandardERC20Gateway));

        assertEq(address(0), router.getERC20Gateway(address(l1Token)));
        router.setERC20Gateway(_tokens, _gateways);
        assertEq(address(l2StandardERC20Gateway), router.getERC20Gateway(address(l1Token)));
    }

    function testFinalizeDepositERC20() public {
        hevm.expectRevert("should never be called");
        router.finalizeDepositERC20(address(0), address(0), address(0), address(0), 0, "");
    }

    function testFinalizeDepositETH() public {
        hevm.expectRevert("should never be called");
        router.finalizeDepositETH(address(0), address(0), 0, "");
    }
}
