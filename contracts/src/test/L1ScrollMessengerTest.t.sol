// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {EnforcedTxGateway} from "../L1/gateways/EnforcedTxGateway.sol";
import {L1MessageQueue} from "../L1/rollup/L1MessageQueue.sol";
import {L2GasPriceOracle} from "../L1/rollup/L2GasPriceOracle.sol";
import {IScrollChain, ScrollChain} from "../L1/rollup/ScrollChain.sol";
import {Whitelist} from "../L2/predeploys/Whitelist.sol";
import {IL1ScrollMessenger, L1ScrollMessenger} from "../L1/L1ScrollMessenger.sol";
import {L2ScrollMessenger} from "../L2/L2ScrollMessenger.sol";

import {L1GatewayTestBase} from "./L1GatewayTestBase.t.sol";

contract L1ScrollMessengerTest is L1GatewayTestBase {
    event OnDropMessageCalled(bytes);
    event UpdateMaxReplayTimes(uint256 oldMaxReplayTimes, uint256 newMaxReplayTimes);

    function setUp() public {
        __L1GatewayTestBase_setUp();
    }

    function testForbidCallMessageQueueFromL2() external {
        bytes32 _xDomainCalldataHash = keccak256(
            abi.encodeWithSignature(
                "relayMessage(address,address,uint256,uint256,bytes)",
                address(this),
                address(messageQueue),
                0,
                0,
                new bytes(0)
            )
        );
        prepareL2MessageRoot(_xDomainCalldataHash);

        IL1ScrollMessenger.L2MessageProof memory proof;
        proof.batchIndex = rollup.lastFinalizedBatchIndex();

        hevm.expectRevert("Forbid to call message queue");
        l1Messenger.relayMessageWithProof(address(this), address(messageQueue), 0, 0, new bytes(0), proof);
    }

    function testForbidCallSelfFromL2() external {
        bytes32 _xDomainCalldataHash = keccak256(
            abi.encodeWithSignature(
                "relayMessage(address,address,uint256,uint256,bytes)",
                address(this),
                address(l1Messenger),
                0,
                0,
                new bytes(0)
            )
        );
        prepareL2MessageRoot(_xDomainCalldataHash);
        IL1ScrollMessenger.L2MessageProof memory proof;
        proof.batchIndex = rollup.lastFinalizedBatchIndex();

        hevm.expectRevert("Forbid to call self");
        l1Messenger.relayMessageWithProof(address(this), address(l1Messenger), 0, 0, new bytes(0), proof);
    }

    function testSendMessage(uint256 exceedValue, address refundAddress) external {
        hevm.assume(refundAddress.code.length == 0);
        hevm.assume(uint256(uint160(refundAddress)) > 100); // ignore some precompile contracts
        hevm.assume(refundAddress != address(0x000000000000000000636F6e736F6c652e6c6f67)); // ignore console/console2

        exceedValue = bound(exceedValue, 1, address(this).balance / 2);

        // Insufficient msg.value
        hevm.expectRevert("Insufficient msg.value");
        l1Messenger.sendMessage(address(0), 1, new bytes(0), defaultGasLimit, refundAddress);

        // refund exceed fee
        uint256 balanceBefore = refundAddress.balance;
        l1Messenger.sendMessage{value: 1 + exceedValue}(address(0), 1, new bytes(0), defaultGasLimit, refundAddress);
        assertEq(balanceBefore + exceedValue, refundAddress.balance);
    }

    function testReplayMessage(uint256 exceedValue, address refundAddress) external {
        hevm.assume(refundAddress.code.length == 0);
        hevm.assume(uint256(uint160(refundAddress)) > uint256(100)); // ignore some precompile contracts
        hevm.assume(refundAddress != feeVault);
        hevm.assume(refundAddress != address(0x000000000000000000636F6e736F6c652e6c6f67)); // ignore console/console2

        exceedValue = bound(exceedValue, 1, address(this).balance / 2);

        l1Messenger.updateMaxReplayTimes(0);

        // append a message
        l1Messenger.sendMessage{value: 100}(address(0), 100, new bytes(0), defaultGasLimit, refundAddress);

        // Provided message has not been enqueued
        hevm.expectRevert("Provided message has not been enqueued");
        l1Messenger.replayMessage(address(this), address(0), 101, 0, new bytes(0), defaultGasLimit, refundAddress);

        messageQueue.setL2BaseFee(1);
        // Insufficient msg.value
        hevm.expectRevert("Insufficient msg.value for fee");
        l1Messenger.replayMessage(address(this), address(0), 100, 0, new bytes(0), defaultGasLimit, refundAddress);

        uint256 _fee = messageQueue.l2BaseFee() * defaultGasLimit;

        // Exceed maximum replay times
        hevm.expectRevert("Exceed maximum replay times");
        l1Messenger.replayMessage{value: _fee}(
            address(this),
            address(0),
            100,
            0,
            new bytes(0),
            defaultGasLimit,
            refundAddress
        );

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
            defaultGasLimit,
            refundAddress
        );
        assertEq(balanceBefore + exceedValue, refundAddress.balance);
        assertEq(feeVaultBefore + _fee, feeVault.balance);

        // test replay list
        // 1. send a message with nonce 2
        // 2. replay 3 times
        messageQueue.setL2BaseFee(0);
        l1Messenger.updateMaxReplayTimes(100);
        l1Messenger.sendMessage{value: 100}(address(0), 100, new bytes(0), defaultGasLimit, refundAddress);
        bytes32 hash = keccak256(
            abi.encodeWithSignature(
                "relayMessage(address,address,uint256,uint256,bytes)",
                address(this),
                address(0),
                100,
                2,
                new bytes(0)
            )
        );
        (uint256 _replayTimes, uint256 _lastIndex) = l1Messenger.replayStates(hash);
        assertEq(_replayTimes, 0);
        assertEq(_lastIndex, 0);
        for (uint256 i = 0; i < 3; i++) {
            l1Messenger.replayMessage(address(this), address(0), 100, 2, new bytes(0), defaultGasLimit, refundAddress);
            (_replayTimes, _lastIndex) = l1Messenger.replayStates(hash);
            assertEq(_replayTimes, i + 1);
            assertEq(_lastIndex, i + 3);
            assertEq(l1Messenger.prevReplayIndex(i + 3), i + 2 + 1);
            for (uint256 j = 0; j <= i; j++) {
                assertEq(l1Messenger.prevReplayIndex(i + 3 - j), i + 2 - j + 1);
            }
        }
    }

    function testUpdateMaxReplayTimes(uint256 _maxReplayTimes) external {
        // not owner, revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        l1Messenger.updateMaxReplayTimes(_maxReplayTimes);
        hevm.stopPrank();

        hevm.expectEmit(false, false, false, true);
        emit UpdateMaxReplayTimes(3, _maxReplayTimes);

        assertEq(l1Messenger.maxReplayTimes(), 3);
        l1Messenger.updateMaxReplayTimes(_maxReplayTimes);
        assertEq(l1Messenger.maxReplayTimes(), _maxReplayTimes);
    }

    function testSetPause() external {
        // not owner, revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        l1Messenger.setPause(false);
        hevm.stopPrank();

        // pause
        l1Messenger.setPause(true);
        assertBoolEq(true, l1Messenger.paused());

        hevm.expectRevert("Pausable: paused");
        l1Messenger.sendMessage(address(0), 0, new bytes(0), defaultGasLimit);
        hevm.expectRevert("Pausable: paused");
        l1Messenger.sendMessage(address(0), 0, new bytes(0), defaultGasLimit, address(0));
        hevm.expectRevert("Pausable: paused");
        IL1ScrollMessenger.L2MessageProof memory _proof;
        l1Messenger.relayMessageWithProof(address(0), address(0), 0, 0, new bytes(0), _proof);
        hevm.expectRevert("Pausable: paused");
        l1Messenger.replayMessage(address(0), address(0), 0, 0, new bytes(0), 0, address(0));
        hevm.expectRevert("Pausable: paused");
        l1Messenger.dropMessage(address(0), address(0), 0, 0, new bytes(0));

        // unpause
        l1Messenger.setPause(false);
        assertBoolEq(false, l1Messenger.paused());
    }

    function testIntrinsicGasLimit() external {
        uint256 _fee = messageQueue.l2BaseFee() * 24000;
        uint256 value = 1;

        // _xDomainCalldata contains
        //   4B function identifier
        //   20B sender addr (encoded as 32B)
        //   20B target addr (encoded as 32B)
        //   32B value
        //   32B nonce
        //   message byte array (32B offset + 32B length + bytes (padding to multiple of 32))
        // So the intrinsic gas must be greater than 21000 + 16 * 228 = 24648
        l1Messenger.sendMessage{value: _fee + value}(address(0), value, hex"0011220033", 24648);

        // insufficient intrinsic gas
        hevm.expectRevert("Insufficient gas limit, must be above intrinsic gas");
        l1Messenger.sendMessage{value: _fee + value}(address(0), 1, hex"0011220033", 24647);

        // gas limit exceeds the max value
        uint256 gasLimit = 100000000;
        _fee = messageQueue.l2BaseFee() * gasLimit;
        hevm.expectRevert("Gas limit must not exceed maxGasLimit");
        l1Messenger.sendMessage{value: _fee + value}(address(0), value, hex"0011220033", gasLimit);

        // update max gas limit
        messageQueue.updateMaxGasLimit(gasLimit);
        l1Messenger.sendMessage{value: _fee + value}(address(0), value, hex"0011220033", gasLimit);
    }

    function testDropMessage() external {
        // Provided message has not been enqueued, revert
        hevm.expectRevert("Provided message has not been enqueued");
        l1Messenger.dropMessage(address(0), address(0), 0, 0, new bytes(0));

        // send one message with nonce 0
        l1Messenger.sendMessage(address(0), 0, new bytes(0), defaultGasLimit);
        assertEq(messageQueue.nextCrossDomainMessageIndex(), 1);

        // drop pending message, revert
        hevm.expectRevert("cannot drop pending message");
        l1Messenger.dropMessage(address(this), address(0), 0, 0, new bytes(0));

        l1Messenger.updateMaxReplayTimes(10);

        // replay 1 time
        l1Messenger.replayMessage(address(this), address(0), 0, 0, new bytes(0), defaultGasLimit, address(0));
        assertEq(messageQueue.nextCrossDomainMessageIndex(), 2);

        // skip all 2 messages
        hevm.startPrank(address(rollup));
        messageQueue.popCrossDomainMessage(0, 2, 0x3);
        assertEq(messageQueue.pendingQueueIndex(), 2);
        hevm.stopPrank();
        for (uint256 i = 0; i < 2; ++i) {
            assertBoolEq(messageQueue.isMessageSkipped(i), true);
            assertBoolEq(messageQueue.isMessageDropped(i), false);
        }
        hevm.expectEmit(false, false, false, true);
        emit OnDropMessageCalled(new bytes(0));
        l1Messenger.dropMessage(address(this), address(0), 0, 0, new bytes(0));
        for (uint256 i = 0; i < 2; ++i) {
            assertBoolEq(messageQueue.isMessageSkipped(i), true);
            assertBoolEq(messageQueue.isMessageDropped(i), true);
        }

        // send one message with nonce 2 and replay 3 times
        l1Messenger.sendMessage(address(0), 0, new bytes(0), defaultGasLimit);
        assertEq(messageQueue.nextCrossDomainMessageIndex(), 3);
        for (uint256 i = 0; i < 3; i++) {
            l1Messenger.replayMessage(address(this), address(0), 0, 2, new bytes(0), defaultGasLimit, address(0));
        }
        assertEq(messageQueue.nextCrossDomainMessageIndex(), 6);

        // only first 3 are skipped
        hevm.startPrank(address(rollup));
        messageQueue.popCrossDomainMessage(2, 4, 0x7);
        assertEq(messageQueue.pendingQueueIndex(), 6);
        hevm.stopPrank();
        for (uint256 i = 2; i < 6; i++) {
            assertBoolEq(messageQueue.isMessageSkipped(i), i < 5);
            assertBoolEq(messageQueue.isMessageDropped(i), false);
        }

        // drop non-skipped message, revert
        hevm.expectRevert("drop non-skipped message");
        l1Messenger.dropMessage(address(this), address(0), 0, 2, new bytes(0));

        // send one message with nonce 6 and replay 4 times
        l1Messenger.sendMessage(address(0), 0, new bytes(0), defaultGasLimit);
        for (uint256 i = 0; i < 4; i++) {
            l1Messenger.replayMessage(address(this), address(0), 0, 6, new bytes(0), defaultGasLimit, address(0));
        }
        assertEq(messageQueue.nextCrossDomainMessageIndex(), 11);

        // skip all 5 messages
        hevm.startPrank(address(rollup));
        messageQueue.popCrossDomainMessage(6, 5, 0x1f);
        assertEq(messageQueue.pendingQueueIndex(), 11);
        hevm.stopPrank();
        for (uint256 i = 6; i < 11; ++i) {
            assertBoolEq(messageQueue.isMessageSkipped(i), true);
            assertBoolEq(messageQueue.isMessageDropped(i), false);
        }
        hevm.expectEmit(false, false, false, true);
        emit OnDropMessageCalled(new bytes(0));
        l1Messenger.dropMessage(address(this), address(0), 0, 6, new bytes(0));
        for (uint256 i = 6; i < 11; ++i) {
            assertBoolEq(messageQueue.isMessageSkipped(i), true);
            assertBoolEq(messageQueue.isMessageDropped(i), true);
        }

        // Message already dropped, revert
        hevm.expectRevert("Message already dropped");
        l1Messenger.dropMessage(address(this), address(0), 0, 0, new bytes(0));
        hevm.expectRevert("Message already dropped");
        l1Messenger.dropMessage(address(this), address(0), 0, 6, new bytes(0));

        // replay dropped message, revert
        hevm.expectRevert("Message already dropped");
        l1Messenger.replayMessage(address(this), address(0), 0, 0, new bytes(0), defaultGasLimit, address(0));
        hevm.expectRevert("Message already dropped");
        l1Messenger.replayMessage(address(this), address(0), 0, 6, new bytes(0), defaultGasLimit, address(0));
    }

    function onDropMessage(bytes memory message) external payable {
        emit OnDropMessageCalled(message);
    }
}
