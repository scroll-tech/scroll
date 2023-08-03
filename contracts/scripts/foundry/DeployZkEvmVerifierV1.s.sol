// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {ZkEvmVerifierV1} from "../../src/libraries/verifier/ZkEvmVerifierV1.sol";

contract DeployWeth is Script {
    uint256 L1_DEPLOYER_PRIVATE_KEY = vm.envUint("L1_DEPLOYER_PRIVATE_KEY");

    address L1_PLONK_VERIFIER_ADDR = vm.envAddress("L1_PLONK_VERIFIER_ADDR");

    function run() external {
        vm.startBroadcast(L1_DEPLOYER_PRIVATE_KEY);

        deployZkEvmVerifierV1();

        vm.stopBroadcast();
    }

    function deployZkEvmVerifierV1() internal {
        ZkEvmVerifierV1 verifier = new ZkEvmVerifierV1(L1_PLONK_VERIFIER_ADDR);

        logAddress("L1_ZKEVM_VERIFIER_V1_ADDR", address(verifier));
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
