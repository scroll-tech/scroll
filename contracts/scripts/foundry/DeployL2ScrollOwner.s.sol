// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {TimelockController} from "@openzeppelin/contracts/governance/TimelockController.sol";

import {ScrollOwner} from "../../src/misc/ScrollOwner.sol";

// solhint-disable state-visibility
// solhint-disable var-name-mixedcase

contract DeployL2ScrollOwner is Script {
    uint256 L2_DEPLOYER_PRIVATE_KEY = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");

    address SCROLL_MULTISIG_ADDR = vm.envAddress("L2_SCROLL_MULTISIG_ADDR");

    address SECURITY_COUNCIL_ADDR = vm.envAddress("L2_SECURITY_COUNCIL_ADDR");

    address L2_PROPOSAL_EXECUTOR_ADDR = vm.envAddress("L2_PROPOSAL_EXECUTOR_ADDR");

    function run() external {
        vm.startBroadcast(L2_DEPLOYER_PRIVATE_KEY);

        deployScrollOwner();

        deployTimelockController(1);
        deployTimelockController(7);
        deployTimelockController(14);

        vm.stopBroadcast();
    }

    function deployScrollOwner() internal {
        ScrollOwner owner = new ScrollOwner();

        logAddress("L2_SCROLL_OWNER_ADDR", address(owner));
    }

    function deployTimelockController(uint256 delayInDay) internal {
        address[] memory proposers = new address[](1);
        address[] memory executors = new address[](1);

        proposers[0] = SCROLL_MULTISIG_ADDR;
        executors[0] = L2_PROPOSAL_EXECUTOR_ADDR;

        TimelockController timelock = new TimelockController(
            delayInDay * 1 days,
            proposers,
            executors,
            SECURITY_COUNCIL_ADDR
        );

        logAddress(string(abi.encodePacked("L2_", vm.toString(delayInDay), "D_TIMELOCK_ADDR")), address(timelock));
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
