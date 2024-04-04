// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.24;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {Hash} from "../../src/misc/hash.sol";

contract DeployHash is Script {
    function run() external {
        uint256 L2_DEPLOYER_PRIVATE_KEY = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");
        vm.startBroadcast(L2_DEPLOYER_PRIVATE_KEY);
        Hash hash = new Hash();
        address L2_HASH_ADDR = address(hash);
        vm.stopBroadcast();

        logAddress("L2_HASH_ADDR", L2_HASH_ADDR);
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
