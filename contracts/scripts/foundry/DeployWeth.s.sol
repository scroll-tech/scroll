// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.24;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {WrappedEther} from "../../src/L2/predeploys/WrappedEther.sol";

contract DeployWeth is Script {
    address L1_WETH_ADDR = vm.envAddress("L1_WETH_ADDR");
    address L2_WETH_ADDR = vm.envAddress("L2_WETH_ADDR");

    function run() external {
        // deploy weth only if we're running a private L1 network
        if (L1_WETH_ADDR == address(0)) {
            uint256 L1_WETH_DEPLOYER_PRIVATE_KEY = vm.envUint("L1_WETH_DEPLOYER_PRIVATE_KEY");
            vm.startBroadcast(L1_WETH_DEPLOYER_PRIVATE_KEY);
            WrappedEther weth = new WrappedEther();
            L1_WETH_ADDR = address(weth);
            vm.stopBroadcast();
        }

        logAddress("L1_WETH_ADDR", L1_WETH_ADDR);
        logAddress("L2_WETH_ADDR", L2_WETH_ADDR);
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
