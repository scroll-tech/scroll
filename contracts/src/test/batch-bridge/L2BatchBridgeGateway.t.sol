// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {MockERC20} from "solmate/test/utils/mocks/MockERC20.sol";

import {ITransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";
import {Strings} from "@openzeppelin/contracts/utils/Strings.sol";

import {L1BatchBridgeGateway} from "../../batch-bridge/L1BatchBridgeGateway.sol";
import {L2BatchBridgeGateway} from "../../batch-bridge/L2BatchBridgeGateway.sol";
import {BatchBridgeCodec} from "../../batch-bridge/BatchBridgeCodec.sol";

import {RevertOnTransferToken} from "../mocks/tokens/RevertOnTransferToken.sol";
import {MockScrollMessenger} from "../mocks/MockScrollMessenger.sol";
import {ScrollTestBase} from "../ScrollTestBase.t.sol";

contract L2BatchBridgeGatewayTest is ScrollTestBase {
    event UpdateTokenMapping(address indexed l2Token, address indexed oldL1Token, address indexed newL1Token);
    event FinalizeBatchDeposit(address indexed l1Token, address indexed l2Token, uint256 indexed batchIndex);
    event BatchDistribute(address indexed l1Token, address indexed l2Token, uint256 indexed batchIndex);
    event DistributeFailed(address indexed l2Token, uint256 indexed batchIndex, address receiver, uint256 amount);

    L1BatchBridgeGateway private counterpartBatch;
    L2BatchBridgeGateway private batch;

    MockScrollMessenger messenger;

    MockERC20 private l1Token;
    MockERC20 private l2Token;
    RevertOnTransferToken private maliciousL2Token;

    bool revertOnReceive;
    bool loopOnReceive;

    // two safe EOAs to receive ETH
    address private recipient1;
    address private recipient2;

    receive() external payable {
        if (revertOnReceive) revert();
        if (loopOnReceive) {
            for (uint256 i = 0; i < 1000000000; i++) {
                recipient1 = address(uint160(address(this)) - 1);
            }
        }
    }

    function setUp() public {
        __ScrollTestBase_setUp();

        recipient1 = address(uint160(address(this)) - 1);
        recipient2 = address(uint160(address(this)) - 2);

        // Deploy tokens
        l1Token = new MockERC20("Mock L1", "ML1", 18);
        l2Token = new MockERC20("Mock L2", "ML2", 18);
        maliciousL2Token = new RevertOnTransferToken("X", "Y", 18);

        messenger = new MockScrollMessenger();
        counterpartBatch = new L1BatchBridgeGateway(address(1), address(1), address(1), address(1));
        batch = L2BatchBridgeGateway(payable(_deployProxy(address(0))));

        // Initialize L2 contracts
        admin.upgrade(
            ITransparentUpgradeableProxy(address(batch)),
            address(new L2BatchBridgeGateway(address(counterpartBatch), address(messenger)))
        );
        batch.initialize();
    }

    function testInitialized() external {
        assertBoolEq(true, batch.hasRole(bytes32(0), address(this)));
        assertEq(address(counterpartBatch), batch.counterpart());
        assertEq(address(messenger), batch.messenger());

        hevm.expectRevert("Initializable: contract is already initialized");
        batch.initialize();
    }

    function testFinalizeBatchDeposit() external {
        // revert caller not messenger
        hevm.expectRevert(L2BatchBridgeGateway.ErrorCallerNotMessenger.selector);
        batch.finalizeBatchDeposit(address(0), address(0), 0, bytes32(0));

        // revert xDomainMessageSender not counterpart
        hevm.expectRevert(L2BatchBridgeGateway.ErrorMessageSenderNotCounterpart.selector);
        messenger.callTarget(
            address(batch),
            abi.encodeCall(
                L2BatchBridgeGateway.finalizeBatchDeposit,
                (address(l1Token), address(l2Token), 0, bytes32(0))
            )
        );

        messenger.setXDomainMessageSender(address(counterpartBatch));

        // emit FinalizeBatchDeposit
        assertEq(address(0), batch.tokenMapping(address(l2Token)));
        hevm.expectEmit(true, true, true, true);
        emit FinalizeBatchDeposit(address(l1Token), address(l2Token), 1);
        messenger.callTarget(
            address(batch),
            abi.encodeCall(
                L2BatchBridgeGateway.finalizeBatchDeposit,
                (address(l1Token), address(l2Token), 1, bytes32(uint256(1)))
            )
        );
        assertEq(address(l1Token), batch.tokenMapping(address(l2Token)));
        assertEq(batch.batchHashes(address(l2Token), 1), bytes32(uint256(1)));

        // revert token not match
        hevm.expectRevert(L2BatchBridgeGateway.ErrorL1TokenMismatched.selector);
        messenger.callTarget(
            address(batch),
            abi.encodeCall(L2BatchBridgeGateway.finalizeBatchDeposit, (address(0), address(l2Token), 0, bytes32(0)))
        );
    }

    function testFinalizeBatchDepositFuzzing(
        address token1,
        address token2,
        uint256 batchIndex,
        bytes32 hash
    ) external {
        messenger.setXDomainMessageSender(address(counterpartBatch));

        assertEq(address(0), batch.tokenMapping(token2));
        hevm.expectEmit(true, true, true, true);
        emit FinalizeBatchDeposit(token1, token2, batchIndex);
        messenger.callTarget(
            address(batch),
            abi.encodeCall(L2BatchBridgeGateway.finalizeBatchDeposit, (token1, token2, batchIndex, hash))
        );

        assertEq(token1, batch.tokenMapping(token2));
        assertEq(batch.batchHashes(token2, batchIndex), hash);
    }

    function testDistributeETH() external {
        // revert not keeper
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0xfc8737ab85eb45125971625a9ebdb75cc78e01d5c1fa80c4c6e5203f47bc4fab"
        );
        batch.distribute(address(0), 0, new bytes32[](0));
        hevm.stopPrank();

        batch.grantRole(batch.KEEPER_ROLE(), address(this));

        // revert ErrorBatchHashMismatch
        hevm.expectRevert(L2BatchBridgeGateway.ErrorBatchHashMismatch.selector);
        batch.distribute(address(0), 1, new bytes32[](0));

        // send some ETH to `L2BatchBridgeGateway`.
        messenger.setXDomainMessageSender(address(counterpartBatch));
        messenger.callTarget{value: 1 ether}(address(batch), "");

        address[] memory receivers = new address[](2);
        uint256[] memory amounts = new uint256[](2);
        receivers[0] = recipient1;
        receivers[1] = recipient2;
        amounts[0] = 100;
        amounts[1] = 200;

        // all success
        (bytes32[] memory nodes, bytes32 batchHash) = _encodeNodes(address(0), 0, receivers, amounts);
        messenger.callTarget(
            address(batch),
            abi.encodeCall(L2BatchBridgeGateway.finalizeBatchDeposit, (address(0), address(0), 0, batchHash))
        );
        assertEq(0, recipient1.balance);
        assertEq(0, recipient2.balance);
        uint256 batchBalanceBefore = address(batch).balance;
        hevm.expectEmit(true, true, true, true);
        emit BatchDistribute(address(0), address(0), 0);
        batch.distribute(address(0), 0, nodes);
        assertEq(100, recipient1.balance);
        assertEq(200, recipient2.balance);
        assertEq(batchBalanceBefore - 300, address(batch).balance);
        assertBoolEq(true, batch.isDistributed(batchHash));

        // revert ErrorBatchDistributed
        hevm.expectRevert(L2BatchBridgeGateway.ErrorBatchDistributed.selector);
        batch.distribute(address(0), 0, nodes);

        // all failed due to revert
        revertOnReceive = true;
        loopOnReceive = false;
        receivers[0] = address(this);
        receivers[1] = address(this);
        (nodes, batchHash) = _encodeNodes(address(0), 1, receivers, amounts);
        messenger.callTarget(
            address(batch),
            abi.encodeCall(L2BatchBridgeGateway.finalizeBatchDeposit, (address(0), address(0), 1, batchHash))
        );
        uint256 thisBalanceBefore = address(this).balance;
        batchBalanceBefore = address(batch).balance;
        hevm.expectEmit(true, true, false, true);
        emit DistributeFailed(address(0), 1, address(this), 100);
        hevm.expectEmit(true, true, false, true);
        emit DistributeFailed(address(0), 1, address(this), 200);
        hevm.expectEmit(true, true, true, true);
        emit BatchDistribute(address(0), address(0), 1);
        batch.distribute(address(0), 1, nodes);
        assertEq(batchBalanceBefore, address(batch).balance);
        assertEq(thisBalanceBefore, address(this).balance);
        assertBoolEq(true, batch.isDistributed(batchHash));
        assertEq(300, batch.failedAmount(address(0)));

        // all failed due to out of gas
        revertOnReceive = false;
        loopOnReceive = true;
        (nodes, batchHash) = _encodeNodes(address(0), 2, receivers, amounts);
        messenger.callTarget(
            address(batch),
            abi.encodeCall(L2BatchBridgeGateway.finalizeBatchDeposit, (address(0), address(0), 2, batchHash))
        );
        thisBalanceBefore = address(this).balance;
        batchBalanceBefore = address(batch).balance;
        hevm.expectEmit(true, true, false, true);
        emit DistributeFailed(address(0), 2, address(this), 100);
        hevm.expectEmit(true, true, false, true);
        emit DistributeFailed(address(0), 2, address(this), 200);
        hevm.expectEmit(true, true, true, true);
        emit BatchDistribute(address(0), address(0), 2);
        batch.distribute(address(0), 2, nodes);
        assertEq(batchBalanceBefore, address(batch).balance);
        assertEq(thisBalanceBefore, address(this).balance);
        assertBoolEq(true, batch.isDistributed(batchHash));
        assertEq(600, batch.failedAmount(address(0)));
    }

    function testDistributeERC20() external {
        // revert not keeper
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0xfc8737ab85eb45125971625a9ebdb75cc78e01d5c1fa80c4c6e5203f47bc4fab"
        );
        batch.distribute(address(l2Token), 0, new bytes32[](0));
        hevm.stopPrank();

        batch.grantRole(batch.KEEPER_ROLE(), address(this));

        // revert ErrorBatchHashMismatch
        hevm.expectRevert(L2BatchBridgeGateway.ErrorBatchHashMismatch.selector);
        batch.distribute(address(l2Token), 1, new bytes32[](0));

        // mint some ERC20 to `L2BatchBridgeGateway`.
        messenger.setXDomainMessageSender(address(counterpartBatch));
        l2Token.mint(address(batch), 1 ether);

        address[] memory receivers = new address[](2);
        uint256[] memory amounts = new uint256[](2);
        receivers[0] = recipient1;
        receivers[1] = recipient2;
        amounts[0] = 100;
        amounts[1] = 200;

        // all success
        (bytes32[] memory nodes, bytes32 batchHash) = _encodeNodes(address(l1Token), 0, receivers, amounts);
        messenger.callTarget(
            address(batch),
            abi.encodeCall(
                L2BatchBridgeGateway.finalizeBatchDeposit,
                (address(l1Token), address(l2Token), 0, batchHash)
            )
        );
        assertEq(0, recipient1.balance);
        assertEq(0, recipient2.balance);
        uint256 batchBalanceBefore = l2Token.balanceOf(address(batch));
        hevm.expectEmit(true, true, true, true);
        emit BatchDistribute(address(l1Token), address(l2Token), 0);
        batch.distribute(address(l2Token), 0, nodes);
        assertEq(100, l2Token.balanceOf(recipient1));
        assertEq(200, l2Token.balanceOf(recipient2));
        assertEq(batchBalanceBefore - 300, l2Token.balanceOf(address(batch)));
        assertBoolEq(true, batch.isDistributed(batchHash));

        // revert ErrorBatchDistributed
        hevm.expectRevert(L2BatchBridgeGateway.ErrorBatchDistributed.selector);
        batch.distribute(address(l2Token), 0, nodes);

        maliciousL2Token.mint(address(batch), 1 ether);

        // all failed due to revert
        maliciousL2Token.setRevertOnTransfer(true);
        receivers[0] = address(this);
        receivers[1] = address(this);
        (nodes, batchHash) = _encodeNodes(address(l1Token), 1, receivers, amounts);
        messenger.callTarget(
            address(batch),
            abi.encodeCall(
                L2BatchBridgeGateway.finalizeBatchDeposit,
                (address(l1Token), address(maliciousL2Token), 1, batchHash)
            )
        );
        uint256 thisBalanceBefore = maliciousL2Token.balanceOf(address(this));
        batchBalanceBefore = maliciousL2Token.balanceOf(address(batch));
        hevm.expectEmit(true, true, false, true);
        emit DistributeFailed(address(maliciousL2Token), 1, address(this), 100);
        hevm.expectEmit(true, true, false, true);
        emit DistributeFailed(address(maliciousL2Token), 1, address(this), 200);
        hevm.expectEmit(true, true, true, true);
        emit BatchDistribute(address(l1Token), address(maliciousL2Token), 1);
        batch.distribute(address(maliciousL2Token), 1, nodes);
        assertEq(batchBalanceBefore, maliciousL2Token.balanceOf(address(batch)));
        assertEq(thisBalanceBefore, maliciousL2Token.balanceOf(address(this)));
        assertBoolEq(true, batch.isDistributed(batchHash));
        assertEq(300, batch.failedAmount(address(maliciousL2Token)));

        // all failed due to transfer return false
        maliciousL2Token.setRevertOnTransfer(false);
        maliciousL2Token.setTransferReturn(false);
        (nodes, batchHash) = _encodeNodes(address(l1Token), 2, receivers, amounts);
        messenger.callTarget(
            address(batch),
            abi.encodeCall(
                L2BatchBridgeGateway.finalizeBatchDeposit,
                (address(l1Token), address(maliciousL2Token), 2, batchHash)
            )
        );
        thisBalanceBefore = maliciousL2Token.balanceOf(address(this));
        batchBalanceBefore = maliciousL2Token.balanceOf(address(batch));
        hevm.expectEmit(true, true, false, true);
        emit DistributeFailed(address(maliciousL2Token), 2, address(this), 100);
        hevm.expectEmit(true, true, false, true);
        emit DistributeFailed(address(maliciousL2Token), 2, address(this), 200);
        hevm.expectEmit(true, true, true, true);
        emit BatchDistribute(address(l1Token), address(maliciousL2Token), 2);
        batch.distribute(address(maliciousL2Token), 2, nodes);
        assertEq(batchBalanceBefore, maliciousL2Token.balanceOf(address(batch)));
        assertEq(thisBalanceBefore, maliciousL2Token.balanceOf(address(this)));
        assertBoolEq(true, batch.isDistributed(batchHash));
        assertEq(600, batch.failedAmount(address(maliciousL2Token)));
    }

    function testWithdrawFailedAmountETH() external {
        batch.grantRole(batch.KEEPER_ROLE(), address(this));

        // revert not admin
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0x0000000000000000000000000000000000000000000000000000000000000000"
        );
        batch.withdrawFailedAmount(address(0), address(this));
        hevm.stopPrank();

        // revert no failed
        hevm.expectRevert(L2BatchBridgeGateway.ErrorNoFailedDistribution.selector);
        batch.withdrawFailedAmount(address(0), address(this));

        // send some ETH to `L2BatchBridgeGateway`.
        messenger.setXDomainMessageSender(address(counterpartBatch));
        messenger.callTarget{value: 1 ether}(address(batch), "");

        // make a failed distribution
        address[] memory receivers = new address[](2);
        uint256[] memory amounts = new uint256[](2);
        receivers[0] = address(this);
        receivers[1] = address(this);
        amounts[0] = 100;
        amounts[1] = 200;
        revertOnReceive = true;
        (bytes32[] memory nodes, bytes32 batchHash) = _encodeNodes(address(0), 1, receivers, amounts);
        messenger.callTarget(
            address(batch),
            abi.encodeCall(L2BatchBridgeGateway.finalizeBatchDeposit, (address(0), address(0), 1, batchHash))
        );
        assertEq(0, batch.failedAmount(address(0)));
        batch.distribute(address(0), 1, nodes);
        assertEq(300, batch.failedAmount(address(0)));

        // withdraw failed
        uint256 thisBalance = recipient1.balance;
        uint256 batchBalance = address(batch).balance;
        batch.withdrawFailedAmount(address(0), recipient1);
        assertEq(0, batch.failedAmount(address(0)));
        assertEq(thisBalance + 300, recipient1.balance);
        assertEq(batchBalance - 300, address(batch).balance);

        // revert no failed
        hevm.expectRevert(L2BatchBridgeGateway.ErrorNoFailedDistribution.selector);
        batch.withdrawFailedAmount(address(0), recipient1);
    }

    function testWithdrawFailedAmountERC20() external {
        batch.grantRole(batch.KEEPER_ROLE(), address(this));

        // revert not admin
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0x0000000000000000000000000000000000000000000000000000000000000000"
        );
        batch.withdrawFailedAmount(address(0), address(this));
        hevm.stopPrank();

        // revert no failed
        hevm.expectRevert(L2BatchBridgeGateway.ErrorNoFailedDistribution.selector);
        batch.withdrawFailedAmount(address(0), address(this));

        // send some ETH to `L2BatchBridgeGateway`.
        messenger.setXDomainMessageSender(address(counterpartBatch));
        maliciousL2Token.mint(address(batch), 1 ether);

        // make a failed distribution
        address[] memory receivers = new address[](2);
        uint256[] memory amounts = new uint256[](2);
        receivers[0] = address(this);
        receivers[1] = address(this);
        amounts[0] = 100;
        amounts[1] = 200;
        maliciousL2Token.setRevertOnTransfer(true);
        (bytes32[] memory nodes, bytes32 batchHash) = _encodeNodes(address(l1Token), 1, receivers, amounts);
        messenger.callTarget(
            address(batch),
            abi.encodeCall(
                L2BatchBridgeGateway.finalizeBatchDeposit,
                (address(l1Token), address(maliciousL2Token), 1, batchHash)
            )
        );
        assertEq(0, batch.failedAmount(address(maliciousL2Token)));
        batch.distribute(address(maliciousL2Token), 1, nodes);
        assertEq(300, batch.failedAmount(address(maliciousL2Token)));

        // withdraw failed
        maliciousL2Token.setRevertOnTransfer(false);
        maliciousL2Token.setTransferReturn(true);
        uint256 thisBalance = maliciousL2Token.balanceOf(recipient1);
        uint256 batchBalance = maliciousL2Token.balanceOf(address(batch));
        batch.withdrawFailedAmount(address(maliciousL2Token), recipient1);
        assertEq(0, batch.failedAmount(address(maliciousL2Token)));
        assertEq(thisBalance + 300, maliciousL2Token.balanceOf(recipient1));
        assertEq(batchBalance - 300, maliciousL2Token.balanceOf(address(batch)));

        // revert no failed
        hevm.expectRevert(L2BatchBridgeGateway.ErrorNoFailedDistribution.selector);
        batch.withdrawFailedAmount(address(maliciousL2Token), recipient1);
    }

    function _encodeNodes(
        address token,
        uint256 batchIndex,
        address[] memory receivers,
        uint256[] memory amounts
    ) private returns (bytes32[] memory nodes, bytes32 hash) {
        nodes = new bytes32[](receivers.length);
        hash = BatchBridgeCodec.encodeInitialNode(token, uint64(batchIndex));
        for (uint256 i = 0; i < receivers.length; i++) {
            nodes[i] = BatchBridgeCodec.encodeNode(receivers[i], uint96(amounts[i]));
            hash = BatchBridgeCodec.hash(hash, nodes[i]);
        }
    }
}
