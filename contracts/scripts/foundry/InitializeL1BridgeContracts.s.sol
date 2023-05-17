// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import {Script} from "forge-std/Script.sol";

import {L1CustomERC20Gateway} from "../../src/L1/gateways/L1CustomERC20Gateway.sol";
import {L1ERC1155Gateway} from "../../src/L1/gateways/L1ERC1155Gateway.sol";
import {L1ERC721Gateway} from "../../src/L1/gateways/L1ERC721Gateway.sol";
import {L1ETHGateway} from "../../src/L1/gateways/L1ETHGateway.sol";
import {L1GatewayRouter} from "../../src/L1/gateways/L1GatewayRouter.sol";
import {L1ScrollMessenger} from "../../src/L1/L1ScrollMessenger.sol";
import {L1StandardERC20Gateway} from "../../src/L1/gateways/L1StandardERC20Gateway.sol";
import {L1WETHGateway} from "../../src/L1/gateways/L1WETHGateway.sol";
import {ScrollChain} from "../../src/L1/rollup/ScrollChain.sol";
import {L1MessageQueue} from "../../src/L1/rollup/L1MessageQueue.sol";
import {L2GasPriceOracle} from "../../src/L1/rollup/L2GasPriceOracle.sol";
import {EnforcedTxGateway} from "../../src/L1/gateways/EnforcedTxGateway.sol";

