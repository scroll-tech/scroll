// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {MockERC20} from "solmate/test/utils/mocks/MockERC20.sol";

import {ITransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import {L1ETHGateway} from "../L1/gateways/L1ETHGateway.sol";
import {L1GatewayRouter} from "../L1/gateways/L1GatewayRouter.sol";
import {L1StandardERC20Gateway} from "../L1/gateways/L1StandardERC20Gateway.sol";
import {L2ETHGateway} from "../L2/gateways/L2ETHGateway.sol";
import {L2StandardERC20Gateway} from "../L2/gateways/L2StandardERC20Gateway.sol";
import {ScrollStandardERC20} from "../libraries/token/ScrollStandardERC20.sol";
import {ScrollStandardERC20Factory} from "../libraries/token/ScrollStandardERC20Factory.sol";

import {L1GatewayTestBase} from "./L1GatewayTestBase.t.sol";

import {TransferReentrantToken} from "./mocks/tokens/TransferReentrantToken.sol";

contract L1GatewayRouterTest is L1GatewayTestBase {
    // from L1GatewayRouter
    event SetETHGateway(address indexed oldETHGateway, address indexed newEthGateway);
    event SetDefaultERC20Gateway(address indexed oldDefaultERC20Gateway, address indexed newDefaultERC20Gateway);
    event SetERC20Gateway(address indexed token, address indexed oldGateway, address indexed newGateway);

    ScrollStandardERC20 private template;
    ScrollStandardERC20Factory private factory;

    L1StandardERC20Gateway private l1StandardERC20Gateway;
    L2StandardERC20Gateway private l2StandardERC20Gateway;

    L1ETHGateway private l1ETHGateway;
    L2ETHGateway private l2ETHGateway;

    L1GatewayRouter private router;
    MockERC20 private l1Token;

    function setUp() public {
        __L1GatewayTestBase_setUp();

        // Deploy tokens
        l1Token = new MockERC20("Mock", "M", 18);

        // Deploy L2 contracts
        template = new ScrollStandardERC20();
        factory = new ScrollStandardERC20Factory(address(template));
        l2StandardERC20Gateway = new L2StandardERC20Gateway(address(1), address(1), address(1), address(factory));
        l2ETHGateway = new L2ETHGateway(address(1), address(1), address(1));

        // Deploy L1 contracts
        l1StandardERC20Gateway = L1StandardERC20Gateway(_deployProxy(address(0)));
        l1ETHGateway = L1ETHGateway(_deployProxy(address(0)));
        router = L1GatewayRouter(_deployProxy(address(new L1GatewayRouter())));
        admin.upgrade(
            ITransparentUpgradeableProxy(address(l1StandardERC20Gateway)),
            address(
                new L1StandardERC20Gateway(
                    address(l2StandardERC20Gateway),
                    address(router),
                    address(l1Messenger),
                    address(template),
                    address(factory)
                )
            )
        );
        admin.upgrade(
            ITransparentUpgradeableProxy(address(l1ETHGateway)),
            address(new L1ETHGateway(address(l2ETHGateway), address(router), address(l1Messenger)))
        );

        // Initialize L1 contracts
        l1StandardERC20Gateway.initialize(
            address(l2StandardERC20Gateway),
            address(router),
            address(l1Messenger),
            address(template),
            address(factory)
        );
        l1ETHGateway.initialize(address(l2ETHGateway), address(router), address(l1Messenger));
        router.initialize(address(l1ETHGateway), address(l1StandardERC20Gateway));
    }

    function testOwnership() public {
        assertEq(address(this), router.owner());
    }

    function testInitialized() public {
        assertEq(address(l1StandardERC20Gateway), router.defaultERC20Gateway());
        assertEq(
            factory.computeL2TokenAddress(address(l2StandardERC20Gateway), address(l1Token)),
            router.getL2ERC20Address(address(l1Token))
        );
        assertEq(address(l1StandardERC20Gateway), router.getERC20Gateway(address(l1Token)));

        hevm.expectRevert("Initializable: contract is already initialized");
        router.initialize(address(l1ETHGateway), address(l1StandardERC20Gateway));
    }

    function testSetDefaultERC20Gateway() public {
        router.setDefaultERC20Gateway(address(0));

        // set by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        router.setDefaultERC20Gateway(address(l1StandardERC20Gateway));
        hevm.stopPrank();

        // set by owner, should succeed
        hevm.expectEmit(true, true, false, true);
        emit SetDefaultERC20Gateway(address(0), address(l1StandardERC20Gateway));

        assertEq(address(0), router.getERC20Gateway(address(l1Token)));
        assertEq(address(0), router.defaultERC20Gateway());
        router.setDefaultERC20Gateway(address(l1StandardERC20Gateway));
        assertEq(address(l1StandardERC20Gateway), router.getERC20Gateway(address(l1Token)));
        assertEq(address(l1StandardERC20Gateway), router.defaultERC20Gateway());
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
        _gateways[0] = address(l1StandardERC20Gateway);

        hevm.expectEmit(true, true, true, true);
        emit SetERC20Gateway(address(l1Token), address(0), address(l1StandardERC20Gateway));

        assertEq(address(0), router.getERC20Gateway(address(l1Token)));
        router.setERC20Gateway(_tokens, _gateways);
        assertEq(address(l1StandardERC20Gateway), router.getERC20Gateway(address(l1Token)));
    }

    function testFinalizeWithdrawERC20() public {
        hevm.expectRevert("should never be called");
        router.finalizeWithdrawERC20(address(0), address(0), address(0), address(0), 0, "");
    }

    function testFinalizeWithdrawETH() public {
        hevm.expectRevert("should never be called");
        router.finalizeWithdrawETH(address(0), address(0), 0, "");
    }

    function testRequestERC20(
        address _sender,
        address _token,
        uint256 _amount
    ) public {
        hevm.expectRevert("Only in deposit context");
        router.requestERC20(_sender, _token, _amount);
    }

    function testReentrant() public {
        TransferReentrantToken reentrantToken = new TransferReentrantToken("Reentrant", "R", 18);
        reentrantToken.mint(address(this), type(uint128).max);
        reentrantToken.approve(address(router), type(uint256).max);

        reentrantToken.setReentrantCall(
            address(router),
            0,
            abi.encodeWithSelector(
                router.depositERC20AndCall.selector,
                address(reentrantToken),
                address(this),
                0,
                new bytes(0),
                0
            ),
            true
        );
        hevm.expectRevert("Only not in context");
        router.depositERC20(address(reentrantToken), 1, 0);

        reentrantToken.setReentrantCall(
            address(router),
            0,
            abi.encodeWithSelector(
                router.depositERC20AndCall.selector,
                address(reentrantToken),
                address(this),
                0,
                new bytes(0),
                0
            ),
            false
        );
        hevm.expectRevert("Only not in context");
        router.depositERC20(address(reentrantToken), 1, 0);
    }
}
