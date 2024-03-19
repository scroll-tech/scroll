// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.16;

// solhint-disable no-console

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

// L1 contracts
import {L1USDCGateway} from "../../src/L1/gateways/usdc/L1USDCGateway.sol";
import {L1CustomERC20Gateway} from "../../src/L1/gateways/L1CustomERC20Gateway.sol";
import {L1ERC1155Gateway} from "../../src/L1/gateways/L1ERC1155Gateway.sol";
import {L1ERC721Gateway} from "../../src/L1/gateways/L1ERC721Gateway.sol";
import {L1ETHGateway} from "../../src/L1/gateways/L1ETHGateway.sol";
import {L1StandardERC20Gateway} from "../../src/L1/gateways/L1StandardERC20Gateway.sol";
import {L1WETHGateway} from "../../src/L1/gateways/L1WETHGateway.sol";
import {ScrollChain} from "../../src/L1/rollup/ScrollChain.sol";
import {L1MessageQueueWithGasPriceOracle} from "../../src/L1/rollup/L1MessageQueueWithGasPriceOracle.sol";
import {L1ScrollMessenger} from "../../src/L1/L1ScrollMessenger.sol";

import {L2USDCGateway} from "../../src/L2/gateways/usdc/L2USDCGateway.sol";
import {L2CustomERC20Gateway} from "../../src/L2/gateways/L2CustomERC20Gateway.sol";
import {L2ERC1155Gateway} from "../../src/L2/gateways/L2ERC1155Gateway.sol";
import {L2ERC721Gateway} from "../../src/L2/gateways/L2ERC721Gateway.sol";
import {L2ETHGateway} from "../../src/L2/gateways/L2ETHGateway.sol";
import {L2StandardERC20Gateway} from "../../src/L2/gateways/L2StandardERC20Gateway.sol";
import {L2WETHGateway} from "../../src/L2/gateways/L2WETHGateway.sol";
import {L2ScrollMessenger} from "../../src/L2/L2ScrollMessenger.sol";

// solhint-disable max-states-count
// solhint-disable state-visibility
// solhint-disable var-name-mixedcase

