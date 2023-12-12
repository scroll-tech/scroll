// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {L1Blocks} from "../../src/L2/L1Blocks.sol";

contract DeployL2L1BlocksContract is Script {
    uint256 L2_DEPLOYER_PRIVATE_KEY = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");
    uint64 L1_BLOCKS_FIRST_APPLIED = uint64(vm.envUint("L1_BLOCKS_FIRST_APPLIED"));

    function run() external {
        vm.startBroadcast(L2_DEPLOYER_PRIVATE_KEY);

        L1Blocks l1Blocks = new L1Blocks(L1_BLOCKS_FIRST_APPLIED);
        logAddress("L2_L1BLOCKS_ADDR", address(l1Blocks));

        vm.stopBroadcast();
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
