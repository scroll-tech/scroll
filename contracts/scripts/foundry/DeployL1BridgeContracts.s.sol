// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {ProxyAdmin} from "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import {TransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import {EnforcedTxGateway} from "../../src/L1/gateways/EnforcedTxGateway.sol";
import {L1CustomERC20Gateway} from "../../src/L1/gateways/L1CustomERC20Gateway.sol";
import {L1ERC1155Gateway} from "../../src/L1/gateways/L1ERC1155Gateway.sol";
import {L1ERC721Gateway} from "../../src/L1/gateways/L1ERC721Gateway.sol";
import {L1ETHGateway} from "../../src/L1/gateways/L1ETHGateway.sol";
import {L1GatewayRouter} from "../../src/L1/gateways/L1GatewayRouter.sol";
import {L1MessageQueue} from "../../src/L1/rollup/L1MessageQueue.sol";
import {L1ScrollMessenger} from "../../src/L1/L1ScrollMessenger.sol";
import {L1StandardERC20Gateway} from "../../src/L1/gateways/L1StandardERC20Gateway.sol";
import {L1WETHGateway} from "../../src/L1/gateways/L1WETHGateway.sol";
import {L1DAIGateway} from "../../src/L1/gateways/L1DAIGateway.sol";
import {L2GasPriceOracle} from "../../src/L1/rollup/L2GasPriceOracle.sol";
import {MultipleVersionRollupVerifier} from "../../src/L1/rollup/MultipleVersionRollupVerifier.sol";
import {ScrollChain} from "../../src/L1/rollup/ScrollChain.sol";
import {Whitelist} from "../../src/L2/predeploys/Whitelist.sol";
import {ZkEvmVerifierV1} from "../../src/libraries/verifier/ZkEvmVerifierV1.sol";

contract DeployL1BridgeContracts is Script {
    uint256 L1_DEPLOYER_PRIVATE_KEY = vm.envUint("L1_DEPLOYER_PRIVATE_KEY");

    uint32 CHAIN_ID_L2 = uint32(vm.envUint("CHAIN_ID_L2"));

    address L1_WETH_ADDR = vm.envAddress("L1_WETH_ADDR");
    address L2_WETH_ADDR = vm.envAddress("L2_WETH_ADDR");

    address L1_PLONK_VERIFIER_ADDR = vm.envAddress("L1_PLONK_VERIFIER_ADDR");

    ZkEvmVerifierV1 zkEvmVerifierV1;
    ProxyAdmin proxyAdmin;

    function run() external {
        vm.startBroadcast(L1_DEPLOYER_PRIVATE_KEY);

        deployZkEvmVerifierV1();
        deployMultipleVersionRollupVerifier();
        deployProxyAdmin();
        deployL1Whitelist();
        deployL1MessageQueue();
        deployL2GasPriceOracle();
        deployScrollChain();
        deployL1ETHGateway();
        deployL1WETHGateway();
        deployL1StandardERC20Gateway();
        deployL1GatewayRouter();
        deployL1ScrollMessenger();
        deployEnforcedTxGateway();
        deployL1CustomERC20Gateway();
        deployL1ERC721Gateway();
        deployL1ERC1155Gateway();
        deployL1DAIGateway();

        vm.stopBroadcast();
    }

    function deployZkEvmVerifierV1() internal {
        zkEvmVerifierV1 = new ZkEvmVerifierV1(L1_PLONK_VERIFIER_ADDR);

        logAddress("L1_ZKEVM_VERIFIER_V1_ADDR", address(zkEvmVerifierV1));
    }

    function deployMultipleVersionRollupVerifier() internal {
        MultipleVersionRollupVerifier rollupVerifier = new MultipleVersionRollupVerifier(address(zkEvmVerifierV1));

        logAddress("L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR", address(rollupVerifier));
    }

    function deployProxyAdmin() internal {
        proxyAdmin = new ProxyAdmin();

        logAddress("L1_PROXY_ADMIN_ADDR", address(proxyAdmin));
    }

    function deployL1Whitelist() internal {
        address owner = vm.addr(L1_DEPLOYER_PRIVATE_KEY);
        Whitelist whitelist = new Whitelist(owner);

        logAddress("L1_WHITELIST_ADDR", address(whitelist));
    }

    function deployScrollChain() internal {
        ScrollChain impl = new ScrollChain(CHAIN_ID_L2);
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(impl),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_SCROLL_CHAIN_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_SCROLL_CHAIN_PROXY_ADDR", address(proxy));
    }

    function deployL1MessageQueue() internal {
        L1MessageQueue impl = new L1MessageQueue();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(impl),
            address(proxyAdmin),
            new bytes(0)
        );
        logAddress("L1_MESSAGE_QUEUE_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_MESSAGE_QUEUE_PROXY_ADDR", address(proxy));
    }

    function deployL2GasPriceOracle() internal {
        L2GasPriceOracle impl = new L2GasPriceOracle();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(impl),
            address(proxyAdmin),
            new bytes(0)
        );
        logAddress("L2_GAS_PRICE_ORACLE_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L2_GAS_PRICE_ORACLE_PROXY_ADDR", address(proxy));
    }

    function deployL1StandardERC20Gateway() internal {
        L1StandardERC20Gateway impl = new L1StandardERC20Gateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(impl),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1ETHGateway() internal {
        L1ETHGateway impl = new L1ETHGateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(impl),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_ETH_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_ETH_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1WETHGateway() internal {
        L1WETHGateway impl = new L1WETHGateway(L1_WETH_ADDR, L2_WETH_ADDR);
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(impl),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_WETH_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_WETH_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1GatewayRouter() internal {
        L1GatewayRouter impl = new L1GatewayRouter();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(impl),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_GATEWAY_ROUTER_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_GATEWAY_ROUTER_PROXY_ADDR", address(proxy));
    }

    function deployL1ScrollMessenger() internal {
        L1ScrollMessenger impl = new L1ScrollMessenger();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(impl),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_SCROLL_MESSENGER_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_SCROLL_MESSENGER_PROXY_ADDR", address(proxy));
    }

    function deployEnforcedTxGateway() internal {
        EnforcedTxGateway impl = new EnforcedTxGateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(impl),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_ENFORCED_TX_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_ENFORCED_TX_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1CustomERC20Gateway() internal {
        L1CustomERC20Gateway impl = new L1CustomERC20Gateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(impl),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1DAIGateway() internal {
        L1DAIGateway impl = new L1DAIGateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(impl),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_DAI_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_DAI_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1ERC721Gateway() internal {
        L1ERC721Gateway impl = new L1ERC721Gateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(impl),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_ERC721_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_ERC721_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function deployL1ERC1155Gateway() internal {
        L1ERC1155Gateway impl = new L1ERC1155Gateway();
        TransparentUpgradeableProxy proxy = new TransparentUpgradeableProxy(
            address(impl),
            address(proxyAdmin),
            new bytes(0)
        );

        logAddress("L1_ERC1155_GATEWAY_IMPLEMENTATION_ADDR", address(impl));
        logAddress("L1_ERC1155_GATEWAY_PROXY_ADDR", address(proxy));
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
