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
import {L1DAIGateway} from "../../src/L1/gateways/L1DAIGateway.sol";
import {MultipleVersionRollupVerifier} from "../../src/L1/rollup/MultipleVersionRollupVerifier.sol";
import {ScrollChain} from "../../src/L1/rollup/ScrollChain.sol";
import {L1MessageQueue} from "../../src/L1/rollup/L1MessageQueue.sol";
import {L2GasPriceOracle} from "../../src/L1/rollup/L2GasPriceOracle.sol";
import {EnforcedTxGateway} from "../../src/L1/gateways/EnforcedTxGateway.sol";

contract InitializeL1BridgeContracts is Script {
    uint256 L1_DEPLOYER_PRIVATE_KEY = vm.envUint("L1_DEPLOYER_PRIVATE_KEY");

    uint256 CHAIN_ID_L2 = vm.envUint("CHAIN_ID_L2");
    uint256 MAX_L2_TX_IN_CHUNK = vm.envUint("MAX_L2_TX_IN_CHUNK");
    uint256 MAX_L1_MESSAGE_GAS_LIMIT = vm.envUint("MAX_L1_MESSAGE_GAS_LIMIT");
    address L1_COMMIT_SENDER_ADDRESS = vm.envAddress("L1_COMMIT_SENDER_ADDRESS");
    address L1_FINALIZE_SENDER_ADDRESS = vm.envAddress("L1_FINALIZE_SENDER_ADDRESS");
    address L1_FEE_VAULT_ADDR = vm.envAddress("L1_FEE_VAULT_ADDR");
    address L1_WETH_ADDR = vm.envAddress("L1_WETH_ADDR");

    address L1_DAI_ADDR = vm.envAddress("L1_DAI_ADDR");
    address L2_DAI_ADDR = vm.envAddress("L2_DAI_ADDR");

    address L1_WHITELIST_ADDR = vm.envAddress("L1_WHITELIST_ADDR");
    address L1_SCROLL_CHAIN_PROXY_ADDR = vm.envAddress("L1_SCROLL_CHAIN_PROXY_ADDR");
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
    address L1_DAI_GATEWAY_PROXY_ADDR = vm.envAddress("L1_DAI_GATEWAY_PROXY_ADDR");
    address L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR = vm.envAddress("L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR");
    address L1_ENFORCED_TX_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ENFORCED_TX_GATEWAY_PROXY_ADDR");

    address L2_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L2_SCROLL_MESSENGER_PROXY_ADDR");
    address L2_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L2_GATEWAY_ROUTER_PROXY_ADDR");
    address L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L2_ERC721_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC721_GATEWAY_PROXY_ADDR");
    address L2_ERC1155_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC1155_GATEWAY_PROXY_ADDR");
    address L2_ETH_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ETH_GATEWAY_PROXY_ADDR");
    address L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L2_WETH_GATEWAY_PROXY_ADDR = vm.envAddress("L2_WETH_GATEWAY_PROXY_ADDR");
    address L2_DAI_GATEWAY_PROXY_ADDR = vm.envAddress("L2_DAI_GATEWAY_PROXY_ADDR");
    address L2_SCROLL_STANDARD_ERC20_ADDR = vm.envAddress("L2_SCROLL_STANDARD_ERC20_ADDR");
    address L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR = vm.envAddress("L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR");

    function run() external {
        vm.startBroadcast(L1_DEPLOYER_PRIVATE_KEY);

        // initialize ScrollChain
        ScrollChain(L1_SCROLL_CHAIN_PROXY_ADDR).initialize(
            L1_MESSAGE_QUEUE_PROXY_ADDR,
            L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR,
            MAX_L2_TX_IN_CHUNK
        );
        ScrollChain(L1_SCROLL_CHAIN_PROXY_ADDR).addSequencer(L1_COMMIT_SENDER_ADDRESS);
        ScrollChain(L1_SCROLL_CHAIN_PROXY_ADDR).addProver(L1_FINALIZE_SENDER_ADDRESS);

        // initialize MultipleVersionRollupVerifier
        MultipleVersionRollupVerifier(L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR).initialize(L1_SCROLL_CHAIN_PROXY_ADDR);

        // initialize L2GasPriceOracle
        L2GasPriceOracle(L2_GAS_PRICE_ORACLE_PROXY_ADDR).initialize(
            21000, // _txGas
            53000, // _txGasContractCreation
            4, // _zeroGas
            16 // _nonZeroGas
        );
        L2GasPriceOracle(L2_GAS_PRICE_ORACLE_PROXY_ADDR).updateWhitelist(L1_WHITELIST_ADDR);

        // initialize L1MessageQueue
        L1MessageQueue(L1_MESSAGE_QUEUE_PROXY_ADDR).initialize(
            L1_SCROLL_MESSENGER_PROXY_ADDR,
            L1_SCROLL_CHAIN_PROXY_ADDR,
            L1_ENFORCED_TX_GATEWAY_PROXY_ADDR,
            L2_GAS_PRICE_ORACLE_PROXY_ADDR,
            MAX_L1_MESSAGE_GAS_LIMIT
        );

        // initialize L1ScrollMessenger
        L1ScrollMessenger(payable(L1_SCROLL_MESSENGER_PROXY_ADDR)).initialize(
            L2_SCROLL_MESSENGER_PROXY_ADDR,
            L1_FEE_VAULT_ADDR,
            L1_SCROLL_CHAIN_PROXY_ADDR,
            L1_MESSAGE_QUEUE_PROXY_ADDR
        );

        // initialize EnforcedTxGateway
        EnforcedTxGateway(payable(L1_ENFORCED_TX_GATEWAY_PROXY_ADDR)).initialize(
            L1_MESSAGE_QUEUE_PROXY_ADDR,
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

        // initialize L1DAIGateway
        L1DAIGateway(L1_DAI_GATEWAY_PROXY_ADDR).initialize(
            L2_DAI_GATEWAY_PROXY_ADDR,
            L1_GATEWAY_ROUTER_PROXY_ADDR,
            L1_SCROLL_MESSENGER_PROXY_ADDR
        );
        L1DAIGateway(L1_DAI_GATEWAY_PROXY_ADDR).updateTokenMapping(L1_DAI_ADDR, L2_DAI_ADDR);

        // set WETH and DAI gateways in router
        {
            address[] memory _tokens = new address[](2);
            _tokens[0] = L1_WETH_ADDR;
            _tokens[1] = L1_DAI_ADDR;
            address[] memory _gateways = new address[](2);
            _gateways[0] = L1_WETH_GATEWAY_PROXY_ADDR;
            _gateways[1] = L1_DAI_GATEWAY_PROXY_ADDR;
            L1GatewayRouter(L1_GATEWAY_ROUTER_PROXY_ADDR).setERC20Gateway(_tokens, _gateways);
        }

        vm.stopBroadcast();
    }
}
