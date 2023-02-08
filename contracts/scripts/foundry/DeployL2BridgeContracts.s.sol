// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import { Script } from "forge-std/Script.sol";
import { console} from "forge-std/console.sol";

import { ProxyAdmin } from "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import { TransparentUpgradeableProxy } from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import { L2CustomERC20Gateway } from "../../src/L2/gateways/L2CustomERC20Gateway.sol";
import { L2ERC1155Gateway } from "../../src/L2/gateways/L2ERC1155Gateway.sol";
import { L2ERC721Gateway } from "../../src/L2/gateways/L2ERC721Gateway.sol";
import { L2GatewayRouter } from "../../src/L2/gateways/L2GatewayRouter.sol";
import { L2ScrollMessenger } from "../../src/L2/L2ScrollMessenger.sol";
import { L2StandardERC20Gateway } from "../../src/L2/gateways/L2StandardERC20Gateway.sol";
import { Whitelist } from "../../src/L2/predeploys/Whitelist.sol";
import { ScrollStandardERC20 } from "../../src/libraries/token/ScrollStandardERC20.sol";
import { ScrollStandardERC20Factory } from "../../src/libraries/token/ScrollStandardERC20Factory.sol";

contract DeployL2BridgeContracts is Script {
    uint256 L2_DEPLOYER_PRIVATE_KEY = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");
    ProxyAdmin proxyAdmin;

    function run() external {
        vm.startBroadcast(L2_DEPLOYER_PRIVATE_KEY);

        deployL2ScrollMessenger();
        deployProxyAdmin();
        deployL2StandardERC20Gateway();
        deployL2GatewayRouter();
        deployScrollStandardERC20Factory();
        deployL2CustomERC20Gateway();
        deployL2ERC721Gateway();
        deployL2ERC1155Gateway();
        deployL2Whitelist();

        vm.stopBroadcast();
    }

    function deployL2ScrollMessenger() internal {
        address owner = vm.addr(L2_DEPLOYER_PRIVATE_KEY);
        L2ScrollMessenger l2ScrollMessenger = new L2ScrollMessenger(owner);

        logAddress("L2_SCROLL_MESSENGER_ADDR", address(l2ScrollMessenger));
    }

    function deployProxyAdmin() internal {
        proxyAdmin = new ProxyAdmin();

        logAddress("L2_PROXY_ADMIN_ADDR", address(proxyAdmin));
    }

    function deployL2StandardERC20Gateway() internal {
        L2StandardERC20Gateway impl = new L2StandardERC20Gateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(address(impl), address(proxyAdmin), new bytes(0));

        logAddress("L2_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL2GatewayRouter() internal {
        L2GatewayRouter impl = new L2GatewayRouter();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(address(impl), address(proxyAdmin), new bytes(0));

        logAddress("L2_GATEWAY_ROUTER_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L2_GATEWAY_ROUTER_PROXY_ADDR", address(proxy));
    }

    function deployScrollStandardERC20Factory() internal {
        ScrollStandardERC20 tokenImpl = new ScrollStandardERC20();
        ScrollStandardERC20Factory scrollStandardERC20Factory = new ScrollStandardERC20Factory(address(tokenImpl));

        logAddress("L2_SCROLL_STANDARD_ERC20_ADDR", address(tokenImpl));
        logAddress("L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR", address(scrollStandardERC20Factory));
    }

    function deployL2CustomERC20Gateway() internal {
        L2CustomERC20Gateway impl = new L2CustomERC20Gateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(address(impl), address(proxyAdmin), new bytes(0));

        logAddress("L2_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL2ERC721Gateway() internal {
        L2ERC721Gateway impl = new L2ERC721Gateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(address(impl), address(proxyAdmin), new bytes(0));

        logAddress("L2_ERC721_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L2_ERC721_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL2ERC1155Gateway() internal {
        L2ERC1155Gateway impl = new L2ERC1155Gateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(address(impl), address(proxyAdmin), new bytes(0));

        logAddress("L2_ERC1155_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L2_ERC1155_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL2Whitelist() internal {
        address owner = vm.addr(L2_DEPLOYER_PRIVATE_KEY);
        Whitelist whitelist = new Whitelist(owner);

        logAddress("L2_WHITELIST_ADDR", address(whitelist));
    }

    function logAddress(string memory name, address addr) internal {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
