// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.8.0 <0.9.0;

import {stdJson} from "forge-std/StdJson.sol";
import "forge-std/console.sol";

import {BaseScript, EmptyContract} from ".//BaseScript.s.sol";

import {ProxyAdmin} from "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import {
    TransparentUpgradeableProxy,
    ITransparentUpgradeableProxy
} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import {ScrollStandardERC20} from "../../src/libraries/token/ScrollStandardERC20.sol";
import {L2CustomERC20Gateway} from "../../src/L2/gateways/L2CustomERC20Gateway.sol";

/**
 * @title DeployScrollPuffETH
 * @author Puffer Finance
 * @notice Deploys PufETH on Scroll
 * @dev
 *
 *         NOTE:
 *
 *         If you ran the deployment script, but did not `--broadcast` the transaction, it will still update your local chainId-deployment.json file.
 *         Other scripts will fail because addresses will be updated in deployments file, but the deployment never happened.
 *
 *         BaseScript.sol holds the private key logic, if you don't have `PK` ENV variable, it will use the default one PK from `makeAddr("pufferDeployer")`
 *
 */
contract DeployPufETHScrollL2 is BaseScript {
    // Scroll's contracts
    address constant L2_ROUTER = 0x4C0926FF5252A435FD19e10ED15e5a249Ba19d79;
    address constant L2_MESSANGER = 0x781e90f1c8Fc4611c9b7497C3B47F99Ef6969CbC;

    // Puffer's contracts
    // L1
    address constant L1_GATEWAY = 0xA033Ff09f2da45f0e9ae495f525363722Df42b2a;
    address constant L1_TOKEN = 0xD9A442856C234a39a81a089C06451EBAa4306a72;

    // L2
    ProxyAdmin PROXY_ADMIN = ProxyAdmin(0xea0f8ae7155466C119340618Cd28bFD04E7309B8);
    address constant L2_GATEWAY_PROXY = 0x9eBf2f33526CD571f8b2ad312492cb650870CFd6;
    address constant L2_TOKEN = 0xc4d46E8402F476F269c379677C99F18E22Ea030e;

    function step1() internal {
        ProxyAdmin proxyAdmin = new ProxyAdmin();
        address emptyContract = address(new EmptyContract());
        TransparentUpgradeableProxy l2GatewayProxy =
            new TransparentUpgradeableProxy(emptyContract, address(proxyAdmin), "");
        TransparentUpgradeableProxy tokenProxy = new TransparentUpgradeableProxy(emptyContract, address(proxyAdmin), "");

        ScrollStandardERC20 newImplementation = new ScrollStandardERC20();

        proxyAdmin.upgradeAndCall(
            ITransparentUpgradeableProxy(address(tokenProxy)),
            address(newImplementation),
            abi.encodeCall(
                newImplementation.initialize, ("PufferVault", "PufETH", 18, address(l2GatewayProxy), L1_TOKEN)
            )
        );

        console.log("L2 emptyContract", emptyContract);
        console.log("L2 Gateway Proxy", address(l2GatewayProxy));
        console.log("L2 Proxy Admin", address(proxyAdmin));
        console.log("L2 Token Proxy", address(tokenProxy));
        console.log("L2 Token impl", address(newImplementation));
    }

    function step3() internal {
        L2CustomERC20Gateway l2Gatway = new L2CustomERC20Gateway(L1_GATEWAY, L2_ROUTER, L2_MESSANGER);
        PROXY_ADMIN.upgradeAndCall(
            ITransparentUpgradeableProxy(L2_GATEWAY_PROXY),
            address(l2Gatway),
            abi.encodeCall(l2Gatway.initialize, (L1_GATEWAY, L2_ROUTER, L2_MESSANGER))
        );
        PROXY_ADMIN.upgradeAndCall(
            ITransparentUpgradeableProxy(L2_GATEWAY_PROXY),
            address(l2Gatway),
            abi.encodeCall(l2Gatway.updateTokenMapping, (L2_TOKEN, L1_TOKEN))
        );
    }

    function run() public broadcast {
        // We run this first then go and update the L1 deployment script with the logged addresses.
        //step1();

        // We run this after deploying L1 contracts and updating this script with L1 contract addresses.
        //step3();
    }
}
