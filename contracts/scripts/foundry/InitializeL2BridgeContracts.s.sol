// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import {Script} from "forge-std/Script.sol";

import {L2ScrollMessenger} from "../../src/L2/L2ScrollMessenger.sol";
import {L2CustomERC20Gateway} from "../../src/L2/gateways/L2CustomERC20Gateway.sol";
import {L2ERC1155Gateway} from "../../src/L2/gateways/L2ERC1155Gateway.sol";
import {L2ERC721Gateway} from "../../src/L2/gateways/L2ERC721Gateway.sol";
import {L2ETHGateway} from "../../src/L2/gateways/L2ETHGateway.sol";
import {L2GatewayRouter} from "../../src/L2/gateways/L2GatewayRouter.sol";
import {L2StandardERC20Gateway} from "../../src/L2/gateways/L2StandardERC20Gateway.sol";
import {L2WETHGateway} from "../../src/L2/gateways/L2WETHGateway.sol";
import {L2DAIGateway} from "../../src/L2/gateways/L2DAIGateway.sol";
import {L2MessageQueue} from "../../src/L2/predeploys/L2MessageQueue.sol";
import {L2TxFeeVault} from "../../src/L2/predeploys/L2TxFeeVault.sol";
import {L1GasPriceOracle} from "../../src/L2/predeploys/L1GasPriceOracle.sol";
import {Whitelist} from "../../src/L2/predeploys/Whitelist.sol";
import {ScrollStandardERC20Factory} from "../../src/libraries/token/ScrollStandardERC20Factory.sol";

contract InitializeL2BridgeContracts is Script {
    uint256 deployerPrivateKey = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");

    address L2_WETH_ADDR = vm.envAddress("L2_WETH_ADDR");

    address L1_DAI_ADDR = vm.envAddress("L1_DAI_ADDR");
    address L2_DAI_ADDR = vm.envAddress("L2_DAI_ADDR");

    address L1_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L1_SCROLL_MESSENGER_PROXY_ADDR");
    address L1_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L1_GATEWAY_ROUTER_PROXY_ADDR");
    address L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L1_ERC721_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ERC721_GATEWAY_PROXY_ADDR");
    address L1_ERC1155_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ERC1155_GATEWAY_PROXY_ADDR");
    address L1_ETH_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ETH_GATEWAY_PROXY_ADDR");
    address L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L1_WETH_GATEWAY_PROXY_ADDR = vm.envAddress("L1_WETH_GATEWAY_PROXY_ADDR");
    address L1_DAI_GATEWAY_PROXY_ADDR = vm.envAddress("L1_DAI_GATEWAY_PROXY_ADDR");

    address L2_TX_FEE_VAULT_ADDR = vm.envAddress("L2_TX_FEE_VAULT_ADDR");
    address L1_GAS_PRICE_ORACLE_ADDR = vm.envAddress("L1_GAS_PRICE_ORACLE_ADDR");
    address L2_WHITELIST_ADDR = vm.envAddress("L2_WHITELIST_ADDR");
    address L2_MESSAGE_QUEUE_ADDR = vm.envAddress("L2_MESSAGE_QUEUE_ADDR");

    address L2_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L2_SCROLL_MESSENGER_PROXY_ADDR");
    address L2_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L2_GATEWAY_ROUTER_PROXY_ADDR");
    address L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L2_ERC721_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC721_GATEWAY_PROXY_ADDR");
    address L2_ERC1155_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC1155_GATEWAY_PROXY_ADDR");
    address L2_ETH_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ETH_GATEWAY_PROXY_ADDR");
    address L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L2_WETH_GATEWAY_PROXY_ADDR = vm.envAddress("L2_WETH_GATEWAY_PROXY_ADDR");
    address L2_DAI_GATEWAY_PROXY_ADDR = vm.envAddress("L2_DAI_GATEWAY_PROXY_ADDR");
    address L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR = vm.envAddress("L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR");

    function run() external {
        vm.startBroadcast(deployerPrivateKey);

        // initialize L2MessageQueue
        L2MessageQueue(L2_MESSAGE_QUEUE_ADDR).initialize(L2_SCROLL_MESSENGER_PROXY_ADDR);

        // initialize L2TxFeeVault
        L2TxFeeVault(payable(L2_TX_FEE_VAULT_ADDR)).updateMessenger(L2_SCROLL_MESSENGER_PROXY_ADDR);

        // initialize L1GasPriceOracle
        L1GasPriceOracle(L1_GAS_PRICE_ORACLE_ADDR).updateWhitelist(L2_WHITELIST_ADDR);

        // initialize L2ScrollMessenger
        L2ScrollMessenger(payable(L2_SCROLL_MESSENGER_PROXY_ADDR)).initialize(L1_SCROLL_MESSENGER_PROXY_ADDR);

        // initialize L2GatewayRouter
        L2GatewayRouter(L2_GATEWAY_ROUTER_PROXY_ADDR).initialize(
            L2_ETH_GATEWAY_PROXY_ADDR,
            L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR
        );

        // initialize L2CustomERC20Gateway
        L2CustomERC20Gateway(L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR).initialize(
            L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );

        // initialize L2ERC1155Gateway
        L2ERC1155Gateway(L2_ERC1155_GATEWAY_PROXY_ADDR).initialize(
            L1_ERC1155_GATEWAY_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );

        // initialize L2ERC721Gateway
        L2ERC721Gateway(L2_ERC721_GATEWAY_PROXY_ADDR).initialize(
            L1_ERC721_GATEWAY_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );

        // initialize L2ETHGateway
        L2ETHGateway(L2_ETH_GATEWAY_PROXY_ADDR).initialize(
            L1_ETH_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );

        // initialize L2StandardERC20Gateway
        L2StandardERC20Gateway(L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR).initialize(
            L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR,
            L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR
        );

        // initialize L2WETHGateway
        L2WETHGateway(payable(L2_WETH_GATEWAY_PROXY_ADDR)).initialize(
            L1_WETH_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );

        // initialize L2DAIGateway
        L2DAIGateway(L2_DAI_GATEWAY_PROXY_ADDR).initialize(
            L1_DAI_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );
        L2DAIGateway(L2_DAI_GATEWAY_PROXY_ADDR).updateTokenMapping(L2_DAI_ADDR, L1_DAI_ADDR);

        // set WETH and DAI gateways in router
        {
            address[] memory _tokens = new address[](2);
            _tokens[0] = L2_WETH_ADDR;
            _tokens[1] = L2_DAI_ADDR;
            address[] memory _gateways = new address[](2);
            _gateways[0] = L2_WETH_GATEWAY_PROXY_ADDR;
            _gateways[1] = L2_DAI_GATEWAY_PROXY_ADDR;
            L2GatewayRouter(L2_GATEWAY_ROUTER_PROXY_ADDR).setERC20Gateway(_tokens, _gateways);
        }

        // initialize ScrollStandardERC20Factory
        ScrollStandardERC20Factory(L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR).transferOwnership(
            L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR
        );

        vm.stopBroadcast();
    }
}
