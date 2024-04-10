// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.24;

// solhint-disable no-console

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {ProxyAdmin} from "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import {TransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import {EmptyContract} from "../../src/misc/EmptyContract.sol";

// solhint-disable state-visibility
// solhint-disable var-name-mixedcase

contract DeployL2BridgeProxyPlaceholder is Script {
    uint256 L2_DEPLOYER_PRIVATE_KEY = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");

    ProxyAdmin proxyAdmin;
    EmptyContract placeholder;

    function run() external {
        vm.startBroadcast(L2_DEPLOYER_PRIVATE_KEY);

        // upgradable
        deployProxyAdmin();
        deployPlaceHolder();
        deployL2ScrollMessenger();
        deployL2ETHGateway();
        deployL2WETHGateway();
        deployL2StandardERC20Gateway();
        deployL2CustomERC20Gateway();
        deployL2ERC721Gateway();
        deployL2ERC1155Gateway();

        vm.stopBroadcast();
    }

    function deployProxyAdmin() internal {
        proxyAdmin = new ProxyAdmin();

        logAddress("L2_PROXY_ADMIN_ADDR", address(proxyAdmin));
    }

    function deployPlaceHolder() internal {
        placeholder = new EmptyContract();

        logAddress("L2_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR", address(placeholder));
    }

    function deployL2ScrollMessenger() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L2_SCROLL_MESSENGER_PROXY_ADDR", address(proxy));
    }

    function deployL2StandardERC20Gateway() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL2ETHGateway() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L2_ETH_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL2WETHGateway() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L2_WETH_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL2CustomERC20Gateway() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL2ERC721Gateway() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L2_ERC721_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL2ERC1155Gateway() internal {
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(placeholder),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L2_ERC1155_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
