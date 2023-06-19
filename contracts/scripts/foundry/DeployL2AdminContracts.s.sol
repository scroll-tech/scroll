// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.10;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

import {Safe} from "safe-contracts/Safe.sol";
import {SafeProxy} from "safe-contracts/proxies/SafeProxy.sol";
import {TimelockController} from "@openzeppelin/contracts/governance/TimelockController.sol";
import {Forwarder} from "../../src/misc/Forwarder.sol";
import {MockTarget} from "../../src/mocks/MockTarget.sol";

interface ISafe {
    function setup(
        address[] calldata _owners,
        uint256 _threshold,
        address to,
        bytes calldata data,
        address fallbackHandler,
        address paymentToken,
        uint256 payment,
        address payable paymentReceiver
    ) external;
}

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
        // TODO: get timelock delay from env. for now just use 0
        address scroll_timelock = deployTimelockController(scroll_safe, 0);

        logAddress("L2_SCROLL_SAFE_ADDR", address(scroll_safe));
        logAddress("L2_SCROLL_TIMELOCK_ADDR", address(scroll_timelock));

        address forwarder = deployForwarder(address(council_timelock), address(scroll_timelock));
        logAddress("L1_FORWARDER_ADDR", address(forwarder));

        MockTarget target = new MockTarget();
        logAddress("L2_TARGET_ADDR", address(target));

        vm.stopBroadcast();
    }

    function deployForwarder(address admin, address superAdmin) internal returns (address) {
        Forwarder forwarder = new Forwarder(admin, superAdmin);
        return address(forwarder);
    }

    function deploySafe() internal returns (address) {
        address owner = vm.addr(L2_DEPLOYER_PRIVATE_KEY);
        // TODO: get safe signers from env

        Safe safe = new Safe();
        SafeProxy proxy = new SafeProxy(address(safe));
        address[] memory owners = new address[](1);
        owners[0] = owner;
        // deployer 1/1. no gas refunds for now
        ISafe(address(proxy)).setup(
            owners,
            1,
            address(0),
            new bytes(0),
            address(0),
            address(0),
            0,
            payable(address(0))
        );

        return address(proxy);
    }

    function deployTimelockController(address safe, uint256 delay) internal returns (address) {
        address deployer = vm.addr(L2_DEPLOYER_PRIVATE_KEY);

        address[] memory proposers = new address[](1);
        proposers[0] = safe;

        address[] memory executors = new address[](1);
        executors[0] = address(0);
        // add SAFE as the only proposer, anyone can execute
        TimelockController timelock = new TimelockController(delay, proposers, executors);

        bytes32 TIMELOCK_ADMIN_ROLE = keccak256("TIMELOCK_ADMIN_ROLE");

        // make safe admin of timelock, then revoke deployer's rights
        timelock.grantRole(TIMELOCK_ADMIN_ROLE, address(safe));
        timelock.revokeRole(TIMELOCK_ADMIN_ROLE, deployer);

        return address(timelock);
    }

    function logBytes32(string memory name, bytes32 value) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(bytes32(value)))));
    }

    function logUint(string memory name, uint256 value) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(uint256(value)))));
    }

    function logAddress(string memory name, address addr) internal view {
        console.log(string(abi.encodePacked(name, "=", vm.toString(address(addr)))));
    }
}
