// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {L1GatewayRouter} from "../L1/gateways/L1GatewayRouter.sol";
import {IL1ETHGateway, L1ETHGateway} from "../L1/gateways/L1ETHGateway.sol";
import {IL1ScrollMessenger} from "../L1/IL1ScrollMessenger.sol";
import {IL2ETHGateway, L2ETHGateway} from "../L2/gateways/L2ETHGateway.sol";
import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";
import {ScrollConstants} from "../libraries/constants/ScrollConstants.sol";

import {L1GatewayTestBase} from "./L1GatewayTestBase.t.sol";
import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {MockGatewayRecipient} from "./mocks/MockGatewayRecipient.sol";

contract L1ETHGatewayTest is L1GatewayTestBase {
    // from L1ETHGateway
    event DepositETH(address indexed from, address indexed to, uint256 amount, bytes data);
    event FinalizeWithdrawETH(address indexed from, address indexed to, uint256 amount, bytes data);
    event RefundETH(address indexed recipient, uint256 amount);

    L1ETHGateway private gateway;
    L1GatewayRouter private router;

    L2ETHGateway private counterpartGateway;

    function setUp() public {
        setUpBase();

        // Deploy L1 contracts
        gateway = _deployGateway();
        router = L1GatewayRouter(address(new ERC1967Proxy(address(new L1GatewayRouter()), new bytes(0))));

        // Deploy L2 contracts
        counterpartGateway = new L2ETHGateway();

        // Initialize L1 contracts
        gateway.initialize(address(counterpartGateway), address(router), address(l1Messenger));
        router.initialize(address(gateway), address(0));
    }

    function testInitialized() public {
        assertEq(address(counterpartGateway), gateway.counterpart());
        assertEq(address(router), gateway.router());
        assertEq(address(l1Messenger), gateway.messenger());

        hevm.expectRevert("Initializable: contract is already initialized");
        gateway.initialize(address(counterpartGateway), address(router), address(l1Messenger));
    }

    function testDepositETH(
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _depositETH(false, amount, gasLimit, feePerGas);
    }

    function testDepositETHWithRecipient(
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _depositETHWithRecipient(false, amount, recipient, gasLimit, feePerGas);
    }

    function testDepositETHWithRecipientAndCalldata(
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _depositETHWithRecipientAndCalldata(false, amount, recipient, dataToCall, gasLimit, feePerGas);
    }

    function testRouterDepositETH(
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _depositETH(true, amount, gasLimit, feePerGas);
    }

    function testRouterDepositETHWithRecipient(
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _depositETHWithRecipient(true, amount, recipient, gasLimit, feePerGas);
    }

    function testRouterDepositETHWithRecipientAndCalldata(
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _depositETHWithRecipientAndCalldata(true, amount, recipient, dataToCall, gasLimit, feePerGas);
    }

    function testDropMessageMocking() public {
        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway();
        gateway.initialize(address(counterpartGateway), address(router), address(mockMessenger));

        // only messenger can call, revert
        hevm.expectRevert("only messenger can call");
        gateway.onDropMessage(new bytes(0));

        // only called in drop context, revert
        hevm.expectRevert("only called in drop context");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(gateway.onDropMessage.selector, new bytes(0))
        );

        mockMessenger.setXDomainMessageSender(ScrollConstants.DROP_XDOMAIN_MESSAGE_SENDER);

        // invalid selector, revert
        hevm.expectRevert("invalid selector");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(gateway.onDropMessage.selector, new bytes(4))
        );

        bytes memory message = abi.encodeWithSelector(
            IL2ETHGateway.finalizeDepositETH.selector,
            address(this),
            address(this),
            100,
            new bytes(0)
        );

        // msg.value mismatch, revert
        hevm.expectRevert("msg.value mismatch");
        mockMessenger.callTarget{value: 99}(
            address(gateway),
            abi.encodeWithSelector(gateway.onDropMessage.selector, message)
        );
    }

    function testDropMessage(
        uint256 amount,
        address recipient,
        bytes memory dataToCall
    ) public {
        amount = bound(amount, 1, address(this).balance);
        bytes memory message = abi.encodeWithSelector(
            IL2ETHGateway.finalizeDepositETH.selector,
            address(this),
            recipient,
            amount,
            dataToCall
        );
        gateway.depositETHAndCall{value: amount}(recipient, amount, dataToCall, defaultGasLimit);

        // skip message 0
        hevm.startPrank(address(rollup));
        messageQueue.popCrossDomainMessage(0, 1, 0x1);
        assertEq(messageQueue.pendingQueueIndex(), 1);
        hevm.stopPrank();

        // ETH transfer failed, revert
        revertOnReceive = true;
        hevm.expectRevert("ETH transfer failed");
        l1Messenger.dropMessage(address(gateway), address(counterpartGateway), amount, 0, message);

        // drop message 0
        hevm.expectEmit(true, true, false, true);
        emit RefundETH(address(this), amount);

        revertOnReceive = false;
        uint256 balance = address(this).balance;
        l1Messenger.dropMessage(address(gateway), address(counterpartGateway), amount, 0, message);
        assertEq(balance + amount, address(this).balance);
    }

    function testFinalizeWithdrawETHFailedMocking(
        address sender,
        address recipient,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        amount = bound(amount, 1, address(this).balance / 2);

        // revert when caller is not messenger
        hevm.expectRevert("only messenger can call");
        gateway.finalizeWithdrawETH(sender, recipient, amount, dataToCall);

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway();
        gateway.initialize(address(counterpartGateway), address(router), address(mockMessenger));

        // only call by counterpart
        hevm.expectRevert("only call by counterpart");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(gateway.finalizeWithdrawETH.selector, sender, recipient, amount, dataToCall)
        );

        mockMessenger.setXDomainMessageSender(address(counterpartGateway));

        // msg.value mismatch
        hevm.expectRevert("msg.value mismatch");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(gateway.finalizeWithdrawETH.selector, sender, recipient, amount, dataToCall)
        );

        // ETH transfer failed
        revertOnReceive = true;
        hevm.expectRevert("ETH transfer failed");
        mockMessenger.callTarget{value: amount}(
            address(gateway),
            abi.encodeWithSelector(gateway.finalizeWithdrawETH.selector, sender, address(this), amount, dataToCall)
        );
    }

    function testFinalizeWithdrawETHFailed(
        address sender,
        address recipient,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        amount = bound(amount, 1, address(this).balance / 2);

        // deposit some ETH to L1ScrollMessenger
        gateway.depositETH{value: amount}(amount, defaultGasLimit);

        // do finalize withdraw eth
        bytes memory message = abi.encodeWithSelector(
            IL1ETHGateway.finalizeWithdrawETH.selector,
            sender,
            recipient,
            amount,
            dataToCall
        );
        bytes memory xDomainCalldata = abi.encodeWithSignature(
            "relayMessage(address,address,uint256,uint256,bytes)",
            address(uint160(address(counterpartGateway)) + 1),
            address(gateway),
            amount,
            0,
            message
        );

        prepareL2MessageRoot(keccak256(xDomainCalldata));

        IL1ScrollMessenger.L2MessageProof memory proof;
        proof.batchIndex = rollup.lastFinalizedBatchIndex();

        // counterpart is not L2ETHGateway
        // emit FailedRelayedMessage from L1ScrollMessenger
        hevm.expectEmit(true, false, false, true);
        emit FailedRelayedMessage(keccak256(xDomainCalldata));

        uint256 messengerBalance = address(l1Messenger).balance;
        uint256 recipientBalance = recipient.balance;
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(
            address(uint160(address(counterpartGateway)) + 1),
            address(gateway),
            amount,
            0,
            message,
            proof
        );
        assertEq(messengerBalance, address(l1Messenger).balance);
        assertEq(recipientBalance, recipient.balance);
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testFinalizeWithdrawETH(
        address sender,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        MockGatewayRecipient recipient = new MockGatewayRecipient();

        amount = bound(amount, 1, address(this).balance / 2);

        // deposit some ETH to L1ScrollMessenger
        gateway.depositETH{value: amount}(amount, defaultGasLimit);

        // do finalize withdraw eth
        bytes memory message = abi.encodeWithSelector(
            IL1ETHGateway.finalizeWithdrawETH.selector,
            sender,
            address(recipient),
            amount,
            dataToCall
        );
        bytes memory xDomainCalldata = abi.encodeWithSignature(
            "relayMessage(address,address,uint256,uint256,bytes)",
            address(counterpartGateway),
            address(gateway),
            amount,
            0,
            message
        );

        prepareL2MessageRoot(keccak256(xDomainCalldata));

        IL1ScrollMessenger.L2MessageProof memory proof;
        proof.batchIndex = rollup.lastFinalizedBatchIndex();

        // emit FinalizeWithdrawETH from L1ETHGateway
        {
            hevm.expectEmit(true, true, false, true);
            emit FinalizeWithdrawETH(sender, address(recipient), amount, dataToCall);
        }

        // emit RelayedMessage from L1ScrollMessenger
        {
            hevm.expectEmit(true, false, false, true);
            emit RelayedMessage(keccak256(xDomainCalldata));
        }

        uint256 messengerBalance = address(l1Messenger).balance;
        uint256 recipientBalance = address(recipient).balance;
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(address(counterpartGateway), address(gateway), amount, 0, message, proof);
        assertEq(messengerBalance - amount, address(l1Messenger).balance);
        assertEq(recipientBalance + amount, address(recipient).balance);
        assertBoolEq(true, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function _depositETH(
        bool useRouter,
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, address(this).balance / 2);
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL2ETHGateway.finalizeDepositETH.selector,
            address(this),
            address(this),
            amount,
            new bytes(0)
        );
        bytes memory xDomainCalldata = abi.encodeWithSignature(
            "relayMessage(address,address,uint256,uint256,bytes)",
            address(gateway),
            address(counterpartGateway),
            amount,
            0,
            message
        );

        if (amount == 0) {
            hevm.expectRevert("deposit zero eth");
            if (useRouter) {
                router.depositETH{value: amount}(amount, gasLimit);
            } else {
                gateway.depositETH{value: amount}(amount, gasLimit);
            }
        } else {
            // emit QueueTransaction from L1MessageQueue
            {
                hevm.expectEmit(true, true, false, true);
                address sender = AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger));
                emit QueueTransaction(sender, address(l2Messenger), 0, 0, gasLimit, xDomainCalldata);
            }

            // emit SentMessage from L1ScrollMessenger
            {
                hevm.expectEmit(true, true, false, true);
                emit SentMessage(address(gateway), address(counterpartGateway), amount, 0, gasLimit, message);
            }

            // emit DepositETH from L1ETHGateway
            hevm.expectEmit(true, true, false, true);
            emit DepositETH(address(this), address(this), amount, new bytes(0));

            uint256 messengerBalance = address(l1Messenger).balance;
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.depositETH{value: amount + feeToPay + extraValue}(amount, gasLimit);
            } else {
                gateway.depositETH{value: amount + feeToPay + extraValue}(amount, gasLimit);
            }
            assertEq(amount + messengerBalance, address(l1Messenger).balance);
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertGt(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        }
    }

    function _depositETHWithRecipient(
        bool useRouter,
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, address(this).balance / 2);
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL2ETHGateway.finalizeDepositETH.selector,
            address(this),
            recipient,
            amount,
            new bytes(0)
        );
        bytes memory xDomainCalldata = abi.encodeWithSignature(
            "relayMessage(address,address,uint256,uint256,bytes)",
            address(gateway),
            address(counterpartGateway),
            amount,
            0,
            message
        );

        if (amount == 0) {
            hevm.expectRevert("deposit zero eth");
            if (useRouter) {
                router.depositETH{value: amount}(recipient, amount, gasLimit);
            } else {
                gateway.depositETH{value: amount}(recipient, amount, gasLimit);
            }
        } else {
            // emit QueueTransaction from L1MessageQueue
            {
                hevm.expectEmit(true, true, false, true);
                address sender = AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger));
                emit QueueTransaction(sender, address(l2Messenger), 0, 0, gasLimit, xDomainCalldata);
            }

            // emit SentMessage from L1ScrollMessenger
            {
                hevm.expectEmit(true, true, false, true);
                emit SentMessage(address(gateway), address(counterpartGateway), amount, 0, gasLimit, message);
            }

            // emit DepositETH from L1ETHGateway
            hevm.expectEmit(true, true, false, true);
            emit DepositETH(address(this), recipient, amount, new bytes(0));

            uint256 messengerBalance = address(l1Messenger).balance;
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.depositETH{value: amount + feeToPay + extraValue}(recipient, amount, gasLimit);
            } else {
                gateway.depositETH{value: amount + feeToPay + extraValue}(recipient, amount, gasLimit);
            }
            assertEq(amount + messengerBalance, address(l1Messenger).balance);
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertGt(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        }
    }

    function _depositETHWithRecipientAndCalldata(
        bool useRouter,
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, address(this).balance / 2);
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL2ETHGateway.finalizeDepositETH.selector,
            address(this),
            recipient,
            amount,
            dataToCall
        );
        bytes memory xDomainCalldata = abi.encodeWithSignature(
            "relayMessage(address,address,uint256,uint256,bytes)",
            address(gateway),
            address(counterpartGateway),
            amount,
            0,
            message
        );

        if (amount == 0) {
            hevm.expectRevert("deposit zero eth");
            if (useRouter) {
                router.depositETHAndCall{value: amount}(recipient, amount, dataToCall, gasLimit);
            } else {
                gateway.depositETHAndCall{value: amount}(recipient, amount, dataToCall, gasLimit);
            }
        } else {
            // emit QueueTransaction from L1MessageQueue
            {
                hevm.expectEmit(true, true, false, true);
                address sender = AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger));
                emit QueueTransaction(sender, address(l2Messenger), 0, 0, gasLimit, xDomainCalldata);
            }

            // emit SentMessage from L1ScrollMessenger
            {
                hevm.expectEmit(true, true, false, true);
                emit SentMessage(address(gateway), address(counterpartGateway), amount, 0, gasLimit, message);
            }

            // emit DepositETH from L1ETHGateway
            hevm.expectEmit(true, true, false, true);
            emit DepositETH(address(this), recipient, amount, dataToCall);

            uint256 messengerBalance = address(l1Messenger).balance;
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.depositETHAndCall{value: amount + feeToPay + extraValue}(
                    recipient,
                    amount,
                    dataToCall,
                    gasLimit
                );
            } else {
                gateway.depositETHAndCall{value: amount + feeToPay + extraValue}(
                    recipient,
                    amount,
                    dataToCall,
                    gasLimit
                );
            }
            assertEq(amount + messengerBalance, address(l1Messenger).balance);
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertGt(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        }
    }

    function _deployGateway() internal returns (L1ETHGateway) {
        return L1ETHGateway(address(new ERC1967Proxy(address(new L1ETHGateway()), new bytes(0))));
    }
}
