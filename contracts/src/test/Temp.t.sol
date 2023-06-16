// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";
import "forge-std/Vm.sol";
// import {Vm, VmSafe} from "./Vm.sol";
import "forge-std/Test.sol";
import "forge-std/console.sol";

import {Safe} from "safe-contracts/Safe.sol";
import {SafeProxy} from "safe-contracts/proxies/SafeProxy.sol";
import {TimelockController} from "@openzeppelin/contracts/governance/TimelockController.sol";
import {Forwarder} from "../../src/misc/Forwarder.sol";
import {MockTarget} from "../../src/mocks/MockTarget.sol";

interface ISafe {
    // enum
    enum Operation {
        Call,
        DelegateCall
    }

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

    function execTransaction(
        address to,
        uint256 value,
        bytes calldata data,
        Operation operation,
        uint256 safeTxGas,
        uint256 baseGas,
        uint256 gasPrice,
        address gasToken,
        address payable refundReceiver,
        bytes memory signatures
    ) external returns (bool success);

    function checkNSignatures(
        bytes32 dataHash,
        bytes memory data,
        bytes memory signatures,
        uint256 requiredSignatures
    ) external;
}

// scratchpad

contract Temp is DSTestPlus {
    address scroll_safe;

    // function setUp() external {
    //     hevm.prank(0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266);

    //     address council_safe = deploySafe();
    //     // deploy timelock with no delay, just to keep council and scroll admin flows be parallel
    //     address council_timelock = deployTimelockController(council_safe, 0);

    //     // logAddress("L2_COUNCIL_SAFE_ADDR", address(council_safe));
    //     // logAddress("L2_COUNCIL_TIMELOCK_ADDR", address(council_timelock));

    //     address scroll_safe = deploySafe();
    //     // TODO: get timelock delay from env. for now just use 0
    //     address scroll_timelock = deployTimelockController(scroll_safe, 0);

    //     // logAddress("L2_SCROLL_SAFE_ADDR", address(scroll_safe));
    //     // logAddress("L2_SCROLL_TIMELOCK_ADDR", address(scroll_timelock));

    //     address forwarder = deployForwarder(address(council_safe), address(scroll_safe));
    //     // logAddress("L1_FORWARDER_ADDR", address(forwarder));

    //     MockTarget target = new MockTarget();
    //     // logAddress("L2_TARGET_ADDR", address(target));

    //     // vm.stopBroadcast();
    // }
    function testEcrecover() external {
        bytes32 dataHash = 0xb453bd4e271eed985cbab8231da609c4ce0a9cf1f763b6c1594e76315510e0f1;
        // (uint8 v, bytes32 r, bytes32 s) = signatureSplit(
        //     hex"078461ca16494711508b8602c1ea3ef515e5bfe11d67fc76e45b9217d42059f57abdde7cb9bf83b094991e2b6e61fd8b1146de575fd12080d65eaedd2e0c74da1c",
        //     0
        // );
        bytes
            memory signatures = hex"078461ca16494711508b8602c1ea3ef515e5bfe11d67fc76e45b9217d42059f57abdde7cb9bf83b094991e2b6e61fd8b1146de575fd12080d65eaedd2e0c74da1c";
        uint256 requiredSignatures = 1;
        uint8 v;
        bytes32 r;
        bytes32 s;
        uint256 i;
        for (i = 0; i < requiredSignatures; i++) {
            (v, r, s) = signatureSplit(signatures, i);
            emit log_uint(v);
            emit log_bytes32(r);
            emit log_bytes32(s);
            address currentOwner = ecrecover(
                keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", dataHash)),
                v,
                r,
                s
            );
            assertEq(address(0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf), currentOwner);
        }
    }

    function testEcrecover1() external {
        bytes
            memory sig = hex"078461ca16494711508b8602c1ea3ef515e5bfe11d67fc76e45b9217d42059f57abdde7cb9bf83b094991e2b6e61fd8b1146de575fd12080d65eaedd2e0c74da1c";
        uint8 v;
        bytes32 r;
        bytes32 s;
        (v, r, s) = signatureSplit(sig, 0);
        emit log_uint(v);
        emit log_bytes32(r);
        emit log_bytes32(s);

        require(r == 0x078461ca16494711508b8602c1ea3ef515e5bfe11d67fc76e45b9217d42059f5, "r");
        require(s == 0x7abdde7cb9bf83b094991e2b6e61fd8b1146de575fd12080d65eaedd2e0c74da, "s");
        require(v == 28, "v");
    }

    function testSigVerify() external {
        address currentOwner = ecrecover(
            keccak256(
                abi.encodePacked(
                    "\x19Ethereum Signed Message:\n32",
                    bytes32(0xb453bd4e271eed985cbab8231da609c4ce0a9cf1f763b6c1594e76315510e0f1)
                )
            ),
            28,
            0x078461ca16494711508b8602c1ea3ef515e5bfe11d67fc76e45b9217d42059f5,
            0x7abdde7cb9bf83b094991e2b6e61fd8b1146de575fd12080d65eaedd2e0c74da
        );
        require(currentOwner == 0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf, "SIG FAIL ABC");
    }

    function signatureSplit(bytes memory signatures, uint256 pos)
        public
        returns (
            uint8 v,
            bytes32 r,
            bytes32 s
        )
    {
        // solhint-disable-next-line no-inline-assembly
        assembly {
            let signaturePos := mul(0x41, pos)
            r := mload(add(signatures, add(signaturePos, 0x20)))
            s := mload(add(signatures, add(signaturePos, 0x40)))
            /**
             * Here we are loading the last 32 bytes, including 31 bytes
             * of 's'. There is no 'mload8' to do this.
             * 'byte' is not working due to the Solidity parser, so lets
             * use the second best option, 'and'
             */
            v := and(mload(add(signatures, add(signaturePos, 0x41))), 0xff)
        }
    }

    function deployForwarder(address admin, address superAdmin) internal returns (address) {
        Forwarder forwarder = new Forwarder(admin, superAdmin);
        return address(forwarder);
    }

    function deploySafe() internal returns (address) {
        address owner = 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266;
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
        address deployer = 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266;

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
}