contract DeployBridgeOptimizationContracts is Script {
    string NETWORK = vm.envString("NETWORK");

    uint256 L1_DEPLOYER_PRIVATE_KEY = vm.envUint("L1_DEPLOYER_PRIVATE_KEY");
    uint256 L2_DEPLOYER_PRIVATE_KEY = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");

    address L1_WETH_ADDR = vm.envAddress("L1_WETH_ADDR");
    address L2_WETH_ADDR = vm.envAddress("L2_WETH_ADDR");

    uint64 CHAIN_ID_L2 = uint64(vm.envUint("CHAIN_ID_L2"));

    address L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR = vm.envAddress("L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR");
    address L1_ENFORCED_TX_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ENFORCED_TX_GATEWAY_PROXY_ADDR");

    address L2_SCROLL_STANDARD_ERC20_ADDR = vm.envAddress("L2_SCROLL_STANDARD_ERC20_ADDR");
    address L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR = vm.envAddress("L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR");
    address L2_MESSAGE_QUEUE_ADDR = vm.envAddress("L2_MESSAGE_QUEUE_ADDR");

    // proxy contracts in L1
    address L1_SCROLL_CHAIN_PROXY_ADDR = vm.envAddress("L1_SCROLL_CHAIN_PROXY_ADDR");
    address L1_MESSAGE_QUEUE_PROXY_ADDR = vm.envAddress("L1_MESSAGE_QUEUE_PROXY_ADDR");
    address L1_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L1_SCROLL_MESSENGER_PROXY_ADDR");
    address L1_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L1_GATEWAY_ROUTER_PROXY_ADDR");
    address L1_WETH_GATEWAY_PROXY_ADDR = vm.envAddress("L1_WETH_GATEWAY_PROXY_ADDR");
    address L1_ETH_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ETH_GATEWAY_PROXY_ADDR");
    address L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L1_ERC721_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ERC721_GATEWAY_PROXY_ADDR");
    address L1_ERC1155_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ERC1155_GATEWAY_PROXY_ADDR");
    address L1_USDC_GATEWAY_PROXY_ADDR = vm.envAddress("L1_USDC_GATEWAY_PROXY_ADDR");

    // proxy contracts in L2
    address L2_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L2_SCROLL_MESSENGER_PROXY_ADDR");
    address L2_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L2_GATEWAY_ROUTER_PROXY_ADDR");
    address L2_WETH_GATEWAY_PROXY_ADDR = vm.envAddress("L2_WETH_GATEWAY_PROXY_ADDR");
    address L2_ETH_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ETH_GATEWAY_PROXY_ADDR");
    address L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L2_ERC721_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC721_GATEWAY_PROXY_ADDR");
    address L2_ERC1155_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC1155_GATEWAY_PROXY_ADDR");
    address L2_USDC_GATEWAY_PROXY_ADDR = vm.envAddress("L2_USDC_GATEWAY_PROXY_ADDR");

    function run() external {
        if (keccak256(abi.encodePacked(NETWORK)) == keccak256(abi.encodePacked("L1"))) {
            deployL1Contracts();
        } else if (keccak256(abi.encodePacked(NETWORK)) == keccak256(abi.encodePacked("L2"))) {
            deployL2Contracts();
        }
    }

    function deployL1Contracts() private {
        vm.startBroadcast(L1_DEPLOYER_PRIVATE_KEY);

        // deploy L1ScrollMessenger impl
        L1ScrollMessenger implL1ScrollMessenger = new L1ScrollMessenger(
            L2_SCROLL_MESSENGER_PROXY_ADDR,
            L1_SCROLL_CHAIN_PROXY_ADDR,
            L1_MESSAGE_QUEUE_PROXY_ADDR
        );
        logAddress("L1_SCROLL_MESSENGER_IMPLEMENTATION_ADDR", address(implL1ScrollMessenger));

        // depoly ScrollChain impl
        ScrollChain implScrollChain = new ScrollChain(
            CHAIN_ID_L2,
            L1_MESSAGE_QUEUE_PROXY_ADDR,
            L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR
        );
        logAddress("L1_SCROLL_CHAIN_IMPLEMENTATION_ADDR", address(implScrollChain));

        // deploy L1MessageQueueWithGasPriceOracle impl
        L1MessageQueueWithGasPriceOracle implL1MessageQueue = new L1MessageQueueWithGasPriceOracle(
            L1_SCROLL_MESSENGER_PROXY_ADDR,
            L1_SCROLL_CHAIN_PROXY_ADDR,
            L1_ENFORCED_TX_GATEWAY_PROXY_ADDR
        );
        logAddress("L1_MESSAGE_QUEUE_IMPLEMENTATION_ADDR", address(implL1MessageQueue));

        // deploy L1StandardERC20Gateway impl
        L1StandardERC20Gateway implL1StandardERC20Gateway = new L1StandardERC20Gateway(
            L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR,
            L1_GATEWAY_ROUTER_PROXY_ADDR,
            L1_SCROLL_MESSENGER_PROXY_ADDR,
            L2_SCROLL_STANDARD_ERC20_ADDR,
            L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR
        );
        logAddress("L1_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR", address(implL1StandardERC20Gateway));

        // deploy L1ETHGateway impl
        L1ETHGateway implL1ETHGateway = new L1ETHGateway(
            L2_ETH_GATEWAY_PROXY_ADDR,
            L1_GATEWAY_ROUTER_PROXY_ADDR,
            L1_SCROLL_MESSENGER_PROXY_ADDR
        );
        logAddress("L1_ETH_GATEWAY_IMPLEMENTATION_ADDR", address(implL1ETHGateway));

        // deploy L1WETHGateway impl
        L1WETHGateway implL1WETHGateway = new L1WETHGateway(
            L1_WETH_ADDR,
            L2_WETH_ADDR,
            L2_WETH_GATEWAY_PROXY_ADDR,
            L1_GATEWAY_ROUTER_PROXY_ADDR,
            L1_SCROLL_MESSENGER_PROXY_ADDR
        );
        logAddress("L1_WETH_GATEWAY_IMPLEMENTATION_ADDR", address(implL1WETHGateway));

        // deploy L1CustomERC20Gateway impl
        L1CustomERC20Gateway implL1CustomERC20Gateway = new L1CustomERC20Gateway(
            L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR,
            L1_GATEWAY_ROUTER_PROXY_ADDR,
            L1_SCROLL_MESSENGER_PROXY_ADDR
        );
        logAddress("L1_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR", address(implL1CustomERC20Gateway));

        // deploy L1ERC721Gateway impl
        L1ERC721Gateway implL1ERC721Gateway = new L1ERC721Gateway(
            L2_ERC721_GATEWAY_PROXY_ADDR,
            L1_SCROLL_MESSENGER_PROXY_ADDR
        );
        logAddress("L1_ERC721_GATEWAY_IMPLEMENTATION_ADDR", address(implL1ERC721Gateway));

        // deploy L1ERC1155Gateway impl
        L1ERC1155Gateway implL1ERC1155Gateway = new L1ERC1155Gateway(
            L2_ERC1155_GATEWAY_PROXY_ADDR,
            L1_SCROLL_MESSENGER_PROXY_ADDR
        );
        logAddress("L1_ERC1155_GATEWAY_IMPLEMENTATION_ADDR", address(implL1ERC1155Gateway));

        // deploy L1USDCGateway impl only in mainnet
        if (CHAIN_ID_L2 != 534351) {
            address L1_USDC_ADDR = vm.envAddress("L1_USDC_ADDR");
            address L2_USDC_PROXY_ADDR = vm.envAddress("L2_USDC_PROXY_ADDR");
            L1USDCGateway implL1USDCGateway = new L1USDCGateway(
                L1_USDC_ADDR,
                L2_USDC_PROXY_ADDR,
                L2_USDC_GATEWAY_PROXY_ADDR,
                L1_GATEWAY_ROUTER_PROXY_ADDR,
                L1_SCROLL_MESSENGER_PROXY_ADDR
            );
            logAddress("L1_USDC_GATEWAY_IMPLEMENTATION_ADDR", address(implL1USDCGateway));
        }

        vm.stopBroadcast();
    }

    function deployL2Contracts() private {
        vm.startBroadcast(L2_DEPLOYER_PRIVATE_KEY);

        // deploy L2ScrollMessenger impl
        L2ScrollMessenger implL2ScrollMessenger = new L2ScrollMessenger(
            L1_SCROLL_MESSENGER_PROXY_ADDR,
            L2_MESSAGE_QUEUE_ADDR
        );
        logAddress("L2_SCROLL_MESSENGER_IMPLEMENTATION_ADDR", address(implL2ScrollMessenger));

        // deploy L2StandardERC20Gateway impl
        L2StandardERC20Gateway implL2StandardERC20Gateway = new L2StandardERC20Gateway(
            L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR,
            L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR
        );
        logAddress("L2_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR", address(implL2StandardERC20Gateway));

        // deploy L2ETHGateway impl
        L2ETHGateway implL2ETHGateway = new L2ETHGateway(
            L1_ETH_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );
        logAddress("L2_ETH_GATEWAY_IMPLEMENTATION_ADDR", address(implL2ETHGateway));

        // deploy L2WETHGateway impl
        L2WETHGateway implL2WETHGateway = new L2WETHGateway(
            L2_WETH_ADDR,
            L1_WETH_ADDR,
            L1_WETH_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );
        logAddress("L2_WETH_GATEWAY_IMPLEMENTATION_ADDR", address(implL2WETHGateway));

        // deploy L2CustomERC20Gateway impl
        L2CustomERC20Gateway implL2CustomERC20Gateway = new L2CustomERC20Gateway(
            L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR,
            L2_GATEWAY_ROUTER_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );
        logAddress("L2_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR", address(implL2CustomERC20Gateway));

        // deploy L2ERC721Gateway impl
        L2ERC721Gateway implL2ERC721Gateway = new L2ERC721Gateway(
            L1_ERC721_GATEWAY_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );
        logAddress("L2_ERC721_GATEWAY_IMPLEMENTATION_ADDR", address(implL2ERC721Gateway));

        // deploy L2ERC1155Gateway impl
        L2ERC1155Gateway implL2ERC1155Gateway = new L2ERC1155Gateway(
            L1_ERC1155_GATEWAY_PROXY_ADDR,
            L2_SCROLL_MESSENGER_PROXY_ADDR
        );
        logAddress("L2_ERC1155_GATEWAY_IMPLEMENTATION_ADDR", address(implL2ERC1155Gateway));

        // deploy L2USDCGateway impl only in mainnet
        if (CHAIN_ID_L2 != 534351) {
            address L1_USDC_ADDR = vm.envAddress("L1_USDC_ADDR");
            address L2_USDC_PROXY_ADDR = vm.envAddress("L2_USDC_PROXY_ADDR");
            L2USDCGateway implL2USDCGateway = new L2USDCGateway(
                L1_USDC_ADDR,
                L2_USDC_PROXY_ADDR,
                L1_USDC_GATEWAY_PROXY_ADDR,
                L2_GATEWAY_ROUTER_PROXY_ADDR,
                L2_SCROLL_MESSENGER_PROXY_ADDR
            );
            logAddress("L2_USDC_GATEWAY_IMPLEMENTATION_ADDR", address(implL2USDCGateway));
        }

        vm.stopBroadcast();
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
