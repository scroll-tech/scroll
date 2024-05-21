// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.24;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {TestCurieOpcodes} from "../../src/misc/TestCurieOpcodes.sol";

contract DeployEcc is Script {
    address L2_TEST_CURIE_OPCODES_ADDR = vm.envAddress("L2_TEST_CURIE_OPCODES_ADDR");

    function run() external {
        uint256 L2_DEPLOYER_PRIVATE_KEY = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");
        vm.startBroadcast(L2_DEPLOYER_PRIVATE_KEY);
        Ecc ecc = new Ecc();
        L2_TEST_CURIE_OPCODES_ADDR = address(ecc);
        vm.stopBroadcast();

        logAddress("L2_TEST_CURIE_OPCODES_ADDR", L2_TEST_CURIE_OPCODES_ADDR);
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
