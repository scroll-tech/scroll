// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {Safe} from "safe-contracts/Safe.sol";
import {TimelockController} from "@openzeppelin/contracts/governance/TimelockController.sol";

contract DeployL2AdminContracts is Script {
    uint256 L2_DEPLOYER_PRIVATE_KEY = vm.envUint("L2_DEPLOYER_PRIVATE_KEY");

    function run() external {
        vm.startBroadcast(L2_DEPLOYER_PRIVATE_KEY);

        address council_safe = deploySafe();
        // deploy timelock with no delay, just to keep council and scroll admin flows be parallel
        address council_timelock = deployTimelockController(council_safe, 0);
        
        logAddress("L2_COUNCIL_SAFE_ADDR", address(council_safe));
        logAddress("L2_COUNCIL_TIMELOCK_ADDR", address(council_timelock));

        address scroll_safe = deploySafe();
        // TODO: get timelock delay from env. for now just use 2 days
        address scroll_timelock = deployTimelockController(scroll_safe, 2 days);
        
        logAddress("L2_SCROLL_SAFE_ADDR", address(scroll_safe));
        logAddress("L2_SCROLL_TIMELOCK_ADDR", address(scroll_timelock));

        vm.stopBroadcast();
    }

    function deploySafe() internal returns (address) {
        address owner = vm.addr(L2_DEPLOYER_PRIVATE_KEY);
        // TODO: get safe signers from env

        Safe safe = new Safe();
        address[] memory owners = new address[](1);
        owners[0] = owner;
        // deployer 1/1. no gas refunds for now
        safe.setup(
            owners,
            1,
            address(0),
            new bytes(0),
            address(0),
            address(0),
            0,
            payable(address(0))
        );
        return address(safe);
    }

    function deployTimelockController(address safe, uint delay) internal returns(address) {
        address deployer = vm.addr(L2_DEPLOYER_PRIVATE_KEY);

        address[] memory proposers = new address[](1);
        proposers[0] = safe;
        // add SAFE as the only proposer, anyone can execute
        TimelockController timelock = new TimelockController(delay, proposers, new address[](0));

        bytes32 TIMELOCK_ADMIN_ROLE = keccak256("TIMELOCK_ADMIN_ROLE");

        // make safe admin of timelock, then revoke deployer's rights
        timelock.grantRole(TIMELOCK_ADMIN_ROLE, address(safe));
        timelock.revokeRole(TIMELOCK_ADMIN_ROLE, deployer);

        return address(timelock);
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
