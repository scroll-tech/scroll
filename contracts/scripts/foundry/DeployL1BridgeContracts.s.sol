// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import { Script } from "forge-std/Script.sol";
import { console} from "forge-std/console.sol";

import { ProxyAdmin } from "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import { TransparentUpgradeableProxy } from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import { L1CustomERC20Gateway } from "../../src/L1/gateways/L1CustomERC20Gateway.sol";
import { L1ERC1155Gateway } from "../../src/L1/gateways/L1ERC1155Gateway.sol";
import { L1ERC721Gateway } from "../../src/L1/gateways/L1ERC721Gateway.sol";
import { L1GatewayRouter } from "../../src/L1/gateways/L1GatewayRouter.sol";
import { L1ScrollMessenger } from "../../src/L1/L1ScrollMessenger.sol";
import { L1StandardERC20Gateway } from "../../src/L1/gateways/L1StandardERC20Gateway.sol";
import { RollupVerifier } from "../../src/libraries/verifier/RollupVerifier.sol";
import { ZKRollup } from "../../src/L1/rollup/ZKRollup.sol";

contract DeployL1BridgeContracts is Script {
    uint256 L1_DEPLOYER_PRIVATE_KEY = vm.envUint("L1_DEPLOYER_PRIVATE_KEY");
    ProxyAdmin proxyAdmin;

    function run() external {
        vm.startBroadcast(L1_DEPLOYER_PRIVATE_KEY);

        // note: the RollupVerifier library is deployed implicitly

        deployProxyAdmin();
        deployZKRollup();
        deployL1StandardERC20Gateway();
        deployL1GatewayRouter();
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

    function deployZKRollup() internal {
        ZKRollup impl = new ZKRollup();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(address(impl), address(proxyAdmin), new bytes(0));

        logAddress("L1_ZK_ROLLUP_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_ZK_ROLLUP_PROXY_ADDR", address(proxy));
    }

    function deployL1StandardERC20Gateway() internal {
        L1StandardERC20Gateway impl = new L1StandardERC20Gateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(address(impl), address(proxyAdmin), new bytes(0));

        logAddress("L1_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1GatewayRouter() internal {
        L1GatewayRouter impl = new L1GatewayRouter();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(address(impl), address(proxyAdmin), new bytes(0));

        logAddress("L1_GATEWAY_ROUTER_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_GATEWAY_ROUTER_PROXY_ADDR", address(proxy));
    }

    function deployL1ScrollMessenger() internal {
        L1ScrollMessenger impl = new L1ScrollMessenger();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(address(impl), address(proxyAdmin), new bytes(0));

        logAddress("L1_SCROLL_MESSENGER_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_SCROLL_MESSENGER_PROXY_ADDR", address(proxy));
    }

    function deployL1CustomERC20Gateway() internal {
        L1CustomERC20Gateway impl = new L1CustomERC20Gateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(address(impl), address(proxyAdmin), new bytes(0));

        logAddress("L1_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1ERC721Gateway() internal {
        L1ERC721Gateway impl = new L1ERC721Gateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(address(impl), address(proxyAdmin), new bytes(0));

        logAddress("L1_ERC721_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_ERC721_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1ERC1155Gateway() internal {
        L1ERC1155Gateway impl = new L1ERC1155Gateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(address(impl), address(proxyAdmin), new bytes(0));

        logAddress("L1_ERC1155_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_ERC1155_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function logAddress(string memory name, address addr) internal {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