contract InitializeL1BridgeContracts is Script {
    uint256 L1_DEPLOYER_PRIVATE_KEY = vm.envUint("L1_DEPLOYER_PRIVATE_KEY");

    uint256 CHAIN_ID_L2 = vm.envUint("CHAIN_ID_L2");
    address L1_ROLLUP_OPERATOR_ADDR = vm.envAddress("L1_ROLLUP_OPERATOR_ADDR");

    address L1_FEE_VAULT_ADDR = vm.envAddress("L1_FEE_VAULT_ADDR");

    address L1_WHITELIST_ADDR = vm.envAddress("L1_WHITELIST_ADDR");
    address L1_ZK_ROLLUP_PROXY_ADDR = vm.envAddress("L1_ZK_ROLLUP_PROXY_ADDR");
    address L1_MESSAGE_QUEUE_PROXY_ADDR = vm.envAddress("L1_MESSAGE_QUEUE_PROXY_ADDR");
    address L2_GAS_PRICE_ORACLE_PROXY_ADDR = vm.envAddress("L2_GAS_PRICE_ORACLE_PROXY_ADDR");
    address L1_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L1_SCROLL_MESSENGER_PROXY_ADDR");
    address L1_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L1_GATEWAY_ROUTER_PROXY_ADDR");
    address L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L1_ERC721_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ERC721_GATEWAY_PROXY_ADDR");
    address L1_ERC1155_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ERC1155_GATEWAY_PROXY_ADDR");
    address L1_ETH_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ETH_GATEWAY_PROXY_ADDR");
    address L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L1_WETH_GATEWAY_PROXY_ADDR = vm.envAddress("L1_WETH_GATEWAY_PROXY_ADDR");
    address ENFORCED_TX_GATEWAY_PROXY_ADDR = vm.envAddress("ENFORCED_TX_GATEWAY_PROXY_ADDR");

    address L2_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L2_SCROLL_MESSENGER_PROXY_ADDR");
    address L2_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L2_GATEWAY_ROUTER_PROXY_ADDR");
    address L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L2_ERC721_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC721_GATEWAY_PROXY_ADDR");
    address L2_ERC1155_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC1155_GATEWAY_PROXY_ADDR");
    address L2_ETH_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ETH_GATEWAY_PROXY_ADDR");
    address L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L2_WETH_GATEWAY_PROXY_ADDR = vm.envAddress("L2_WETH_GATEWAY_PROXY_ADDR");
    address L2_SCROLL_STANDARD_ERC20_ADDR = vm.envAddress("L2_SCROLL_STANDARD_ERC20_ADDR");
    address L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR = vm.envAddress("L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR");

    function run() external {
        vm.startBroadcast(L1_DEPLOYER_PRIVATE_KEY);

        // initialize ScrollChain
        ScrollChain(L1_ZK_ROLLUP_PROXY_ADDR).initialize(L1_MESSAGE_QUEUE_PROXY_ADDR);
        ScrollChain(L1_ZK_ROLLUP_PROXY_ADDR).updateSequencer(L1_ROLLUP_OPERATOR_ADDR, true);

        // initialize L2GasPriceOracle
        L2GasPriceOracle(L2_GAS_PRICE_ORACLE_PROXY_ADDR).initialize(0, 0, 0, 0);
        L2GasPriceOracle(L2_GAS_PRICE_ORACLE_PROXY_ADDR).updateWhitelist(L1_WHITELIST_ADDR);

        // initialize L1MessageQueue
        L1MessageQueue(L1_MESSAGE_QUEUE_PROXY_ADDR).initialize(
            L1_SCROLL_MESSENGER_PROXY_ADDR,
            ENFORCED_TX_GATEWAY_PROXY_ADDR,
            L2_GAS_PRICE_ORACLE_PROXY_ADDR,
            10000000
        );

        // initialize L1ScrollMessenger
        L1ScrollMessenger(payable(L1_SCROLL_MESSENGER_PROXY_ADDR)).initialize(
            L2_SCROLL_MESSENGER_PROXY_ADDR,
            L1_FEE_VAULT_ADDR,
            L1_ZK_ROLLUP_PROXY_ADDR,
            L1_MESSAGE_QUEUE_PROXY_ADDR
        );
        L1ScrollMessenger(payable(L1_SCROLL_MESSENGER_PROXY_ADDR)).updateWhitelist(L1_WHITELIST_ADDR);

        // initialize EnforcedTxGateway
        EnforcedTxGateway(payable(ENFORCED_TX_GATEWAY_PROXY_ADDR)).initialize(
            L2_SCROLL_MESSENGER_PROXY_ADDR,
            L1_FEE_VAULT_ADDR
        );

        // initialize L1GatewayRouter
        L1GatewayRouter(L1_GATEWAY_ROUTER_PROXY_ADDR).initialize(
            L1_ETH_GATEWAY_PROXY_ADDR,
            L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR
        );

        // initialize L1CustomERC20Gateway
        L1CustomERC20Gateway(L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR).initialize(
            L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR,
            L1_GATEWAY_ROUTER_PROXY_ADDR,
            L1_SCROLL_MESSENGER_PROXY_ADDR
        );

        // initialize L1ERC1155Gateway
        L1ERC1155Gateway(L1_ERC1155_GATEWAY_PROXY_ADDR).initialize(
            L2_ERC1155_GATEWAY_PROXY_ADDR,
            L1_SCROLL_MESSENGER_PROXY_ADDR
        );

        // initialize L1ERC721Gateway
        L1ERC721Gateway(L1_ERC721_GATEWAY_PROXY_ADDR).initialize(
            L2_ERC721_GATEWAY_PROXY_ADDR,
            L1_SCROLL_MESSENGER_PROXY_ADDR
        );

        // initialize L1ETHGateway
        L1ETHGateway(L1_ETH_GATEWAY_PROXY_ADDR).initialize(
            L2_ETH_GATEWAY_PROXY_ADDR,
            L1_GATEWAY_ROUTER_PROXY_ADDR,
            L1_SCROLL_MESSENGER_PROXY_ADDR
        );

        // initialize L1StandardERC20Gateway
        L1StandardERC20Gateway(L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR).initialize(
            L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR,
            L1_GATEWAY_ROUTER_PROXY_ADDR,
            L1_SCROLL_MESSENGER_PROXY_ADDR,
            L2_SCROLL_STANDARD_ERC20_ADDR,
            L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR
        );

        // initialize L1WETHGateway
        L1WETHGateway(payable(L1_WETH_GATEWAY_PROXY_ADDR)).initialize(
            L2_WETH_GATEWAY_PROXY_ADDR,
            L1_GATEWAY_ROUTER_PROXY_ADDR,
            L1_SCROLL_MESSENGER_PROXY_ADDR
        );

        vm.stopBroadcast();
    }
}
