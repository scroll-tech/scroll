// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import { Script } from "forge-std/Script.sol";

import { L2CustomERC20Gateway } from "../../src/L2/gateways/L2CustomERC20Gateway.sol";
import { L2ERC1155Gateway } from "../../src/L2/gateways/L2ERC1155Gateway.sol";
import { L2ERC721Gateway } from "../../src/L2/gateways/L2ERC721Gateway.sol";
import { L2GatewayRouter } from "../../src/L2/gateways/L2GatewayRouter.sol";
import { L2StandardERC20Gateway } from "../../src/L2/gateways/L2StandardERC20Gateway.sol";
import { ScrollStandardERC20Factory } from "../../src/libraries/token/ScrollStandardERC20Factory.sol";

contract InitializeL2BridgeContracts is Script {
    uint256 deployerPrivateKey = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");

    address L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L1_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L1_GATEWAY_ROUTER_PROXY_ADDR");
    address L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L1_ERC721_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ERC721_GATEWAY_PROXY_ADDR");
    address L1_ERC1155_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ERC1155_GATEWAY_PROXY_ADDR");

    address L2_SCROLL_MESSENGER_ADDR = vm.envAddress("L2_SCROLL_MESSENGER_ADDR");
    address L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L2_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L2_GATEWAY_ROUTER_PROXY_ADDR");
    address L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR = vm.envAddress("L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR");
    address L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L2_ERC721_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC721_GATEWAY_PROXY_ADDR");
    address L2_ERC1155_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC1155_GATEWAY_PROXY_ADDR");

    function run() external {
        vm.startBroadcast(deployerPrivateKey);

        // initialize L2StandardERC20Gateway
        L2StandardERC20Gateway(L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR).initialize(
            L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_ADDR,
            L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR
        );

        // initialize L2GatewayRouter
        L2GatewayRouter(L2_GATEWAY_ROUTER_PROXY_ADDR).initialize(
            L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR,
            L1_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_ADDR
        );

        // initialize ScrollStandardERC20Factory
        ScrollStandardERC20Factory(L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR).transferOwnership(
            L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR
        );

        // initialize L2CustomERC20Gateway
        L2CustomERC20Gateway(L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR).initialize(
            L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_ADDR
        );

        // initialize L2ERC1155Gateway
        L2ERC1155Gateway(L2_ERC1155_GATEWAY_PROXY_ADDR).initialize(
            L1_ERC1155_GATEWAY_PROXY_ADDR,
            L2_SCROLL_MESSENGER_ADDR
        );

        // initialize L2ERC721Gateway
        L2ERC721Gateway(L2_ERC721_GATEWAY_PROXY_ADDR).initialize(
            L1_ERC721_GATEWAY_PROXY_ADDR,
            L2_SCROLL_MESSENGER_ADDR
        );

        vm.stopBroadcast();
    }
}
