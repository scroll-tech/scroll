// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.16;

// solhint-disable no-console

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {ProxyAdmin} from "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import {TransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import {EmptyContract} from "../../src/misc/EmptyContract.sol";

// solhint-disable state-visibility
// solhint-disable var-name-mixedcase

contract DeployL1BridgeProxyPlaceholder is Script {
    uint256 L1_DEPLOYER_PRIVATE_KEY = vm.envUint("L1_DEPLOYER_PRIVATE_KEY");

    ProxyAdmin proxyAdmin;
    EmptyContract placeholder;

    function run() external {
        vm.startBroadcast(L1_DEPLOYER_PRIVATE_KEY);

        deployProxyAdmin();
        deployPlaceHolder();
        deployL1MessageQueue();
        deployScrollChain();
        deployL1ETHGateway();
        deployL1WETHGateway();
        deployL1StandardERC20Gateway();
        deployL1ScrollMessenger();
        deployL1CustomERC20Gateway();
        deployL1ERC721Gateway();
        deployL1ERC1155Gateway();

        vm.stopBroadcast();
    }

    function deployProxyAdmin() internal {
        proxyAdmin = new ProxyAdmin();

        logAddress("L1_PROXY_ADMIN_ADDR", address(proxyAdmin));
    }

    function deployPlaceHolder() internal {
        placeholder = new EmptyContract();

        logAddress("L1_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR", address(placeholder));
    }

    function deployScrollChain() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_SCROLL_CHAIN_PROXY_ADDR", address(proxy));
    }

    function deployL1MessageQueue() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );
        logAddress("L1_MESSAGE_QUEUE_PROXY_ADDR", address(proxy));
    }

    function deployL1StandardERC20Gateway() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1ETHGateway() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_ETH_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1WETHGateway() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_WETH_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1ScrollMessenger() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_SCROLL_MESSENGER_PROXY_ADDR", address(proxy));
    }

    function deployL1CustomERC20Gateway() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1ERC721Gateway() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_ERC721_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1ERC1155Gateway() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_ERC1155_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
