// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.24;

import {Script} from "forge-std/Script.sol";

import {ProxyAdmin} from "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import {ITransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import {L2ScrollMessenger} from "../../src/L2/L2ScrollMessenger.sol";
import {L2CustomERC20Gateway} from "../../src/L2/gateways/L2CustomERC20Gateway.sol";
import {L2ERC1155Gateway} from "../../src/L2/gateways/L2ERC1155Gateway.sol";
import {L2ERC721Gateway} from "../../src/L2/gateways/L2ERC721Gateway.sol";
import {L2ETHGateway} from "../../src/L2/gateways/L2ETHGateway.sol";
import {L2GatewayRouter} from "../../src/L2/gateways/L2GatewayRouter.sol";
import {L2StandardERC20Gateway} from "../../src/L2/gateways/L2StandardERC20Gateway.sol";
import {L2WETHGateway} from "../../src/L2/gateways/L2WETHGateway.sol";
import {L2MessageQueue} from "../../src/L2/predeploys/L2MessageQueue.sol";
import {L2TxFeeVault} from "../../src/L2/predeploys/L2TxFeeVault.sol";
import {L1GasPriceOracle} from "../../src/L2/predeploys/L1GasPriceOracle.sol";
import {Whitelist} from "../../src/L2/predeploys/Whitelist.sol";
import {ScrollStandardERC20Factory} from "../../src/libraries/token/ScrollStandardERC20Factory.sol";

// solhint-disable max-states-count
// solhint-disable state-visibility
// solhint-disable var-name-mixedcase

