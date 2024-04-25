// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {VmSafe} from "forge-std/Vm.sol";

import {ProxyAdmin} from "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import {TransparentUpgradeableProxy, ITransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import {EmptyContract} from "../../src/misc/EmptyContract.sol";

import {EnforcedTxGateway} from "../../src/L1/gateways/EnforcedTxGateway.sol";
import {L1CustomERC20Gateway} from "../../src/L1/gateways/L1CustomERC20Gateway.sol";
import {L1ERC1155Gateway} from "../../src/L1/gateways/L1ERC1155Gateway.sol";
import {L1ERC721Gateway} from "../../src/L1/gateways/L1ERC721Gateway.sol";
import {L1ETHGateway} from "../../src/L1/gateways/L1ETHGateway.sol";
import {L1GatewayRouter} from "../../src/L1/gateways/L1GatewayRouter.sol";
import {L1MessageQueueWithGasPriceOracle} from "../../src/L1/rollup/L1MessageQueueWithGasPriceOracle.sol";
import {L1ScrollMessenger} from "../../src/L1/L1ScrollMessenger.sol";
import {L1StandardERC20Gateway} from "../../src/L1/gateways/L1StandardERC20Gateway.sol";
import {L1WETHGateway} from "../../src/L1/gateways/L1WETHGateway.sol";
import {L2GasPriceOracle} from "../../src/L1/rollup/L2GasPriceOracle.sol";
import {MultipleVersionRollupVerifier} from "../../src/L1/rollup/MultipleVersionRollupVerifier.sol";
import {ScrollChain} from "../../src/L1/rollup/ScrollChain.sol";
import {ZkEvmVerifierV1} from "../../src/libraries/verifier/ZkEvmVerifierV1.sol";

import {L2CustomERC20Gateway} from "../../src/L2/gateways/L2CustomERC20Gateway.sol";
import {L2ERC1155Gateway} from "../../src/L2/gateways/L2ERC1155Gateway.sol";
import {L2ERC721Gateway} from "../../src/L2/gateways/L2ERC721Gateway.sol";
import {L2ETHGateway} from "../../src/L2/gateways/L2ETHGateway.sol";
import {L2GatewayRouter} from "../../src/L2/gateways/L2GatewayRouter.sol";
import {L2ScrollMessenger} from "../../src/L2/L2ScrollMessenger.sol";
import {L2StandardERC20Gateway} from "../../src/L2/gateways/L2StandardERC20Gateway.sol";
import {L2WETHGateway} from "../../src/L2/gateways/L2WETHGateway.sol";
import {L1GasPriceOracle} from "../../src/L2/predeploys/L1GasPriceOracle.sol";
import {L2MessageQueue} from "../../src/L2/predeploys/L2MessageQueue.sol";
import {L2TxFeeVault} from "../../src/L2/predeploys/L2TxFeeVault.sol";
import {Whitelist} from "../../src/L2/predeploys/Whitelist.sol";
import {WrappedEther} from "../../src/L2/predeploys/WrappedEther.sol";
import {ScrollStandardERC20} from "../../src/libraries/token/ScrollStandardERC20.sol";
import {ScrollStandardERC20Factory} from "../../src/libraries/token/ScrollStandardERC20Factory.sol";

/// @dev The address of DeterministicDeploymentProxy.
///      See https://github.com/Arachnid/deterministic-deployment-proxy.
address constant DETERMINISTIC_DEPLOYMENT_PROXY_ADDR = 0x4e59b44847b379578588920cA78FbF26c0B4956C;

/// @dev The default salt prefix used for deriving deterministic deployment addresses.
string constant DEFAULT_SALT_PREFIX = "ScrollStack";

contract ProxyAdminSetOwner is ProxyAdmin {
    /// @dev allow setting the owner in the constructor, otherwise
    ///      DeterministicDeploymentProxy would become the owner.
    constructor(address owner) {
        _transferOwnership(owner);
    }
}

contract ScrollStandardERC20FactorySetOwner is ScrollStandardERC20Factory {
    /// @dev allow setting the owner in the constructor, otherwise
    ///      DeterministicDeploymentProxy would become the owner.
    constructor(address owner, address _implementation) ScrollStandardERC20Factory(_implementation) {
        _transferOwnership(owner);
    }
}

contract DeterminsticDeploymentScript is Script {
    string internal saltPrefix;

    constructor(string memory _saltPrefix) {
        saltPrefix = _saltPrefix;
    }

    function deploy(string memory name, bytes memory code) internal returns (address) {
        return _deploy(name, code);
    }

    function deploy(
        string memory name,
        bytes memory code,
        bytes memory args
    ) internal returns (address) {
        return _deploy(name, abi.encodePacked(code, args));
    }

    function _deploy(string memory name, bytes memory codeWithArgs) private returns (address) {
        // check override (mainly used with predeploys)
        address addr = vm.envOr(string(abi.encodePacked(name, "_OVERRIDE")), address(0));

        if (addr != address(0)) {
            if (addr.code.length == 0) {
                (VmSafe.CallerMode mode, , ) = vm.readCallers();

                // if we're ready to start broadcasting transactions, then we
                // must ensure that the override contract has been deployed.
                if (mode == VmSafe.CallerMode.Broadcast || mode == VmSafe.CallerMode.RecurrentBroadcast) {
                    console.log(string(abi.encodePacked("[ERROR] ", name, "_OVERRIDE=", vm.toString(addr), " not deployed in broadcast mode")));
                    revert();
                }
            }

            logAddress(name, addr);
            return addr;
        }

        // predict determinstic deployment address
        addr = _predict(name, codeWithArgs);
        logAddress(name, addr);

        // return if the contract is already deployed,
        // in this case the subsequent initialization steps will probably break
        if (addr.code.length > 0) {
            console.log(string(abi.encodePacked("[WARN] contract ", name, " is already deployed")));
            return addr;
        }

        // deploy contract
        bytes32 salt = _getSalt(name);
        bytes memory data = abi.encodePacked(salt, codeWithArgs);
        (bool success, ) = DETERMINISTIC_DEPLOYMENT_PROXY_ADDR.call(data);
        require(success, "call failed");
        require(addr.code.length != 0, "deployment address mismatch");

        return addr;
    }

    function _getSalt(string memory name) internal view returns (bytes32) {
        return keccak256(abi.encodePacked(saltPrefix, name));
    }

    function _predict(string memory name, bytes memory codeWithArgs) private view returns (address) {
        bytes32 salt = _getSalt(name);

        return address(uint160(uint256(keccak256(abi.encodePacked(
            bytes1(0xff),
            DETERMINISTIC_DEPLOYMENT_PROXY_ADDR,
            salt,
            keccak256(codeWithArgs)
        )))));
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "_ADDR=", vm.toString(address(addr)))));
    }
}

