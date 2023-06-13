// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {EnforcedTxGateway} from "../L1/gateways/EnforcedTxGateway.sol";
import {L1MessageQueue} from "../L1/rollup/L1MessageQueue.sol";
import {L2GasPriceOracle} from "../L1/rollup/L2GasPriceOracle.sol";
import {IScrollChain, ScrollChain} from "../L1/rollup/ScrollChain.sol";
import {Whitelist} from "../L2/predeploys/Whitelist.sol";
import {IL1ScrollMessenger, L1ScrollMessenger} from "../L1/L1ScrollMessenger.sol";
import {L2ScrollMessenger} from "../L2/L2ScrollMessenger.sol";

contract L1ScrollMessengerTest is DSTestPlus {
    L2ScrollMessenger internal l2Messenger;

    address internal feeVault;

    L1ScrollMessenger internal l1Messenger;
    ScrollChain internal scrollChain;
    L1MessageQueue internal l1MessageQueue;
    L2GasPriceOracle internal gasOracle;
    EnforcedTxGateway internal enforcedTxGateway;
    Whitelist internal whitelist;

    function setUp() public {
        // Deploy L2 contracts
        l2Messenger = new L2ScrollMessenger(address(0), address(0), address(0));

        // Deploy L1 contracts
        scrollChain = new ScrollChain(0);
        l1MessageQueue = new L1MessageQueue();
        l1Messenger = new L1ScrollMessenger();
        gasOracle = new L2GasPriceOracle();
        enforcedTxGateway = new EnforcedTxGateway();
        whitelist = new Whitelist(address(this));

        // Initialize L1 contracts
        l1Messenger.initialize(address(l2Messenger), feeVault, address(scrollChain), address(l1MessageQueue));
        l1MessageQueue.initialize(
            address(l1Messenger),
            address(scrollChain),
            address(enforcedTxGateway),
            address(gasOracle),
            10000000
        );
        gasOracle.initialize(0, 0, 0, 0);
        scrollChain.initialize(address(l1MessageQueue), address(0), 44);

        gasOracle.updateWhitelist(address(whitelist));
        address[] memory _accounts = new address[](1);
        _accounts[0] = address(this);
        whitelist.updateWhitelistStatus(_accounts, true);
    }

    function testForbidCallMessageQueueFromL2() external {
        // import genesis batch
        bytes memory _batchHeader = new bytes(89);
        assembly {
            mstore(add(_batchHeader, add(0x20, 25)), 1)
        }
        scrollChain.importGenesisBatch(
            _batchHeader,
            bytes32(uint256(1)),
            bytes32(0x3152134c22e545ab5d345248502b4f04ef5b45f735f939c7fe6ddc0ffefc9c52)
        );

        IL1ScrollMessenger.L2MessageProof memory proof;
        proof.batchIndex = scrollChain.lastFinalizedBatchIndex();

        hevm.expectRevert("Forbid to call message queue");
        l1Messenger.relayMessageWithProof(address(this), address(l1MessageQueue), 0, 0, new bytes(0), proof);
    }

    function testForbidCallSelfFromL2() external {
        // import genesis batch
        bytes memory _batchHeader = new bytes(89);
        assembly {
            mstore(add(_batchHeader, 57), 1)
        }
        scrollChain.importGenesisBatch(
            _batchHeader,
            bytes32(uint256(1)),
            bytes32(0xf7c03e2b13c88e3fca1410b228b001dd94e3f5ab4b4a4a6981d09a4eb3e5b631)
        );

        IL1ScrollMessenger.L2MessageProof memory proof;
        proof.batchIndex = scrollChain.lastFinalizedBatchIndex();

        hevm.expectRevert("Forbid to call self");
        l1Messenger.relayMessageWithProof(address(this), address(l1Messenger), 0, 0, new bytes(0), proof);
    }

    function testSendMessage(uint256 exceedValue, address refundAddress) external {
        hevm.assume(refundAddress.code.length == 0);
        hevm.assume(uint256(uint160(refundAddress)) > 100); // ignore some precompile contracts

        exceedValue = bound(exceedValue, 1, address(this).balance / 2);

        // Insufficient msg.value
        hevm.expectRevert("Insufficient msg.value");
        l1Messenger.sendMessage(address(0), 1, new bytes(0), 0, refundAddress);

        // refund exceed fee
        uint256 balanceBefore = refundAddress.balance;
        l1Messenger.sendMessage{value: 1 + exceedValue}(address(0), 1, new bytes(0), 0, refundAddress);
        assertEq(balanceBefore + exceedValue, refundAddress.balance);
    }

    function testReplayMessage(uint256 exceedValue, address refundAddress) external {
        hevm.assume(refundAddress.code.length == 0);
        hevm.assume(uint256(uint160(refundAddress)) > 100); // ignore some precompile contracts

        exceedValue = bound(exceedValue, 1, address(this).balance / 2);

        // append a message
        l1Messenger.sendMessage{value: 100}(address(0), 100, new bytes(0), 0, refundAddress);

        // Provided message has not been enqueued
        hevm.expectRevert("Provided message has not been enqueued");
        l1Messenger.replayMessage(address(this), address(0), 101, 0, new bytes(0), 1, refundAddress);

        gasOracle.setL2BaseFee(1);
        // Insufficient msg.value
        hevm.expectRevert("Insufficient msg.value for fee");
        l1Messenger.replayMessage(address(this), address(0), 100, 0, new bytes(0), 1, refundAddress);

        uint256 _fee = gasOracle.l2BaseFee() * 100;

        // Exceed maximum replay times
        hevm.expectRevert("Exceed maximum replay times");
        l1Messenger.replayMessage{value: _fee}(address(this), address(0), 100, 0, new bytes(0), 100, refundAddress);

        l1Messenger.updateMaxReplayTimes(1);

        // refund exceed fee
        uint256 balanceBefore = refundAddress.balance;
        uint256 feeVaultBefore = feeVault.balance;
        l1Messenger.replayMessage{value: _fee + exceedValue}(
            address(this),
            address(0),
            100,
            0,
            new bytes(0),
            100,
            refundAddress
        );
        assertEq(balanceBefore + exceedValue, refundAddress.balance);
        assertEq(feeVaultBefore + _fee, feeVault.balance);
    }

    function testIntrinsicGasLimit() external {
        gasOracle.setIntrinsicParams(21000, 53000, 4, 16);
        uint256 _fee = gasOracle.l2BaseFee() * 24000;
        uint256 value = 1;

        // _xDomainCalldata contains
        //   4B function identifier
        //   20B sender addr
        //   20B target addr
        //   32B value
        //   32B nonce
        //   message byte array (32B offset + 32B length + bytes)
        // So the intrinsic gas must be greater than 22000
        l1Messenger.sendMessage{value: _fee + value}(address(0), value, hex"0011220033", 24000);

        // insufficient intrinsic gas
        hevm.expectRevert("Insufficient gas limit, must be above intrinsic gas");
        l1Messenger.sendMessage{value: _fee + value}(address(0), 1, hex"0011220033", 22000);

        // gas limit exceeds the max value
        uint256 gasLimit = 100000000;
        _fee = gasOracle.l2BaseFee() * gasLimit;
        hevm.expectRevert("Gas limit must not exceed maxGasLimit");
        l1Messenger.sendMessage{value: _fee + value}(address(0), value, hex"0011220033", gasLimit);

        // update max gas limit
        l1MessageQueue.updateMaxGasLimit(gasLimit);
        l1Messenger.sendMessage{value: _fee + value}(address(0), value, hex"0011220033", gasLimit);
    }
}