contract InitializeL2BridgeContracts is Script {
    uint256 deployerPrivateKey = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");

    address L2_WETH_ADDR = vm.envAddress("L2_WETH_ADDR");
    address L2_PROXY_ADMIN_ADDR = vm.envAddress("L2_PROXY_ADMIN_ADDR");

    address L1_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L1_SCROLL_MESSENGER_PROXY_ADDR");
    address L1_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L1_GATEWAY_ROUTER_PROXY_ADDR");
    address L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L1_ERC721_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ERC721_GATEWAY_PROXY_ADDR");
    address L1_ERC1155_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ERC1155_GATEWAY_PROXY_ADDR");
    address L1_ETH_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ETH_GATEWAY_PROXY_ADDR");
    address L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L1_WETH_GATEWAY_PROXY_ADDR = vm.envAddress("L1_WETH_GATEWAY_PROXY_ADDR");

    address L2_TX_FEE_VAULT_ADDR = vm.envAddress("L2_TX_FEE_VAULT_ADDR");
    address L1_GAS_PRICE_ORACLE_ADDR = vm.envAddress("L1_GAS_PRICE_ORACLE_ADDR");
    address L2_WHITELIST_ADDR = vm.envAddress("L2_WHITELIST_ADDR");
    address L2_MESSAGE_QUEUE_ADDR = vm.envAddress("L2_MESSAGE_QUEUE_ADDR");

    address L2_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L2_SCROLL_MESSENGER_PROXY_ADDR");
    address L2_SCROLL_MESSENGER_IMPLEMENTATION_ADDR = vm.envAddress("L2_SCROLL_MESSENGER_IMPLEMENTATION_ADDR");
    address L2_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L2_GATEWAY_ROUTER_PROXY_ADDR");
    address L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L2_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR = vm.envAddress("L2_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR");
    address L2_ERC721_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC721_GATEWAY_PROXY_ADDR");
    address L2_ERC721_GATEWAY_IMPLEMENTATION_ADDR = vm.envAddress("L2_ERC721_GATEWAY_IMPLEMENTATION_ADDR");
    address L2_ERC1155_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC1155_GATEWAY_PROXY_ADDR");
    address L2_ERC1155_GATEWAY_IMPLEMENTATION_ADDR = vm.envAddress("L2_ERC1155_GATEWAY_IMPLEMENTATION_ADDR");
    address L2_ETH_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ETH_GATEWAY_PROXY_ADDR");
    address L2_ETH_GATEWAY_IMPLEMENTATION_ADDR = vm.envAddress("L2_ETH_GATEWAY_IMPLEMENTATION_ADDR");
    address L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L2_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR =
        vm.envAddress("L2_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR");
    address L2_WETH_GATEWAY_PROXY_ADDR = vm.envAddress("L2_WETH_GATEWAY_PROXY_ADDR");
    address L2_WETH_GATEWAY_IMPLEMENTATION_ADDR = vm.envAddress("L2_WETH_GATEWAY_IMPLEMENTATION_ADDR");
    address L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR = vm.envAddress("L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR");

    function run() external {
        ProxyAdmin proxyAdmin = ProxyAdmin(L2_PROXY_ADMIN_ADDR);

        vm.startBroadcast(deployerPrivateKey);

        // note: we use call upgrade(...) and initialize(...) instead of upgradeAndCall(...),
        // otherwise the contract owner would become ProxyAdmin.

        // initialize L2MessageQueue
        L2MessageQueue(L2_MESSAGE_QUEUE_ADDR).initialize(L2_SCROLL_MESSENGER_PROXY_ADDR);

        // initialize L2TxFeeVault
        L2TxFeeVault(payable(L2_TX_FEE_VAULT_ADDR)).updateMessenger(L2_SCROLL_MESSENGER_PROXY_ADDR);

        // initialize L1GasPriceOracle
        L1GasPriceOracle(L1_GAS_PRICE_ORACLE_ADDR).updateWhitelist(L2_WHITELIST_ADDR);

        // initialize L2ScrollMessenger
        proxyAdmin.upgrade(
            ITransparentUpgradeableProxy(L2_SCROLL_MESSENGER_PROXY_ADDR),
            L2_SCROLL_MESSENGER_IMPLEMENTATION_ADDR
        );

        L2ScrollMessenger(payable(L2_SCROLL_MESSENGER_PROXY_ADDR)).initialize(L1_SCROLL_MESSENGER_PROXY_ADDR);

        // initialize L2GatewayRouter
        L2GatewayRouter(L2_GATEWAY_ROUTER_PROXY_ADDR).initialize(
            L2_ETH_GATEWAY_PROXY_ADDR,
            L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR
        );

        // initialize L2CustomERC20Gateway
        proxyAdmin.upgrade(
            ITransparentUpgradeableProxy(L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR),
            L2_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR
        );

        L2CustomERC20Gateway(L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR).initialize(
            L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );

        // initialize L2ERC1155Gateway
        proxyAdmin.upgrade(
            ITransparentUpgradeableProxy(L2_ERC1155_GATEWAY_PROXY_ADDR),
            L2_ERC1155_GATEWAY_IMPLEMENTATION_ADDR
        );

        L2ERC1155Gateway(L2_ERC1155_GATEWAY_PROXY_ADDR).initialize(
            L1_ERC1155_GATEWAY_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );

        // initialize L2ERC721Gateway
        proxyAdmin.upgrade(
            ITransparentUpgradeableProxy(L2_ERC721_GATEWAY_PROXY_ADDR),
            L2_ERC721_GATEWAY_IMPLEMENTATION_ADDR
        );

        L2ERC721Gateway(L2_ERC721_GATEWAY_PROXY_ADDR).initialize(
            L1_ERC721_GATEWAY_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );

        // initialize L2ETHGateway
        proxyAdmin.upgrade(ITransparentUpgradeableProxy(L2_ETH_GATEWAY_PROXY_ADDR), L2_ETH_GATEWAY_IMPLEMENTATION_ADDR);

        L2ETHGateway(L2_ETH_GATEWAY_PROXY_ADDR).initialize(
            L1_ETH_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );

        // initialize L2StandardERC20Gateway
        proxyAdmin.upgrade(
            ITransparentUpgradeableProxy(L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR),
            L2_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR
        );

        L2StandardERC20Gateway(L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR).initialize(
            L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR,
            L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR
        );

        // initialize L2WETHGateway
        proxyAdmin.upgrade(
            ITransparentUpgradeableProxy(L2_WETH_GATEWAY_PROXY_ADDR),
            L2_WETH_GATEWAY_IMPLEMENTATION_ADDR
        );

        L2WETHGateway(payable(L2_WETH_GATEWAY_PROXY_ADDR)).initialize(
            L1_WETH_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );

        // set WETH gateway in router
        {
            address[] memory _tokens = new address[](1);
            _tokens[0] = L2_WETH_ADDR;
            address[] memory _gateways = new address[](1);
            _gateways[0] = L2_WETH_GATEWAY_PROXY_ADDR;
            L2GatewayRouter(L2_GATEWAY_ROUTER_PROXY_ADDR).setERC20Gateway(_tokens, _gateways);
        }

        // initialize ScrollStandardERC20Factory
        ScrollStandardERC20Factory(L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR).transferOwnership(
            L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR
        );

        vm.stopBroadcast();
    }
}
