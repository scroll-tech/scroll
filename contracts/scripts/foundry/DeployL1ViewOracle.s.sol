// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {L1ViewOracle} from "../../src/L1/L1ViewOracle.sol";

contract DeployL1ViewOracle is Script {
    uint256 L1_DEPLOYER_PRIVATE_KEY = vm.envUint("L1_DEPLOYER_PRIVATE_KEY");

    function run() external {
        vm.startBroadcast(L1_DEPLOYER_PRIVATE_KEY);

        L1ViewOracle l1ViewOracle = new L1ViewOracle();
        logAddress("L1_VIEW_ORACLE_ADDR", address(l1ViewOracle));

        vm.stopBroadcast();
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
