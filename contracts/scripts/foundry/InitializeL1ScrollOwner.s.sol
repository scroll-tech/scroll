// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.16;

import {Script} from "forge-std/Script.sol";

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {ProxyAdmin} from "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";

import {L1ScrollMessenger} from "../../src/L1/L1ScrollMessenger.sol";
import {L1USDCGateway} from "../../src/L1/gateways/usdc/L1USDCGateway.sol";
import {EnforcedTxGateway} from "../../src/L1/gateways/EnforcedTxGateway.sol";
import {L1CustomERC20Gateway} from "../../src/L1/gateways/L1CustomERC20Gateway.sol";
import {L1ERC1155Gateway} from "../../src/L1/gateways/L1ERC1155Gateway.sol";
import {L1ERC721Gateway} from "../../src/L1/gateways/L1ERC721Gateway.sol";
import {L1GatewayRouter} from "../../src/L1/gateways/L1GatewayRouter.sol";
import {L1MessageQueue} from "../../src/L1/rollup/L1MessageQueue.sol";
import {ScrollMessengerBase} from "../../src/libraries/ScrollMessengerBase.sol";
import {L2GasPriceOracle} from "../../src/L1/rollup/L2GasPriceOracle.sol";
import {MultipleVersionRollupVerifier} from "../../src/L1/rollup/MultipleVersionRollupVerifier.sol";
import {ScrollChain} from "../../src/L1/rollup/ScrollChain.sol";
import {ScrollOwner} from "../../src/misc/ScrollOwner.sol";
import {Whitelist} from "../../src/L2/predeploys/Whitelist.sol";

// solhint-disable max-states-count
// solhint-disable state-visibility
// solhint-disable var-name-mixedcase

