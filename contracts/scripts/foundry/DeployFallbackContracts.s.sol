// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {Fallback} from "../../src/misc/Fallback.sol";

contract DeployFallbackContracts is Script {
    uint256 DEPLOYER_PRIVATE_KEY = vm.envUint("DEPLOYER_PRIVATE_KEY");
    uint256 NUM_CONTRACTS = vm.envUint("NUM_CONTRACTS");

    function run() external {
        vm.startBroadcast(DEPLOYER_PRIVATE_KEY);

        for (uint256 ii = 0; ii < NUM_CONTRACTS; ++ii) {
            Fallback fallbackContract = new Fallback();
            logAddress("FALLBACK", address(fallbackContract));
        }

        vm.stopBroadcast();
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
