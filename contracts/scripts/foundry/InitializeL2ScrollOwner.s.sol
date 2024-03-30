// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.24;

import {Script} from "forge-std/Script.sol";

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {AccessControl} from "@openzeppelin/contracts/access/AccessControl.sol";
import {AccessControlEnumerable} from "@openzeppelin/contracts/access/AccessControlEnumerable.sol";
import {ProxyAdmin} from "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";

import {L2USDCGateway} from "../../src/L2/gateways/usdc/L2USDCGateway.sol";
import {L2CustomERC20Gateway} from "../../src/L2/gateways/L2CustomERC20Gateway.sol";
import {L2CustomERC20Gateway} from "../../src/L2/gateways/L2CustomERC20Gateway.sol";
import {L2ERC1155Gateway} from "../../src/L2/gateways/L2ERC1155Gateway.sol";
import {L2ERC721Gateway} from "../../src/L2/gateways/L2ERC721Gateway.sol";
import {L2GatewayRouter} from "../../src/L2/gateways/L2GatewayRouter.sol";
import {ScrollMessengerBase} from "../../src/libraries/ScrollMessengerBase.sol";
import {L1GasPriceOracle} from "../../src/L2/predeploys/L1GasPriceOracle.sol";
import {L2TxFeeVault} from "../../src/L2/predeploys/L2TxFeeVault.sol";
import {Whitelist} from "../../src/L2/predeploys/Whitelist.sol";
import {ScrollOwner} from "../../src/misc/ScrollOwner.sol";

// solhint-disable max-states-count
// solhint-disable state-visibility
// solhint-disable var-name-mixedcase