contract InitializeL1ScrollOwner is Script {
    uint256 L1_DEPLOYER_PRIVATE_KEY = vm.envUint("L1_DEPLOYER_PRIVATE_KEY");

    bytes32 constant SECURITY_COUNCIL_NO_DELAY_ROLE = keccak256("SECURITY_COUNCIL_NO_DELAY_ROLE");
    bytes32 constant SCROLL_MULTISIG_NO_DELAY_ROLE = keccak256("SCROLL_MULTISIG_NO_DELAY_ROLE");
    bytes32 constant EMERGENCY_MULTISIG_NO_DELAY_ROLE = keccak256("EMERGENCY_MULTISIG_NO_DELAY_ROLE");

    bytes32 constant TIMELOCK_1DAY_DELAY_ROLE = keccak256("TIMELOCK_1DAY_DELAY_ROLE");
    bytes32 constant TIMELOCK_7DAY_DELAY_ROLE = keccak256("TIMELOCK_7DAY_DELAY_ROLE");

    address SCROLL_MULTISIG_ADDR = vm.envAddress("L1_SCROLL_MULTISIG_ADDR");
    address SECURITY_COUNCIL_ADDR = vm.envAddress("L1_SECURITY_COUNCIL_ADDR");
    address EMERGENCY_MULTISIG_ADDR = vm.envAddress("L1_EMERGENCY_MULTISIG_ADDR");

    address L1_SCROLL_OWNER_ADDR = vm.envAddress("L1_SCROLL_OWNER_ADDR");
    address L1_1D_TIMELOCK_ADDR = vm.envAddress("L1_1D_TIMELOCK_ADDR");
    address L1_7D_TIMELOCK_ADDR = vm.envAddress("L1_7D_TIMELOCK_ADDR");
    address L1_14D_TIMELOCK_ADDR = vm.envAddress("L1_14D_TIMELOCK_ADDR");

    address L1_PROXY_ADMIN_ADDR = vm.envAddress("L1_PROXY_ADMIN_ADDR");
    address L1_SCROLL_CHAIN_PROXY_ADDR = vm.envAddress("L1_SCROLL_CHAIN_PROXY_ADDR");
    address L1_MESSAGE_QUEUE_PROXY_ADDR = vm.envAddress("L1_MESSAGE_QUEUE_PROXY_ADDR");
    address L2_GAS_PRICE_ORACLE_PROXY_ADDR = vm.envAddress("L2_GAS_PRICE_ORACLE_PROXY_ADDR");
    address L1_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L1_SCROLL_MESSENGER_PROXY_ADDR");
    address L1_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L1_GATEWAY_ROUTER_PROXY_ADDR");
    address L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L1_DAI_GATEWAY_PROXY_ADDR = vm.envAddress("L1_DAI_GATEWAY_PROXY_ADDR");
    address L1_LIDO_GATEWAY_PROXY_ADDR = vm.envAddress("L1_LIDO_GATEWAY_PROXY_ADDR");
    address L1_ETH_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ETH_GATEWAY_PROXY_ADDR");
    address L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L1_USDC_GATEWAY_PROXY_ADDR = vm.envAddress("L1_USDC_GATEWAY_PROXY_ADDR");
    address L1_WETH_GATEWAY_PROXY_ADDR = vm.envAddress("L1_WETH_GATEWAY_PROXY_ADDR");
    address L1_ERC721_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ERC721_GATEWAY_PROXY_ADDR");
    address L1_ERC1155_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ERC1155_GATEWAY_PROXY_ADDR");
    address L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR = vm.envAddress("L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR");
    address L1_ENFORCED_TX_GATEWAY_PROXY_ADDR = vm.envAddress("L1_ENFORCED_TX_GATEWAY_PROXY_ADDR");
    address L1_WHITELIST_ADDR = vm.envAddress("L1_WHITELIST_ADDR");

    ScrollOwner owner;

    function run() external {
        vm.startBroadcast(L1_DEPLOYER_PRIVATE_KEY);

        owner = ScrollOwner(payable(L1_SCROLL_OWNER_ADDR));

        // @note we don't config 14D access, since the default admin is a 14D timelock which can access all methods.
        configProxyAdmin();
        configScrollChain();
        configL1MessageQueue();
        configL1ScrollMessenger();
        configL2GasPriceOracle();
        configL1Whitelist();
        configMultipleVersionRollupVerifier();
        configL1GatewayRouter();
        configL1CustomERC20Gateway();
        configL1ERC721Gateway();
        configL1ERC1155Gateway();

        configL1USDCGateway();
        configEnforcedTxGateway();

        grantRoles();
        transferOwnership();

        vm.stopBroadcast();
    }

    function transferOwnership() internal {
        Ownable(L1_PROXY_ADMIN_ADDR).transferOwnership(address(owner));
        Ownable(L1_SCROLL_CHAIN_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L1_MESSAGE_QUEUE_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L1_SCROLL_MESSENGER_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L1_ENFORCED_TX_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L2_GAS_PRICE_ORACLE_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L1_WHITELIST_ADDR).transferOwnership(address(owner));
        Ownable(L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR).transferOwnership(address(owner));
        Ownable(L1_GATEWAY_ROUTER_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L1_DAI_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L1_LIDO_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L1_ETH_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L1_STANDARD_ERC20_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L1_USDC_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L1_WETH_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L1_ERC721_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L1_ERC1155_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
    }

    function grantRoles() internal {
        owner.grantRole(SECURITY_COUNCIL_NO_DELAY_ROLE, SECURITY_COUNCIL_ADDR);
        owner.grantRole(SCROLL_MULTISIG_NO_DELAY_ROLE, SCROLL_MULTISIG_ADDR);
        owner.grantRole(EMERGENCY_MULTISIG_NO_DELAY_ROLE, EMERGENCY_MULTISIG_ADDR);
        owner.grantRole(TIMELOCK_1DAY_DELAY_ROLE, L1_1D_TIMELOCK_ADDR);
        owner.grantRole(TIMELOCK_7DAY_DELAY_ROLE, L1_7D_TIMELOCK_ADDR);

        owner.grantRole(owner.DEFAULT_ADMIN_ROLE(), L1_14D_TIMELOCK_ADDR);
        owner.revokeRole(owner.DEFAULT_ADMIN_ROLE(), vm.addr(L1_DEPLOYER_PRIVATE_KEY));
    }

    function configProxyAdmin() internal {
        bytes4[] memory _selectors;

        // no delay, security council
        _selectors = new bytes4[](2);
        _selectors[0] = ProxyAdmin.upgrade.selector;
        _selectors[1] = ProxyAdmin.upgradeAndCall.selector;
        owner.updateAccess(L1_PROXY_ADMIN_ADDR, _selectors, SECURITY_COUNCIL_NO_DELAY_ROLE, true);
    }

    function configScrollChain() internal {
        bytes4[] memory _selectors;

        // no delay, scroll multisig and emergency multisig
        _selectors = new bytes4[](4);
        _selectors[0] = ScrollChain.revertBatch.selector;
        _selectors[1] = ScrollChain.removeSequencer.selector;
        _selectors[2] = ScrollChain.removeProver.selector;
        _selectors[3] = ScrollChain.setPause.selector;
        owner.updateAccess(L1_SCROLL_CHAIN_PROXY_ADDR, _selectors, SCROLL_MULTISIG_NO_DELAY_ROLE, true);
        owner.updateAccess(L1_SCROLL_CHAIN_PROXY_ADDR, _selectors, EMERGENCY_MULTISIG_NO_DELAY_ROLE, true);

        // delay 1 day, scroll multisig
        _selectors = new bytes4[](2);
        _selectors[0] = ScrollChain.addSequencer.selector;
        _selectors[1] = ScrollChain.addProver.selector;
        owner.updateAccess(L1_SCROLL_CHAIN_PROXY_ADDR, _selectors, TIMELOCK_1DAY_DELAY_ROLE, true);

        // delay 7 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = ScrollChain.updateMaxNumTxInChunk.selector;
        owner.updateAccess(L1_SCROLL_CHAIN_PROXY_ADDR, _selectors, TIMELOCK_7DAY_DELAY_ROLE, true);
    }

    function configL1MessageQueue() internal {
        bytes4[] memory _selectors;

        // delay 1 day, scroll multisig
        _selectors = new bytes4[](2);
        _selectors[0] = L1MessageQueue.updateGasOracle.selector;
        _selectors[1] = L1MessageQueue.updateMaxGasLimit.selector;
        owner.updateAccess(L1_MESSAGE_QUEUE_PROXY_ADDR, _selectors, TIMELOCK_1DAY_DELAY_ROLE, true);
    }

    function configL1ScrollMessenger() internal {
        bytes4[] memory _selectors;

        // no delay, scroll multisig and emergency multisig
        _selectors = new bytes4[](1);
        _selectors[0] = ScrollMessengerBase.setPause.selector;
        owner.updateAccess(L1_SCROLL_MESSENGER_PROXY_ADDR, _selectors, SCROLL_MULTISIG_NO_DELAY_ROLE, true);
        owner.updateAccess(L1_SCROLL_MESSENGER_PROXY_ADDR, _selectors, EMERGENCY_MULTISIG_NO_DELAY_ROLE, true);

        // delay 1 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = L1ScrollMessenger.updateMaxReplayTimes.selector;
        owner.updateAccess(L1_SCROLL_MESSENGER_PROXY_ADDR, _selectors, TIMELOCK_1DAY_DELAY_ROLE, true);
    }

    function configL2GasPriceOracle() internal {
        bytes4[] memory _selectors;

        // no delay, scroll multisig and emergency multisig
        _selectors = new bytes4[](1);
        _selectors[0] = L2GasPriceOracle.setIntrinsicParams.selector;
        owner.updateAccess(L2_GAS_PRICE_ORACLE_PROXY_ADDR, _selectors, SCROLL_MULTISIG_NO_DELAY_ROLE, true);
        owner.updateAccess(L2_GAS_PRICE_ORACLE_PROXY_ADDR, _selectors, EMERGENCY_MULTISIG_NO_DELAY_ROLE, true);
    }

    function configL1Whitelist() internal {
        bytes4[] memory _selectors;

        // delay 1 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = Whitelist.updateWhitelistStatus.selector;
        owner.updateAccess(L1_WHITELIST_ADDR, _selectors, TIMELOCK_1DAY_DELAY_ROLE, true);
    }

    function configMultipleVersionRollupVerifier() internal {
        bytes4[] memory _selectors;

        // no delay, security council
        _selectors = new bytes4[](1);
        _selectors[0] = MultipleVersionRollupVerifier.updateVerifier.selector;
        owner.updateAccess(L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR, _selectors, SECURITY_COUNCIL_NO_DELAY_ROLE, true);

        // delay 7 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = MultipleVersionRollupVerifier.updateVerifier.selector;
        owner.updateAccess(L1_MULTIPLE_VERSION_ROLLUP_VERIFIER_ADDR, _selectors, TIMELOCK_7DAY_DELAY_ROLE, true);
    }

    function configL1GatewayRouter() internal {
        bytes4[] memory _selectors;

        // delay 1 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = L1GatewayRouter.setERC20Gateway.selector;
        owner.updateAccess(L1_GATEWAY_ROUTER_PROXY_ADDR, _selectors, TIMELOCK_1DAY_DELAY_ROLE, true);
    }

    function configL1CustomERC20Gateway() internal {
        bytes4[] memory _selectors;

        // delay 1 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = L1CustomERC20Gateway.updateTokenMapping.selector;
        owner.updateAccess(L1_CUSTOM_ERC20_GATEWAY_PROXY_ADDR, _selectors, TIMELOCK_1DAY_DELAY_ROLE, true);
    }

    function configL1ERC721Gateway() internal {
        bytes4[] memory _selectors;

        // delay 1 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = L1ERC721Gateway.updateTokenMapping.selector;
        owner.updateAccess(L1_ERC721_GATEWAY_PROXY_ADDR, _selectors, TIMELOCK_1DAY_DELAY_ROLE, true);
    }

    function configL1ERC1155Gateway() internal {
        bytes4[] memory _selectors;

        // delay 1 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = L1ERC1155Gateway.updateTokenMapping.selector;
        owner.updateAccess(L1_ERC1155_GATEWAY_PROXY_ADDR, _selectors, TIMELOCK_1DAY_DELAY_ROLE, true);
    }

    function configL1USDCGateway() internal {
        bytes4[] memory _selectors;

        // delay 7 day, scroll multisig
        _selectors = new bytes4[](3);
        _selectors[0] = L1USDCGateway.updateCircleCaller.selector;
        _selectors[1] = L1USDCGateway.pauseDeposit.selector;
        _selectors[2] = L1USDCGateway.pauseWithdraw.selector;
        owner.updateAccess(L1_USDC_GATEWAY_PROXY_ADDR, _selectors, TIMELOCK_7DAY_DELAY_ROLE, true);
    }

    function configEnforcedTxGateway() internal {
        bytes4[] memory _selectors;

        // no delay, scroll multisig and emergency multisig
        _selectors = new bytes4[](1);
        _selectors[0] = EnforcedTxGateway.setPause.selector;
        owner.updateAccess(L1_ENFORCED_TX_GATEWAY_PROXY_ADDR, _selectors, SCROLL_MULTISIG_NO_DELAY_ROLE, true);
        owner.updateAccess(L1_ENFORCED_TX_GATEWAY_PROXY_ADDR, _selectors, EMERGENCY_MULTISIG_NO_DELAY_ROLE, true);
    }
}
