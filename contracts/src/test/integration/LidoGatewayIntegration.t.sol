// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {MockERC20} from "solmate/test/utils/mocks/MockERC20.sol";

import {GatewayIntegrationBase} from "./GatewayIntegrationBase.t.sol";

import {IL1ERC20Gateway} from "../../L1/gateways/IL1ERC20Gateway.sol";
import {IL2ERC20Gateway} from "../../L2/gateways/IL2ERC20Gateway.sol";
import {L1LidoGateway} from "../../lido/L1LidoGateway.sol";
import {L2LidoGateway} from "../../lido/L2LidoGateway.sol";

interface IWstETH {
    function wrap(uint256 _stETHAmount) external returns (uint256);

    function unwrap(uint256 _wstETHAmount) external returns (uint256);

    function getStETHByWstETH(uint256 _wstETHAmount) external view returns (uint256);

    function getWstETHByStETH(uint256 _stETHAmount) external view returns (uint256);

    function stEthPerToken() external view returns (uint256);

    function tokensPerStEth() external view returns (uint256);
}

contract LidoGatewayIntegrationTest is GatewayIntegrationBase {
    address private constant L1_LIDO_GATEWAY = 0x6625C6332c9F91F2D27c304E729B86db87A3f504;

    address private constant L1_STETH = 0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84;

    address private constant L1_WSTETH = 0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0;

    address private constant L2_LIDO_GATEWAY = 0x8aE8f22226B9d789A36AC81474e633f8bE2856c9;

    address private constant L2_WSTETH = 0xf610A9dfB7C89644979b4A0f27063E9e7d7Cda32;

    function setUp() public {
        __GatewayIntegrationBase_setUp();

        mainnet.selectFork();
        upgrade(
            true,
            L1_LIDO_GATEWAY,
            address(new L1LidoGateway(L1_WSTETH, L2_WSTETH, L2_LIDO_GATEWAY, L1_GATEWAY_ROUTER, L1_SCROLL_MESSENGER))
        );
        L1LidoGateway(L1_LIDO_GATEWAY).initializeV2(address(0), address(0), address(0), address(0));

        scroll.selectFork();
        upgrade(
            false,
            L2_LIDO_GATEWAY,
            address(new L2LidoGateway(L1_WSTETH, L2_WSTETH, L1_LIDO_GATEWAY, L2_GATEWAY_ROUTER, L2_SCROLL_MESSENGER))
        );
        L2LidoGateway(L2_LIDO_GATEWAY).initializeV2(address(0), address(0), address(0), address(0));
    }

    function testWithoutRouter() private {
        depositAndWithdraw(false);
    }

    function testWithRouter() private {
        depositAndWithdraw(true);
    }

    function depositAndWithdraw(bool useRouter) private {
        vm.recordLogs();

        mainnet.selectFork();
        uint256 rate = IWstETH(L1_WSTETH).stEthPerToken();

        // deposit to get some stETH
        (bool succeed, ) = L1_STETH.call{value: 11 * rate}("");
        assertEq(true, succeed);
        assertApproxEqAbs(MockERC20(L1_STETH).balanceOf(address(this)), 11 * rate, 10);

        // wrap stETH to wstETH
        MockERC20(L1_STETH).approve(L1_WSTETH, 10 * rate);
        IWstETH(L1_WSTETH).wrap(10 * rate);
        assertApproxEqAbs(MockERC20(L1_WSTETH).balanceOf(address(this)), 10 ether, 10);

        // deposit 1 wstETH
        uint256 l1GatewayBalance = MockERC20(L1_WSTETH).balanceOf(L1_LIDO_GATEWAY);
        uint256 l1Balance = MockERC20(L1_WSTETH).balanceOf(address(this));
        if (useRouter) {
            MockERC20(L1_WSTETH).approve(L1_GATEWAY_ROUTER, 1 ether);
            IL1ERC20Gateway(L1_GATEWAY_ROUTER).depositERC20{value: 1 ether}(L1_WSTETH, 1 ether, 400000);
        } else {
            MockERC20(L1_WSTETH).approve(L1_LIDO_GATEWAY, 1 ether);
            IL1ERC20Gateway(L1_LIDO_GATEWAY).depositERC20{value: 1 ether}(L1_WSTETH, 1 ether, 400000);
        }
        assertEq(l1Balance - 1 ether, MockERC20(L1_WSTETH).balanceOf(address(this)));
        assertEq(l1GatewayBalance + 1 ether, MockERC20(L1_WSTETH).balanceOf(L1_LIDO_GATEWAY));

        // relay message to Scroll and check balance
        scroll.selectFork();
        uint256 l2Balance = MockERC20(L2_WSTETH).balanceOf(address(this));
        relayFromMainnet();

        // withdraw wstETH
        scroll.selectFork();
        assertEq(l2Balance + 1 ether, MockERC20(L2_WSTETH).balanceOf(address(this)));
        assertEq(0, MockERC20(L2_WSTETH).balanceOf(L2_LIDO_GATEWAY));
        if (useRouter) {
            IL2ERC20Gateway(L2_GATEWAY_ROUTER).withdrawERC20(L2_WSTETH, 1 ether, 0);
        } else {
            IL2ERC20Gateway(L2_LIDO_GATEWAY).withdrawERC20(L2_WSTETH, 1 ether, 0);
        }
        assertEq(l2Balance, MockERC20(L2_WSTETH).balanceOf(address(this)));
        assertEq(0, MockERC20(L2_WSTETH).balanceOf(L2_LIDO_GATEWAY));

        // relay message to Mainnet and check balance
        mainnet.selectFork();
        l1GatewayBalance = MockERC20(L1_WSTETH).balanceOf(L1_LIDO_GATEWAY);
        l1Balance = MockERC20(L1_WSTETH).balanceOf(address(this));
        relayFromScroll();
        mainnet.selectFork();
        assertEq(l1Balance + 1 ether, MockERC20(L1_WSTETH).balanceOf(address(this)));
        assertEq(l1GatewayBalance - 1 ether, MockERC20(L1_WSTETH).balanceOf(L1_LIDO_GATEWAY));
    }
}
