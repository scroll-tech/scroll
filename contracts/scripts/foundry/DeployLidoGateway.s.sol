// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {L1LidoGateway} from "../../src/lido/L1LidoGateway.sol";
import {L2LidoGateway} from "../../src/lido/L2LidoGateway.sol";

// solhint-disable state-visibility
// solhint-disable var-name-mixedcase

contract DeployLidoGateway is Script {
    string NETWORK = vm.envString("NETWORK");

    uint256 L1_DEPLOYER_PRIVATE_KEY = vm.envUint("L1_DEPLOYER_PRIVATE_KEY");

    uint256 L2_DEPLOYER_PRIVATE_KEY = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");

    address L1_WSTETH_ADDR = vm.envAddress("L1_WSTETH_ADDR");

    address L2_WSTETH_ADDR = vm.envAddress("L2_WSTETH_ADDR");

    address L1_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L1_SCROLL_MESSENGER_PROXY_ADDR");
    address L1_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L1_GATEWAY_ROUTER_PROXY_ADDR");
    address L1_LIDO_GATEWAY_PROXY_ADDR = vm.envAddress("L1_LIDO_GATEWAY_PROXY_ADDR");

    address L2_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L2_SCROLL_MESSENGER_PROXY_ADDR");
    address L2_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L2_GATEWAY_ROUTER_PROXY_ADDR");
    address L2_LIDO_GATEWAY_PROXY_ADDR = vm.envAddress("L2_LIDO_GATEWAY_PROXY_ADDR");

    function run() external {
        vm.startBroadcast(L2_DEPLOYER_PRIVATE_KEY);

        if (keccak256(abi.encodePacked(NETWORK)) == keccak256(abi.encodePacked("L1"))) {
            // deploy l1 lido gateway
            L1LidoGateway gateway = new L1LidoGateway(
                L1_WSTETH_ADDR,
                L2_WSTETH_ADDR,
                L2_LIDO_GATEWAY_PROXY_ADDR,
                L1_GATEWAY_ROUTER_PROXY_ADDR,
                L1_SCROLL_MESSENGER_PROXY_ADDR
            );
            logAddress("L1_LIDO_GATEWAY_IMPLEMENTATION_ADDR", address(gateway));
        } else if (keccak256(abi.encodePacked(NETWORK)) == keccak256(abi.encodePacked("L2"))) {
            // deploy l2 lido gateway
            L2LidoGateway gateway = new L2LidoGateway(
                L1_WSTETH_ADDR,
                L2_WSTETH_ADDR,
                L1_LIDO_GATEWAY_PROXY_ADDR,
                L2_GATEWAY_ROUTER_PROXY_ADDR,
                L2_SCROLL_MESSENGER_PROXY_ADDR
            );
            logAddress("L2_LIDO_GATEWAY_IMPLEMENTATION_ADDR", address(gateway));
        }

        vm.stopBroadcast();
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
