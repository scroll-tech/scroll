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
import {L1CustomERC20Gateway} from "../../src/L1/gateways/L1CustomERC20Gateway.sol";

/**
 * @title DeployScrollPuffETH
 * @author Puffer Finance
 * @notice Deploys PufETH on Scroll
 * @dev
 *
 *
 *         NOTE:
 *
 *         If you ran the deployment script, but did not `--broadcast` the transaction, it will still update your local chainId-deployment.json file.
 *         Other scripts will fail because addresses will be updated in deployments file, but the deployment never happened.
 *
 *         BaseScript.sol holds the private key logic, if you don't have `PK` ENV variable, it will use the default one PK from `makeAddr("pufferDeployer")`
 *
 */
contract DeployPufETHScrollL1 is BaseScript {
    // Scroll's contracts
    address constant L1_ROUTER = 0xF8B1378579659D8F7EE5f3C929c2f3E332E41Fd6;
    address constant L1_MESSANGER = 0x6774Bcbd5ceCeF1336b5300fb5186a12DDD8b367;

    // Puffer's contracts
    // L1
    address constant L1_TOKEN = 0xD9A442856C234a39a81a089C06451EBAa4306a72;

    // L2
    address constant L2_TOKEN = 0xc4d46E8402F476F269c379677C99F18E22Ea030e;
    address constant L2_GATEWAY = 0x9eBf2f33526CD571f8b2ad312492cb650870CFd6;

    function step2() internal {
        ProxyAdmin proxyAdmin = new ProxyAdmin();
        TransparentUpgradeableProxy l1GatewayProxy =
            new TransparentUpgradeableProxy(address(new EmptyContract()), address(proxyAdmin), "");
        L1CustomERC20Gateway l1Gatway = new L1CustomERC20Gateway(L2_GATEWAY, L1_ROUTER, L1_MESSANGER);

        console.log("L1 Gateway Proxy", address(l1GatewayProxy));
        console.log("L1 Proxy Admin", address(proxyAdmin));
        console.log("L1 Gateway Impl", address(l1Gatway));

        proxyAdmin.upgradeAndCall(
            ITransparentUpgradeableProxy(address(l1GatewayProxy)),
            address(l1Gatway),
            abi.encodeCall(l1Gatway.initialize, (L2_GATEWAY, L1_ROUTER, L1_MESSANGER))
        );
        proxyAdmin.upgradeAndCall(
            ITransparentUpgradeableProxy(address(l1GatewayProxy)),
            address(l1Gatway),
            abi.encodeCall(l1Gatway.updateTokenMapping, (L1_TOKEN, L2_TOKEN))
        );
    }

    function run() public broadcast {
        // We run this after running step1() on L2 and updating the L2 addresses in this script
        // step2();
    }
}