contract InitializeL2ScrollOwner is Script {
    uint256 L2_DEPLOYER_PRIVATE_KEY = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");

    bytes32 constant SECURITY_COUNCIL_NO_DELAY_ROLE = keccak256("SECURITY_COUNCIL_NO_DELAY_ROLE");
    bytes32 constant SCROLL_MULTISIG_NO_DELAY_ROLE = keccak256("SCROLL_MULTISIG_NO_DELAY_ROLE");
    bytes32 constant EMERGENCY_MULTISIG_NO_DELAY_ROLE = keccak256("EMERGENCY_MULTISIG_NO_DELAY_ROLE");

    bytes32 constant TIMELOCK_1DAY_DELAY_ROLE = keccak256("TIMELOCK_1DAY_DELAY_ROLE");
    bytes32 constant TIMELOCK_7DAY_DELAY_ROLE = keccak256("TIMELOCK_7DAY_DELAY_ROLE");

    address SCROLL_MULTISIG_ADDR = vm.envAddress("L2_SCROLL_MULTISIG_ADDR");
    address SECURITY_COUNCIL_ADDR = vm.envAddress("L2_SECURITY_COUNCIL_ADDR");
    address EMERGENCY_MULTISIG_ADDR = vm.envAddress("L2_EMERGENCY_MULTISIG_ADDR");

    address L2_SCROLL_OWNER_ADDR = vm.envAddress("L2_SCROLL_OWNER_ADDR");
    address L2_1D_TIMELOCK_ADDR = vm.envAddress("L2_1D_TIMELOCK_ADDR");
    address L2_7D_TIMELOCK_ADDR = vm.envAddress("L2_7D_TIMELOCK_ADDR");
    address L2_14D_TIMELOCK_ADDR = vm.envAddress("L2_14D_TIMELOCK_ADDR");

    address L2_PROXY_ADMIN_ADDR = vm.envAddress("L2_PROXY_ADMIN_ADDR");
    address L2_TX_FEE_VAULT_ADDR = vm.envAddress("L2_TX_FEE_VAULT_ADDR");
    address L1_GAS_PRICE_ORACLE_ADDR = vm.envAddress("L1_GAS_PRICE_ORACLE_ADDR");
    address L2_WHITELIST_ADDR = vm.envAddress("L2_WHITELIST_ADDR");
    address L2_MESSAGE_QUEUE_ADDR = vm.envAddress("L2_MESSAGE_QUEUE_ADDR");

    address L2_SCROLL_MESSENGER_PROXY_ADDR = vm.envAddress("L2_SCROLL_MESSENGER_PROXY_ADDR");
    address L2_GATEWAY_ROUTER_PROXY_ADDR = vm.envAddress("L2_GATEWAY_ROUTER_PROXY_ADDR");
    address L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR");
    address L2_DAI_GATEWAY_PROXY_ADDR = vm.envAddress("L2_DAI_GATEWAY_PROXY_ADDR");
    address L2_LIDO_GATEWAY_PROXY_ADDR = vm.envAddress("L2_LIDO_GATEWAY_PROXY_ADDR");
    address L2_ETH_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ETH_GATEWAY_PROXY_ADDR");
    address L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR = vm.envAddress("L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR");
    address L2_USDC_GATEWAY_PROXY_ADDR = vm.envAddress("L2_USDC_GATEWAY_PROXY_ADDR");
    address L2_WETH_GATEWAY_PROXY_ADDR = vm.envAddress("L2_WETH_GATEWAY_PROXY_ADDR");
    address L2_ERC721_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC721_GATEWAY_PROXY_ADDR");
    address L2_ERC1155_GATEWAY_PROXY_ADDR = vm.envAddress("L2_ERC1155_GATEWAY_PROXY_ADDR");

    address L2_USDC_PROXY_ADDR = vm.envAddress("L2_USDC_PROXY_ADDR");
    address L2_USDC_MASTER_MINTER_ADDR = vm.envAddress("L2_USDC_MASTER_MINTER_ADDR");

    ScrollOwner owner;

    function run() external {
        vm.startBroadcast(L2_DEPLOYER_PRIVATE_KEY);

        owner = ScrollOwner(payable(L2_SCROLL_OWNER_ADDR));

        // @note we don't config 14D access, since the default admin is a 14D timelock which can access all methods.
        configProxyAdmin();
        configL1GasPriceOracle();
        configL2TxFeeVault();
        configL2Whitelist();
        configL2ScrollMessenger();
        configL2GatewayRouter();
        configL2CustomERC20Gateway();
        configL2ERC721Gateway();
        configL2ERC1155Gateway();

        configL2USDCGateway();

        grantRoles();
        transferOwnership();

        vm.stopBroadcast();
    }

    function transferOwnership() internal {
        Ownable(L2_PROXY_ADMIN_ADDR).transferOwnership(address(owner));
        Ownable(L2_MESSAGE_QUEUE_ADDR).transferOwnership(address(owner));
        Ownable(L1_GAS_PRICE_ORACLE_ADDR).transferOwnership(address(owner));
        Ownable(L2_TX_FEE_VAULT_ADDR).transferOwnership(address(owner));
        Ownable(L2_WHITELIST_ADDR).transferOwnership(address(owner));
        Ownable(L2_SCROLL_MESSENGER_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L2_GATEWAY_ROUTER_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L2_DAI_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L2_LIDO_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L2_ETH_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L2_STANDARD_ERC20_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L2_WETH_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L2_ERC721_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L2_ERC1155_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));

        Ownable(L2_USDC_GATEWAY_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L2_USDC_PROXY_ADDR).transferOwnership(address(owner));
        Ownable(L2_USDC_MASTER_MINTER_ADDR).transferOwnership(address(owner));
    }

    function grantRoles() internal {
        owner.grantRole(SECURITY_COUNCIL_NO_DELAY_ROLE, SECURITY_COUNCIL_ADDR);
        owner.grantRole(SCROLL_MULTISIG_NO_DELAY_ROLE, SCROLL_MULTISIG_ADDR);
        owner.grantRole(EMERGENCY_MULTISIG_NO_DELAY_ROLE, EMERGENCY_MULTISIG_ADDR);
        owner.grantRole(TIMELOCK_1DAY_DELAY_ROLE, L2_1D_TIMELOCK_ADDR);
        owner.grantRole(TIMELOCK_7DAY_DELAY_ROLE, L2_7D_TIMELOCK_ADDR);

        owner.grantRole(owner.DEFAULT_ADMIN_ROLE(), L2_14D_TIMELOCK_ADDR);
        owner.revokeRole(owner.DEFAULT_ADMIN_ROLE(), vm.addr(L2_DEPLOYER_PRIVATE_KEY));
    }

    function configProxyAdmin() internal {
        bytes4[] memory _selectors;

        // no delay, security council
        _selectors = new bytes4[](2);
        _selectors[0] = ProxyAdmin.upgrade.selector;
        _selectors[1] = ProxyAdmin.upgradeAndCall.selector;
        owner.updateAccess(L2_PROXY_ADMIN_ADDR, _selectors, SECURITY_COUNCIL_NO_DELAY_ROLE, true);
    }

    function configL1GasPriceOracle() internal {
        bytes4[] memory _selectors;

        // no delay, scroll multisig and emergency multisig
        _selectors = new bytes4[](2);
        _selectors[0] = L1GasPriceOracle.setOverhead.selector;
        _selectors[1] = L1GasPriceOracle.setScalar.selector;
        owner.updateAccess(L1_GAS_PRICE_ORACLE_ADDR, _selectors, SCROLL_MULTISIG_NO_DELAY_ROLE, true);
        owner.updateAccess(L1_GAS_PRICE_ORACLE_ADDR, _selectors, EMERGENCY_MULTISIG_NO_DELAY_ROLE, true);
    }

    function configL2TxFeeVault() internal {
        bytes4[] memory _selectors;

        // no delay, scroll multisig and emergency multisig
        _selectors = new bytes4[](1);
        _selectors[0] = L2TxFeeVault.updateMinWithdrawAmount.selector;
        owner.updateAccess(L2_TX_FEE_VAULT_ADDR, _selectors, SCROLL_MULTISIG_NO_DELAY_ROLE, true);
        owner.updateAccess(L2_TX_FEE_VAULT_ADDR, _selectors, EMERGENCY_MULTISIG_NO_DELAY_ROLE, true);
    }

    function configL2Whitelist() internal {
        bytes4[] memory _selectors;

        // delay 1 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = Whitelist.updateWhitelistStatus.selector;
        owner.updateAccess(L2_WHITELIST_ADDR, _selectors, TIMELOCK_1DAY_DELAY_ROLE, true);
    }

    function configL2ScrollMessenger() internal {
        bytes4[] memory _selectors;

        // no delay, scroll multisig and emergency multisig
        _selectors = new bytes4[](1);
        _selectors[0] = ScrollMessengerBase.setPause.selector;
        owner.updateAccess(L2_SCROLL_MESSENGER_PROXY_ADDR, _selectors, SCROLL_MULTISIG_NO_DELAY_ROLE, true);
        owner.updateAccess(L2_SCROLL_MESSENGER_PROXY_ADDR, _selectors, EMERGENCY_MULTISIG_NO_DELAY_ROLE, true);
    }

    function configL2GatewayRouter() internal {
        bytes4[] memory _selectors;

        // delay 1 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = L2GatewayRouter.setERC20Gateway.selector;
        owner.updateAccess(L2_GATEWAY_ROUTER_PROXY_ADDR, _selectors, TIMELOCK_1DAY_DELAY_ROLE, true);
    }

    function configL2CustomERC20Gateway() internal {
        bytes4[] memory _selectors;

        // delay 1 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = L2CustomERC20Gateway.updateTokenMapping.selector;
        owner.updateAccess(L2_CUSTOM_ERC20_GATEWAY_PROXY_ADDR, _selectors, TIMELOCK_1DAY_DELAY_ROLE, true);
    }

    function configL2ERC721Gateway() internal {
        bytes4[] memory _selectors;

        // delay 1 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = L2ERC721Gateway.updateTokenMapping.selector;
        owner.updateAccess(L2_ERC721_GATEWAY_PROXY_ADDR, _selectors, TIMELOCK_1DAY_DELAY_ROLE, true);
    }

    function configL2ERC1155Gateway() internal {
        bytes4[] memory _selectors;

        // delay 1 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = L2ERC1155Gateway.updateTokenMapping.selector;
        owner.updateAccess(L2_ERC1155_GATEWAY_PROXY_ADDR, _selectors, TIMELOCK_1DAY_DELAY_ROLE, true);
    }

    function configL2USDCGateway() internal {
        bytes4[] memory _selectors;

        // delay 7 day, scroll multisig
        _selectors = new bytes4[](3);
        _selectors[0] = L2USDCGateway.updateCircleCaller.selector;
        _selectors[1] = L2USDCGateway.pauseDeposit.selector;
        _selectors[2] = L2USDCGateway.pauseWithdraw.selector;
        owner.updateAccess(L2_USDC_GATEWAY_PROXY_ADDR, _selectors, TIMELOCK_7DAY_DELAY_ROLE, true);

        // delay 7 day, scroll multisig
        _selectors = new bytes4[](1);
        _selectors[0] = Ownable.transferOwnership.selector;
        owner.updateAccess(L2_USDC_PROXY_ADDR, _selectors, TIMELOCK_7DAY_DELAY_ROLE, true);
        owner.updateAccess(L2_USDC_MASTER_MINTER_ADDR, _selectors, TIMELOCK_7DAY_DELAY_ROLE, true);
    }
}