contract DeployScroll is DeterminsticDeploymentScript {
    /**********************
     * Contracts to deploy *
     **********************/

    // L1 addresses
    address L1_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR;
    address L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR;
    address L1_ENFORCED_TX_GATEWAY_IMPLEMENTATION_ADDR;
    address L1_ENFORCED_TX_GATEWAY_PROXY_ADDR;
    address L1_ERC1155_GATEWAY_IMPLEMENTATION_ADDR;
    address L1_ERC1155_GATEWAY_PROXY_ADDR;
    address L1_ERC721_GATEWAY_IMPLEMENTATION_ADDR;
    address L1_ERC721_GATEWAY_PROXY_ADDR;
    address L1_ETH_GATEWAY_IMPLEMENTATION_ADDR;
    address L1_ETH_GATEWAY_PROXY_ADDR;
    address L1_GATEWAY_ROUTER_IMPLEMENTATION_ADDR;
    address L1_GATEWAY_ROUTER_PROXY_ADDR;
    address L1_MESSAGE_QUEUE_IMPLEMENTATION_ADDR;
    address L1_MESSAGE_QUEUE_PROXY_ADDR;
    address L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR;
    address L1_PROXY_ADMIN_ADDR;
    address L1_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR;
    address L1_SCROLL_CHAIN_IMPLEMENTATION_ADDR;
    address L1_SCROLL_CHAIN_PROXY_ADDR;
    address L1_SCROLL_MESSENGER_IMPLEMENTATION_ADDR;
    address L1_SCROLL_MESSENGER_PROXY_ADDR;
    address L1_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR;
    address L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR;
    address L1_WETH_ADDR;
    address L1_WETH_GATEWAY_IMPLEMENTATION_ADDR;
    address L1_WETH_GATEWAY_PROXY_ADDR;
    address L1_WHITELIST_ADDR;
    address L1_ZKEVM_VERIFIER_V1_ADDR;
    address L2_GAS_PRICE_ORACLE_IMPLEMENTATION_ADDR;
    address L2_GAS_PRICE_ORACLE_PROXY_ADDR;

    // L2 addresses
    address L1_GAS_PRICE_ORACLE_ADDR;
    address L2_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR;
    address L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR;
    address L2_ERC1155_GATEWAY_IMPLEMENTATION_ADDR;
    address L2_ERC1155_GATEWAY_PROXY_ADDR;
    address L2_ERC721_GATEWAY_IMPLEMENTATION_ADDR;
    address L2_ERC721_GATEWAY_PROXY_ADDR;
    address L2_ETH_GATEWAY_IMPLEMENTATION_ADDR;
    address L2_ETH_GATEWAY_PROXY_ADDR;
    address L2_GATEWAY_ROUTER_IMPLEMENTATION_ADDR;
    address L2_GATEWAY_ROUTER_PROXY_ADDR;
    address L2_MESSAGE_QUEUE_ADDR;
    address L2_PROXY_ADMIN_ADDR;
    address L2_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR;
    address L2_SCROLL_MESSENGER_IMPLEMENTATION_ADDR;
    address L2_SCROLL_MESSENGER_PROXY_ADDR;
    address L2_SCROLL_STANDARD_ERC20_ADDR;
    address L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR;
    address L2_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR;
    address L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR;
    address L2_TX_FEE_VAULT_ADDR;
    address L2_WETH_ADDR;
    address L2_WETH_GATEWAY_IMPLEMENTATION_ADDR;
    address L2_WETH_GATEWAY_PROXY_ADDR;
    address L2_WHITELIST_ADDR;

    /***************************
     * Configuration parameters *
     ***************************/

    // general configurations
    uint64 CHAIN_ID_L2 = uint64(vm.envUint("CHAIN_ID_L2"));
    uint256 MAX_TX_IN_CHUNK = vm.envUint("MAX_TX_IN_CHUNK");
    uint256 MAX_L1_MESSAGE_GAS_LIMIT = vm.envUint("MAX_L1_MESSAGE_GAS_LIMIT");

    string SALT_PREFIX = vm.envOr("SALT_PREFIX", DEFAULT_SALT_PREFIX);
    string BROADCAST_LAYER = vm.envOr("BROADCAST_LAYER", string(""));

    // accounts
    address L1_COMMIT_SENDER_ADDR = vm.envAddress("L1_COMMIT_SENDER_ADDR");
    address L1_FINALIZE_SENDER_ADDR = vm.envAddress("L1_FINALIZE_SENDER_ADDR");
    address L1_GAS_PRICE_ORACLE_SENDER_ADDR = vm.envAddress("L1_GAS_PRICE_ORACLE_SENDER_ADDR");
    address L2_GAS_PRICE_ORACLE_SENDER_ADDR = vm.envAddress("L2_GAS_PRICE_ORACLE_SENDER_ADDR");

    address DEPLOYER_ADDR; // implicit, derived from private key / wallet

    // contracts deployed outside this script
    address L1_FEE_VAULT_ADDR = vm.envAddress("L1_FEE_VAULT_ADDR");
    address L1_PLONK_VERIFIER_ADDR = vm.envAddress("L1_PLONK_VERIFIER_ADDR");

    /**************
     * Constructor *
     **************/

    constructor() DeterminsticDeploymentScript(SALT_PREFIX) {
        // empty
    }

    /************
     * Utilities *
     ************/

    /// @dev Ensure that `addr` is not the zero address.
    ///      This helps catch bugs arising from incorrect deployment order.
    function notnull(address addr) internal returns (address) {
        require(addr != address(0), "null address");
        return addr;
    }

    /// @dev Only broadcast code block if we run the script on the specified layer.
    modifier broadcast(string memory layer) {
        if (keccak256(bytes(BROADCAST_LAYER)) == keccak256(bytes(layer))) {
            vm.startBroadcast();
        } else {
            // make sure we use the correct sender in simulation
            vm.startPrank(DEPLOYER_ADDR);
        }

        _;

        if (keccak256(bytes(BROADCAST_LAYER)) == keccak256(bytes(layer))) {
            vm.stopBroadcast();
        } else {
            vm.stopPrank();
        }
    }

    /// @dev Only execute block if we run the script on the specified layer.
    modifier only(string memory layer) {
        if (keccak256(bytes(BROADCAST_LAYER)) != keccak256(bytes(layer))) {
            return;
        }
        _;
    }

    /*************************
     * Deployment entry point *
     *************************/

    function run() public {
        DEPLOYER_ADDR = msg.sender;
        logAddress("DEPLOYER", DEPLOYER_ADDR);

        if (DETERMINISTIC_DEPLOYMENT_PROXY_ADDR.code.length == 0) {
            console.log(string(abi.encodePacked("[ERROR] DeterministicDeploymentProxy (", vm.toString(DETERMINISTIC_DEPLOYMENT_PROXY_ADDR), ") is not available")));
            revert();
        }

        deployL1Contracts1stPass();
        deployL2Contracts1stPass();
        deployL1Contracts2ndPass();
        deployL2Contracts2ndPass();
        initializeL1Contracts();
        initializeL2Contracts();
    }

    // @notice deployL1Contracts1stPass deploys L1 contracts whose initialization does not depend on any L2 addresses.
    function deployL1Contracts1stPass() internal broadcast("L1") {
        deployL1Weth();
        deployL1ProxyAdmin();
        deployL1PlaceHolder();
        deployL1Whitelist();
        deployL2GasPriceOracle();
        deployL1ScrollChainProxy();
        deployL1ScrollMessengerProxy();
        deployL1EnforcedTxGateway();
        deployL1ZkEvmVerifierV1();
        deployL1MultipleVersionRollupVerifier();
        deployL1MessageQueue();
        deployL1ScrollChain();
        deployL1GatewayRouter();
        deployL1ETHGatewayProxy();
        deployL1WETHGatewayProxy();
        deployL1StandardERC20GatewayProxy();
        deployL1CustomERC20GatewayProxy();
        deployL1ERC721GatewayProxy();
        deployL1ERC1155GatewayProxy();
    }

    // @notice deployL2Contracts1stPass deploys L2 contracts whose initialization does not depend on any L1 addresses.
    function deployL2Contracts1stPass() internal broadcast("L2") {
        deployL2MessageQueue();
        deployL1GasPriceOracle();
        deployL2Whitelist();
        deployL2Weth();
        deployTxFeeVault();
        deployL2ProxyAdmin();
        deployL2PlaceHolder();
        deployL2ScrollMessengerProxy();
        deployL2ETHGatewayProxy();
        deployL2WETHGatewayProxy();
        deployL2StandardERC20GatewayProxy();
        deployL2CustomERC20GatewayProxy();
        deployL2ERC721GatewayProxy();
        deployL2ERC1155GatewayProxy();
        deployScrollStandardERC20Factory();
    }

    // @notice deployL1Contracts2ndPass deploys L1 contracts whose initialization depends on some L2 addresses.
    function deployL1Contracts2ndPass() internal broadcast("L1") {
        deployL1ScrollMessenger();
        deployL1StandardERC20Gateway();
        deployL1ETHGateway();
        deployL1WETHGateway();
        deployL1CustomERC20Gateway();
        deployL1ERC721Gateway();
        deployL1ERC1155Gateway();
    }

    // @notice deployL2Contracts2ndPass deploys L2 contracts whose initialization depends on some L1 addresses.
    function deployL2Contracts2ndPass() internal broadcast("L2") {
        // upgradable
        deployL2ScrollMessenger();
        deployL2GatewayRouter();
        deployL2StandardERC20Gateway();
        deployL2ETHGateway();
        deployL2WETHGateway();
        deployL2CustomERC20Gateway();
        deployL2ERC721Gateway();
        deployL2ERC1155Gateway();
    }

    // @notice initializeL1Contracts initializes contracts deployed on L1.
    function initializeL1Contracts() internal broadcast("L1") only("L1") {
        initializeScrollChain();
        initializeL2GasPriceOracle();
        initializeL1MessageQueue();
        initializeL1ScrollMessenger();
        initializeEnforcedTxGateway();
        initializeL1GatewayRouter();
        initializeL1CustomERC20Gateway();
        initializeL1ERC1155Gateway();
        initializeL1ERC721Gateway();
        initializeL1ETHGateway();
        initializeL1StandardERC20Gateway();
        initializeL1WETHGateway();
        initializeL1Whitelist();
    }

    // @notice initializeL2Contracts initializes contracts deployed on L2.
    function initializeL2Contracts() internal broadcast("L2") only("L2") {
        initializeL2MessageQueue();
        initializeL2TxFeeVault();
        initializeL1GasPriceOracle();
        initializeL2ScrollMessenger();
        initializeL2GatewayRouter();
        initializeL2CustomERC20Gateway();
        initializeL2ERC1155Gateway();
        initializeL2ERC721Gateway();
        initializeL2ETHGateway();
        initializeL2StandardERC20Gateway();
        initializeL2WETHGateway();
        initializeScrollStandardERC20Factory();
        initializeL2Whitelist();
    }

    /***************************
     * L1: 1st pass deployment *
     **************************/

    function deployL1Weth() internal {
        L1_WETH_ADDR = deploy("L1_WETH", type(WrappedEther).creationCode);
    }

    function deployL1ProxyAdmin() internal {
        bytes memory args = abi.encode(DEPLOYER_ADDR);
        L1_PROXY_ADMIN_ADDR = deploy("L1_PROXY_ADMIN", type(ProxyAdminSetOwner).creationCode, args);
    }

    function deployL1PlaceHolder() internal {
        L1_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR = deploy("L1_PROXY_IMPLEMENTATION_PLACEHOLDER", type(EmptyContract).creationCode);
    }

    function deployL1Whitelist() internal {
        bytes memory args = abi.encode(DEPLOYER_ADDR);
        L1_WHITELIST_ADDR = deploy("L1_WHITELIST", type(Whitelist).creationCode, args);
    }

    function deployL2GasPriceOracle() internal {
        L2_GAS_PRICE_ORACLE_IMPLEMENTATION_ADDR = deploy("L2_GAS_PRICE_ORACLE_IMPLEMENTATION", type(L2GasPriceOracle).creationCode);

        bytes memory args = abi.encode(
            notnull(L2_GAS_PRICE_ORACLE_IMPLEMENTATION_ADDR),
            notnull(L1_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L2_GAS_PRICE_ORACLE_PROXY_ADDR = deploy("L2_GAS_PRICE_ORACLE_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL1ScrollChainProxy() internal {
        bytes memory args = abi.encode(
            notnull(L1_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L1_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L1_SCROLL_CHAIN_PROXY_ADDR = deploy("L1_SCROLL_CHAIN_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL1ScrollMessengerProxy() internal {
        bytes memory args = abi.encode(
            notnull(L1_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L1_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L1_SCROLL_MESSENGER_PROXY_ADDR = deploy("L1_SCROLL_MESSENGER_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL1EnforcedTxGateway() internal {
        L1_ENFORCED_TX_GATEWAY_IMPLEMENTATION_ADDR = deploy("L1_ENFORCED_TX_GATEWAY_IMPLEMENTATION", type(EnforcedTxGateway).creationCode);

        bytes memory args = abi.encode(
            notnull(L1_ENFORCED_TX_GATEWAY_IMPLEMENTATION_ADDR),
            notnull(L1_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L1_ENFORCED_TX_GATEWAY_PROXY_ADDR = deploy("L1_ENFORCED_TX_GATEWAY_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL1ZkEvmVerifierV1() internal {
        logAddress("L1_PLONK_VERIFIER", L1_PLONK_VERIFIER_ADDR);
        bytes memory args = abi.encode(notnull(L1_PLONK_VERIFIER_ADDR));
        L1_ZKEVM_VERIFIER_V1_ADDR = deploy("L1_ZKEVM_VERIFIER_V1", type(ZkEvmVerifierV1).creationCode, args);
    }

    function deployL1MultipleVersionRollupVerifier() internal {
        uint256[] memory _versions = new uint256[](1);
        address[] memory _verifiers = new address[](1);
        _versions[0] = 1;
        _verifiers[0] = notnull(L1_ZKEVM_VERIFIER_V1_ADDR);

        bytes memory args = abi.encode(notnull(L1_SCROLL_CHAIN_PROXY_ADDR), _versions, _verifiers);
        L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR = deploy("L1_MULTIPLE_VERSION_ROLLUP_VERIFIER", type(MultipleVersionRollupVerifier).creationCode, args);
    }

    function deployL1MessageQueue() internal {
        bytes memory args = abi.encode(
            notnull(L1_SCROLL_MESSENGER_PROXY_ADDR),
            notnull(L1_SCROLL_CHAIN_PROXY_ADDR),
            notnull(L1_ENFORCED_TX_GATEWAY_PROXY_ADDR)
        );

        L1_MESSAGE_QUEUE_IMPLEMENTATION_ADDR = deploy("L1_MESSAGE_QUEUE_IMPLEMENTATION", type(L1MessageQueueWithGasPriceOracle).creationCode, args);

        bytes memory args2 = abi.encode(
            notnull(L1_MESSAGE_QUEUE_IMPLEMENTATION_ADDR),
            notnull(L1_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L1_MESSAGE_QUEUE_PROXY_ADDR = deploy("L1_MESSAGE_QUEUE_PROXY", type(TransparentUpgradeableProxy).creationCode, args2);
    }

    function deployL1ScrollChain() internal {
        bytes memory args = abi.encode(
            CHAIN_ID_L2,
            notnull(L1_MESSAGE_QUEUE_PROXY_ADDR),
            notnull(L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR)
        );

        L1_SCROLL_CHAIN_IMPLEMENTATION_ADDR = deploy("L1_SCROLL_CHAIN_IMPLEMENTATION", type(ScrollChain).creationCode, args);

        ProxyAdmin(L1_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L1_SCROLL_CHAIN_PROXY_ADDR)),
            notnull(L1_SCROLL_CHAIN_IMPLEMENTATION_ADDR)
        );
    }

    function deployL1GatewayRouter() internal {
        L1_GATEWAY_ROUTER_IMPLEMENTATION_ADDR = deploy("L1_GATEWAY_ROUTER_IMPLEMENTATION", type(L1GatewayRouter).creationCode);

        bytes memory args = abi.encode(
            notnull(L1_GATEWAY_ROUTER_IMPLEMENTATION_ADDR),
            notnull(L1_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L1_GATEWAY_ROUTER_PROXY_ADDR = deploy("L1_GATEWAY_ROUTER_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL1ETHGatewayProxy() internal {
        bytes memory args = abi.encode(
            notnull(L1_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L1_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L1_ETH_GATEWAY_PROXY_ADDR = deploy("L1_ETH_GATEWAY_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL1WETHGatewayProxy() internal {
        bytes memory args = abi.encode(
            notnull(L1_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L1_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L1_WETH_GATEWAY_PROXY_ADDR = deploy("L1_WETH_GATEWAY_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL1StandardERC20GatewayProxy() internal {
        bytes memory args = abi.encode(
            notnull(L1_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L1_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR = deploy("L1_STANDARD_ERC20_GATEWAY_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL1CustomERC20GatewayProxy() internal {
        bytes memory args = abi.encode(
            notnull(L1_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L1_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = deploy("L1_CUSTOM_ERC20_GATEWAY_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL1ERC721GatewayProxy() internal {
        bytes memory args = abi.encode(
            notnull(L1_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L1_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L1_ERC721_GATEWAY_PROXY_ADDR = deploy("L1_ERC721_GATEWAY_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL1ERC1155GatewayProxy() internal {
        bytes memory args = abi.encode(
            notnull(L1_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L1_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L1_ERC1155_GATEWAY_PROXY_ADDR = deploy("L1_ERC1155_GATEWAY_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    /***************************
     * L2: 1st pass deployment *
     **************************/

    function deployL2MessageQueue() internal {
        bytes memory args = abi.encode(DEPLOYER_ADDR);
        L2_MESSAGE_QUEUE_ADDR = deploy("L2_MESSAGE_QUEUE", type(L2MessageQueue).creationCode, args);
    }

    function deployL1GasPriceOracle() internal {
        bytes memory args = abi.encode(DEPLOYER_ADDR);
        L1_GAS_PRICE_ORACLE_ADDR = deploy("L1_GAS_PRICE_ORACLE", type(L1GasPriceOracle).creationCode, args);
    }

    function deployL2Whitelist() internal {
        bytes memory args = abi.encode(DEPLOYER_ADDR);
        L2_WHITELIST_ADDR = deploy("L2_WHITELIST", type(Whitelist).creationCode, args);
    }

    function deployL2Weth() internal {
        L2_WETH_ADDR = deploy("L2_WETH", type(WrappedEther).creationCode);
    }

    function deployTxFeeVault() internal {
        bytes memory args = abi.encode(
            DEPLOYER_ADDR,
            L1_FEE_VAULT_ADDR,
            10 ether
        );

        L2_TX_FEE_VAULT_ADDR = deploy("L2_TX_FEE_VAULT", type(L2TxFeeVault).creationCode, args);
    }

    function deployL2ProxyAdmin() internal {
        bytes memory args = abi.encode(DEPLOYER_ADDR);
        L2_PROXY_ADMIN_ADDR = deploy("L2_PROXY_ADMIN", type(ProxyAdminSetOwner).creationCode, args);
    }

    function deployL2PlaceHolder() internal {
        L2_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR = deploy("L2_PROXY_IMPLEMENTATION_PLACEHOLDER", type(EmptyContract).creationCode);
    }

    function deployL2ScrollMessengerProxy() internal {
        bytes memory args = abi.encode(
            notnull(L2_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L2_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L2_SCROLL_MESSENGER_PROXY_ADDR = deploy("L2_SCROLL_MESSENGER_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL2StandardERC20GatewayProxy() internal {
        bytes memory args = abi.encode(
            notnull(L2_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L2_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR = deploy("L2_STANDARD_ERC20_GATEWAY_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL2ETHGatewayProxy() internal {
        bytes memory args = abi.encode(
            notnull(L2_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L2_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L2_ETH_GATEWAY_PROXY_ADDR = deploy("L2_ETH_GATEWAY_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL2WETHGatewayProxy() internal {
        bytes memory args = abi.encode(
            notnull(L2_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L2_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L2_WETH_GATEWAY_PROXY_ADDR = deploy("L2_WETH_GATEWAY_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL2CustomERC20GatewayProxy() internal {
        bytes memory args = abi.encode(
            notnull(L2_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L2_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = deploy("L2_CUSTOM_ERC20_GATEWAY_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL2ERC721GatewayProxy() internal {
        bytes memory args = abi.encode(
            notnull(L2_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L2_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L2_ERC721_GATEWAY_PROXY_ADDR = deploy("L2_ERC721_GATEWAY_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL2ERC1155GatewayProxy() internal {
        bytes memory args = abi.encode(
            notnull(L2_PROXY_IMPLEMENTATION_PLACEHOLDER_ADDR),
            notnull(L2_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L2_ERC1155_GATEWAY_PROXY_ADDR = deploy("L2_ERC1155_GATEWAY_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployScrollStandardERC20Factory() internal {
        L2_SCROLL_STANDARD_ERC20_ADDR = deploy("L2_SCROLL_STANDARD_ERC20", type(ScrollStandardERC20).creationCode);

        bytes memory args = abi.encode(
            DEPLOYER_ADDR,
            notnull(L2_SCROLL_STANDARD_ERC20_ADDR)
        );

        L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR = deploy("L2_SCROLL_STANDARD_ERC20_FACTORY", type(ScrollStandardERC20FactorySetOwner).creationCode, args);
    }

    /***************************
     * L1: 2nd pass deployment *
     **************************/

    function deployL1ScrollMessenger() internal {
        bytes memory args = abi.encode(
            notnull(L2_SCROLL_MESSENGER_PROXY_ADDR),
            notnull(L1_SCROLL_CHAIN_PROXY_ADDR),
            notnull(L1_MESSAGE_QUEUE_PROXY_ADDR)
        );

        L1_SCROLL_MESSENGER_IMPLEMENTATION_ADDR = deploy("L1_SCROLL_MESSENGER_IMPLEMENTATION", type(L1ScrollMessenger).creationCode, args);

        ProxyAdmin(L1_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L1_SCROLL_MESSENGER_PROXY_ADDR)),
            notnull(L1_SCROLL_MESSENGER_IMPLEMENTATION_ADDR)
        );
    }

    function deployL1ETHGateway() internal {
        bytes memory args = abi.encode(
            notnull(L2_ETH_GATEWAY_PROXY_ADDR),
            notnull(L1_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L1_SCROLL_MESSENGER_PROXY_ADDR)
        );

        L1_ETH_GATEWAY_IMPLEMENTATION_ADDR = deploy("L1_ETH_GATEWAY_IMPLEMENTATION", type(L1ETHGateway).creationCode, args);

        ProxyAdmin(L1_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L1_ETH_GATEWAY_PROXY_ADDR)),
            notnull(L1_ETH_GATEWAY_IMPLEMENTATION_ADDR)
        );
    }

    function deployL1WETHGateway() internal {
        bytes memory args = abi.encode(
            notnull(L1_WETH_ADDR),
            notnull(L2_WETH_ADDR),
            notnull(L2_WETH_GATEWAY_PROXY_ADDR),
            notnull(L1_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L1_SCROLL_MESSENGER_PROXY_ADDR)
        );

        L1_WETH_GATEWAY_IMPLEMENTATION_ADDR = deploy("L1_WETH_GATEWAY_IMPLEMENTATION", type(L1WETHGateway).creationCode, args);

        ProxyAdmin(L1_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L1_WETH_GATEWAY_PROXY_ADDR)),
            notnull(L1_WETH_GATEWAY_IMPLEMENTATION_ADDR)
        );
    }

    function deployL1StandardERC20Gateway() internal {
        bytes memory args = abi.encode(
            notnull(L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR),
            notnull(L1_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L1_SCROLL_MESSENGER_PROXY_ADDR),
            notnull(L2_SCROLL_STANDARD_ERC20_ADDR),
            notnull(L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR)
        );

        L1_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR = deploy("L1_STANDARD_ERC20_GATEWAY_IMPLEMENTATION", type(L1StandardERC20Gateway).creationCode, args);

        ProxyAdmin(L1_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR)),
            notnull(L1_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR)
        );
    }

    function deployL1CustomERC20Gateway() internal {
        bytes memory args = abi.encode(
            notnull(L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR),
            notnull(L1_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L1_SCROLL_MESSENGER_PROXY_ADDR)
        );

        L1_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR = deploy("L1_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION", type(L1CustomERC20Gateway).creationCode, args);

        ProxyAdmin(L1_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR)),
            notnull(L1_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR)
        );
    }

    function deployL1ERC721Gateway() internal {
        bytes memory args = abi.encode(notnull(L2_ERC721_GATEWAY_PROXY_ADDR), notnull(L1_SCROLL_MESSENGER_PROXY_ADDR));

        L1_ERC721_GATEWAY_IMPLEMENTATION_ADDR = deploy("L1_ERC721_GATEWAY_IMPLEMENTATION", type(L1ERC721Gateway).creationCode, args);

        ProxyAdmin(L1_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L1_ERC721_GATEWAY_PROXY_ADDR)),
            notnull(L1_ERC721_GATEWAY_IMPLEMENTATION_ADDR)
        );
    }

    function deployL1ERC1155Gateway() internal {
        bytes memory args = abi.encode(notnull(L2_ERC1155_GATEWAY_PROXY_ADDR), notnull(L1_SCROLL_MESSENGER_PROXY_ADDR));

        L1_ERC1155_GATEWAY_IMPLEMENTATION_ADDR = deploy("L1_ERC1155_GATEWAY_IMPLEMENTATION", type(L1ERC1155Gateway).creationCode, args);

        ProxyAdmin(L1_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L1_ERC1155_GATEWAY_PROXY_ADDR)),
            notnull(L1_ERC1155_GATEWAY_IMPLEMENTATION_ADDR)
        );
    }

    /***************************
     * L2: 2nd pass deployment *
     **************************/

    function deployL2ScrollMessenger() internal {
        bytes memory args = abi.encode(notnull(L1_SCROLL_MESSENGER_PROXY_ADDR), notnull(L2_MESSAGE_QUEUE_ADDR));

        L2_SCROLL_MESSENGER_IMPLEMENTATION_ADDR = deploy("L2_SCROLL_MESSENGER_IMPLEMENTATION", type(L2ScrollMessenger).creationCode, args);

        ProxyAdmin(L2_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L2_SCROLL_MESSENGER_PROXY_ADDR)),
            notnull(L2_SCROLL_MESSENGER_IMPLEMENTATION_ADDR)
        );
    }

    function deployL2GatewayRouter() internal {
        L2_GATEWAY_ROUTER_IMPLEMENTATION_ADDR = deploy("L2_GATEWAY_ROUTER_IMPLEMENTATION", type(L2GatewayRouter).creationCode);

        bytes memory args = abi.encode(
            notnull(L2_GATEWAY_ROUTER_IMPLEMENTATION_ADDR),
            notnull(L2_PROXY_ADMIN_ADDR),
            new bytes(0)
        );

        L2_GATEWAY_ROUTER_PROXY_ADDR = deploy("L2_GATEWAY_ROUTER_PROXY", type(TransparentUpgradeableProxy).creationCode, args);
    }

    function deployL2StandardERC20Gateway() internal {
        bytes memory args = abi.encode(
            notnull(L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR),
            notnull(L2_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L2_SCROLL_MESSENGER_PROXY_ADDR),
            notnull(L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR)
        );

        L2_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR = deploy("L2_STANDARD_ERC20_GATEWAY_IMPLEMENTATION", type(L2StandardERC20Gateway).creationCode, args);

        ProxyAdmin(L2_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR)),
            notnull(L2_STANDARD_ERC20_GATEWAY_IMPLEMENTATION_ADDR)
        );
    }

    function deployL2ETHGateway() internal {
        bytes memory args = abi.encode(
            notnull(L1_ETH_GATEWAY_PROXY_ADDR),
            notnull(L2_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L2_SCROLL_MESSENGER_PROXY_ADDR)
        );

        L2_ETH_GATEWAY_IMPLEMENTATION_ADDR = deploy("L2_ETH_GATEWAY_IMPLEMENTATION", type(L2ETHGateway).creationCode, args);

        ProxyAdmin(L2_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L2_ETH_GATEWAY_PROXY_ADDR)),
            notnull(L2_ETH_GATEWAY_IMPLEMENTATION_ADDR)
        );
    }

    function deployL2WETHGateway() internal {
        bytes memory args = abi.encode(
            notnull(L2_WETH_ADDR),
            notnull(L1_WETH_ADDR),
            notnull(L1_WETH_GATEWAY_PROXY_ADDR),
            notnull(L2_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L2_SCROLL_MESSENGER_PROXY_ADDR)
        );

        L2_WETH_GATEWAY_IMPLEMENTATION_ADDR = deploy("L2_WETH_GATEWAY_IMPLEMENTATION", type(L2WETHGateway).creationCode, args);

        ProxyAdmin(L2_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L2_WETH_GATEWAY_PROXY_ADDR)),
            notnull(L2_WETH_GATEWAY_IMPLEMENTATION_ADDR)
        );
    }

    function deployL2CustomERC20Gateway() internal {
        bytes memory args = abi.encode(
            notnull(L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR),
            notnull(L2_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L2_SCROLL_MESSENGER_PROXY_ADDR)
        );

        L2_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR = deploy("L2_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION", type(L2CustomERC20Gateway).creationCode, args);

        ProxyAdmin(L2_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR)),
            notnull(L2_CUSTOM_ERC20_GATEWAY_IMPLEMENTATION_ADDR)
        );
    }

    function deployL2ERC721Gateway() internal {
        bytes memory args = abi.encode(notnull(L1_ERC721_GATEWAY_PROXY_ADDR), notnull(L2_SCROLL_MESSENGER_PROXY_ADDR));

        L2_ERC721_GATEWAY_IMPLEMENTATION_ADDR = deploy("L2_ERC721_GATEWAY_IMPLEMENTATION", type(L2ERC721Gateway).creationCode, args);

        ProxyAdmin(L2_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L2_ERC721_GATEWAY_PROXY_ADDR)),
            notnull(L2_ERC721_GATEWAY_IMPLEMENTATION_ADDR)
        );
    }

    function deployL2ERC1155Gateway() internal {
        bytes memory args = abi.encode(notnull(L1_ERC1155_GATEWAY_PROXY_ADDR), notnull(L2_SCROLL_MESSENGER_PROXY_ADDR));

        L2_ERC1155_GATEWAY_IMPLEMENTATION_ADDR = deploy("L2_ERC1155_GATEWAY_IMPLEMENTATION", type(L2ERC1155Gateway).creationCode, args);

        ProxyAdmin(L2_PROXY_ADMIN_ADDR).upgrade(
            ITransparentUpgradeableProxy(notnull(L2_ERC1155_GATEWAY_PROXY_ADDR)),
            notnull(L2_ERC1155_GATEWAY_IMPLEMENTATION_ADDR)
        );
    }

    /**********************
     * L1: initialization *
     *********************/

    function initializeScrollChain() internal {
        ScrollChain(L1_SCROLL_CHAIN_PROXY_ADDR).initialize(
            notnull(L1_MESSAGE_QUEUE_PROXY_ADDR),
            notnull(L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR),
            MAX_TX_IN_CHUNK
        );

        ScrollChain(L1_SCROLL_CHAIN_PROXY_ADDR).addSequencer(L1_COMMIT_SENDER_ADDR);
        ScrollChain(L1_SCROLL_CHAIN_PROXY_ADDR).addProver(L1_FINALIZE_SENDER_ADDR);
    }

    function initializeL2GasPriceOracle() internal {
        L2GasPriceOracle(L2_GAS_PRICE_ORACLE_PROXY_ADDR).initialize(
            21000, // _txGas
            53000, // _txGasContractCreation
            4, // _zeroGas
            16 // _nonZeroGas
        );

        L2GasPriceOracle(L2_GAS_PRICE_ORACLE_PROXY_ADDR).updateWhitelist(L1_WHITELIST_ADDR);
    }

    function initializeL1MessageQueue() internal {
        L1MessageQueueWithGasPriceOracle(L1_MESSAGE_QUEUE_PROXY_ADDR).initialize(
            notnull(L1_SCROLL_MESSENGER_PROXY_ADDR),
            notnull(L1_SCROLL_CHAIN_PROXY_ADDR),
            notnull(L1_ENFORCED_TX_GATEWAY_PROXY_ADDR),
            notnull(L2_GAS_PRICE_ORACLE_PROXY_ADDR),
            MAX_L1_MESSAGE_GAS_LIMIT
        );

        L1MessageQueueWithGasPriceOracle(L1_MESSAGE_QUEUE_PROXY_ADDR).initializeV2();
    }

    function initializeL1ScrollMessenger() internal {
        L1ScrollMessenger(payable(L1_SCROLL_MESSENGER_PROXY_ADDR)).initialize(
            notnull(L2_SCROLL_MESSENGER_PROXY_ADDR),
            notnull(L1_FEE_VAULT_ADDR),
            notnull(L1_SCROLL_CHAIN_PROXY_ADDR),
            notnull(L1_MESSAGE_QUEUE_PROXY_ADDR)
        );
    }

    function initializeEnforcedTxGateway() internal {
        EnforcedTxGateway(payable(L1_ENFORCED_TX_GATEWAY_PROXY_ADDR)).initialize(
            notnull(L1_MESSAGE_QUEUE_PROXY_ADDR),
            notnull(L1_FEE_VAULT_ADDR)
        );

        // disable gateway
        EnforcedTxGateway(payable(L1_ENFORCED_TX_GATEWAY_PROXY_ADDR)).setPause(true);
    }

    function initializeL1GatewayRouter() internal {
        L1GatewayRouter(L1_GATEWAY_ROUTER_PROXY_ADDR).initialize(
            notnull(L1_ETH_GATEWAY_PROXY_ADDR),
            notnull(L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR)
        );
    }

    function initializeL1CustomERC20Gateway() internal {
        L1CustomERC20Gateway(L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR).initialize(
            notnull(L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR),
            notnull(L1_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L1_SCROLL_MESSENGER_PROXY_ADDR)
        );
    }

    function initializeL1ERC1155Gateway() internal {
        L1ERC1155Gateway(L1_ERC1155_GATEWAY_PROXY_ADDR).initialize(
            notnull(L2_ERC1155_GATEWAY_PROXY_ADDR),
            notnull(L1_SCROLL_MESSENGER_PROXY_ADDR)
        );
    }

    function initializeL1ERC721Gateway() internal {
        L1ERC721Gateway(L1_ERC721_GATEWAY_PROXY_ADDR).initialize(
            notnull(L2_ERC721_GATEWAY_PROXY_ADDR),
            notnull(L1_SCROLL_MESSENGER_PROXY_ADDR)
        );
    }

    function initializeL1ETHGateway() internal {
        L1ETHGateway(L1_ETH_GATEWAY_PROXY_ADDR).initialize(
            notnull(L2_ETH_GATEWAY_PROXY_ADDR),
            notnull(L1_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L1_SCROLL_MESSENGER_PROXY_ADDR)
        );
    }

    function initializeL1StandardERC20Gateway() internal {
        L1StandardERC20Gateway(L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR).initialize(
            notnull(L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR),
            notnull(L1_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L1_SCROLL_MESSENGER_PROXY_ADDR),
            notnull(L2_SCROLL_STANDARD_ERC20_ADDR),
            notnull(L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR)
        );
    }

    function initializeL1WETHGateway() internal {
        L1WETHGateway(payable(L1_WETH_GATEWAY_PROXY_ADDR)).initialize(
            notnull(L2_WETH_GATEWAY_PROXY_ADDR),
            notnull(L1_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L1_SCROLL_MESSENGER_PROXY_ADDR)
        );

        // set WETH gateway in router
        {
            address[] memory _tokens = new address[](1);
            _tokens[0] = notnull(L1_WETH_ADDR);
            address[] memory _gateways = new address[](1);
            _gateways[0] = notnull(L1_WETH_GATEWAY_PROXY_ADDR);
            L1GatewayRouter(L1_GATEWAY_ROUTER_PROXY_ADDR).setERC20Gateway(_tokens, _gateways);
        }
    }

    function initializeL1Whitelist() internal {
        address[] memory accounts = new address[](1);
        accounts[0] = L1_GAS_PRICE_ORACLE_SENDER_ADDR;
        Whitelist(L1_WHITELIST_ADDR).updateWhitelistStatus(accounts, true);
    }

    /**********************
     * L2: initialization *
     *********************/

    function initializeL2MessageQueue() internal {
        L2MessageQueue(L2_MESSAGE_QUEUE_ADDR).initialize(notnull(L2_SCROLL_MESSENGER_PROXY_ADDR));
    }

    function initializeL2TxFeeVault() internal {
        L2TxFeeVault(payable(L2_TX_FEE_VAULT_ADDR)).updateMessenger(notnull(L2_SCROLL_MESSENGER_PROXY_ADDR));
    }

    function initializeL1GasPriceOracle() internal {
        L1GasPriceOracle(L1_GAS_PRICE_ORACLE_ADDR).updateWhitelist(notnull(L2_WHITELIST_ADDR));
    }

    function initializeL2ScrollMessenger() internal {
        L2ScrollMessenger(payable(L2_SCROLL_MESSENGER_PROXY_ADDR)).initialize(notnull(L1_SCROLL_MESSENGER_PROXY_ADDR));
    }

    function initializeL2GatewayRouter() internal {
        L2GatewayRouter(L2_GATEWAY_ROUTER_PROXY_ADDR).initialize(
            notnull(L2_ETH_GATEWAY_PROXY_ADDR),
            notnull(L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR)
        );
    }

    function initializeL2CustomERC20Gateway() internal {
        L2CustomERC20Gateway(L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR).initialize(
            notnull(L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR),
            notnull(L2_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L2_SCROLL_MESSENGER_PROXY_ADDR)
        );
    }

    function initializeL2ERC1155Gateway() internal {
        L2ERC1155Gateway(L2_ERC1155_GATEWAY_PROXY_ADDR).initialize(
            notnull(L1_ERC1155_GATEWAY_PROXY_ADDR),
            notnull(L2_SCROLL_MESSENGER_PROXY_ADDR)
        );
    }

    function initializeL2ERC721Gateway() internal {
        L2ERC721Gateway(L2_ERC721_GATEWAY_PROXY_ADDR).initialize(
            notnull(L1_ERC721_GATEWAY_PROXY_ADDR),
            notnull(L2_SCROLL_MESSENGER_PROXY_ADDR)
        );
    }

    function initializeL2ETHGateway() internal {
        L2ETHGateway(L2_ETH_GATEWAY_PROXY_ADDR).initialize(
            notnull(L1_ETH_GATEWAY_PROXY_ADDR),
            notnull(L2_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L2_SCROLL_MESSENGER_PROXY_ADDR)
        );
    }

    function initializeL2StandardERC20Gateway() internal {
        L2StandardERC20Gateway(L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR).initialize(
            notnull(L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR),
            notnull(L2_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L2_SCROLL_MESSENGER_PROXY_ADDR),
            notnull(L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR)
        );
    }

    function initializeL2WETHGateway() internal {
        L2WETHGateway(payable(L2_WETH_GATEWAY_PROXY_ADDR)).initialize(
            notnull(L1_WETH_GATEWAY_PROXY_ADDR),
            notnull(L2_GATEWAY_ROUTER_PROXY_ADDR),
            notnull(L2_SCROLL_MESSENGER_PROXY_ADDR)
        );

        // set WETH gateway in router
        {
            address[] memory _tokens = new address[](1);
            _tokens[0] = notnull(L2_WETH_ADDR);
            address[] memory _gateways = new address[](1);
            _gateways[0] = notnull(L2_WETH_GATEWAY_PROXY_ADDR);
            L2GatewayRouter(L2_GATEWAY_ROUTER_PROXY_ADDR).setERC20Gateway(_tokens, _gateways);
        }
    }

    function initializeScrollStandardERC20Factory() internal {
        ScrollStandardERC20Factory(L2_SCROLL_STANDARD_ERC20_FACTORY_ADDR).transferOwnership(
            notnull(L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR)
        );
    }

    function initializeL2Whitelist() internal {
        address[] memory accounts = new address[](1);
        accounts[0] = L2_GAS_PRICE_ORACLE_SENDER_ADDR;
        Whitelist(L2_WHITELIST_ADDR).updateWhitelistStatus(accounts, true);
    }
}
