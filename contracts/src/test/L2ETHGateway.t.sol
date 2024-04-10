// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {ITransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import {L2GatewayRouter} from "../L2/gateways/L2GatewayRouter.sol";
import {IL1ETHGateway, L1ETHGateway} from "../L1/gateways/L1ETHGateway.sol";
import {IL2ETHGateway, L2ETHGateway} from "../L2/gateways/L2ETHGateway.sol";

import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";

import {L2GatewayTestBase} from "./L2GatewayTestBase.t.sol";
import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {MockGatewayRecipient} from "./mocks/MockGatewayRecipient.sol";

contract L2ETHGatewayTest is L2GatewayTestBase {
    // from L2ETHGateway
    event WithdrawETH(address indexed from, address indexed to, uint256 amount, bytes data);
    event FinalizeDepositETH(address indexed from, address indexed to, uint256 amount, bytes data);

    L2ETHGateway private gateway;
    L2GatewayRouter private router;

    L1ETHGateway private counterpartGateway;

    function setUp() public {
        setUpBase();

        // Deploy L1 contracts
        counterpartGateway = new L1ETHGateway(address(1), address(1), address(1));

        // Deploy L2 contracts
        router = L2GatewayRouter(_deployProxy(address(new L2GatewayRouter())));
        gateway = _deployGateway(address(l2Messenger));

        // Initialize L2 contracts
        gateway.initialize(address(counterpartGateway), address(router), address(l2Messenger));
        router.initialize(address(gateway), address(0));
    }

    function testInitialized() public {
        assertEq(address(counterpartGateway), gateway.counterpart());
        assertEq(address(router), gateway.router());
        assertEq(address(l2Messenger), gateway.messenger());

        hevm.expectRevert("Initializable: contract is already initialized");
        gateway.initialize(address(counterpartGateway), address(router), address(l2Messenger));
    }

    function testWithdrawETH(
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawETH(false, amount, gasLimit, feePerGas);
    }

    function testWithdrawETHWithRecipient(
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawETHWithRecipient(false, amount, recipient, gasLimit, feePerGas);
    }

    function testWithdrawETHWithRecipientAndCalldata(
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawETHWithRecipientAndCalldata(false, amount, recipient, dataToCall, gasLimit, feePerGas);
    }

    function testRouterWithdrawETH(
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawETH(true, amount, gasLimit, feePerGas);
    }

    function testRouterWithdrawETHWithRecipient(
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawETHWithRecipient(true, amount, recipient, gasLimit, feePerGas);
    }

    function testRouterWithdrawETHWithRecipientAndCalldata(
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawETHWithRecipientAndCalldata(true, amount, recipient, dataToCall, gasLimit, feePerGas);
    }

    function testFinalizeDepositETHFailedMocking(
        address sender,
        address recipient,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        amount = bound(amount, 1, address(this).balance / 2);

        // revert when caller is not messenger
        hevm.expectRevert(ErrorCallerIsNotMessenger.selector);
        gateway.finalizeDepositETH(sender, recipient, amount, dataToCall);

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway(address(mockMessenger));
        gateway.initialize(address(counterpartGateway), address(router), address(mockMessenger));

        // only call by counterpart
        hevm.expectRevert(ErrorCallerIsNotCounterpartGateway.selector);
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(gateway.finalizeDepositETH.selector, sender, recipient, amount, dataToCall)
        );

        mockMessenger.setXDomainMessageSender(address(counterpartGateway));

        // msg.value mismatch
        hevm.expectRevert("msg.value mismatch");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(gateway.finalizeDepositETH.selector, sender, recipient, amount, dataToCall)
        );

        // ETH transfer failed
        hevm.expectRevert("ETH transfer failed");
        mockMessenger.callTarget{value: amount}(
            address(gateway),
            abi.encodeWithSelector(gateway.finalizeDepositETH.selector, sender, address(this), amount, dataToCall)
        );
    }

    function testFinalizeWithdrawETHFailed(
        address sender,
        address recipient,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        amount = bound(amount, 1, address(this).balance / 2);

        // send some ETH to L2ScrollMessenger
        gateway.withdrawETH{value: amount}(amount, 21000);

        // do finalize withdraw eth
        bytes memory message = abi.encodeWithSelector(
            IL2ETHGateway.finalizeDepositETH.selector,
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

        // counterpart is not L1ETHGateway
        // emit FailedRelayedMessage from L2ScrollMessenger
        hevm.expectEmit(true, false, false, true);
        emit FailedRelayedMessage(keccak256(xDomainCalldata));

        uint256 messengerBalance = address(l2Messenger).balance;
        uint256 recipientBalance = recipient.balance;
        assertBoolEq(false, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata)));
        hevm.startPrank(AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger)));
        l2Messenger.relayMessage(
            address(uint160(address(counterpartGateway)) + 1),
            address(gateway),
            amount,
            0,
            message
        );
        hevm.stopPrank();
        assertEq(messengerBalance, address(l2Messenger).balance);
        assertEq(recipientBalance, recipient.balance);
        assertBoolEq(false, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testFinalizeWithdrawETH(
        address sender,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        MockGatewayRecipient recipient = new MockGatewayRecipient();

        amount = bound(amount, 1, address(this).balance / 2);

        // send some ETH to L2ScrollMessenger
        gateway.withdrawETH{value: amount}(amount, 21000);

        // do finalize withdraw eth
        bytes memory message = abi.encodeWithSelector(
            IL2ETHGateway.finalizeDepositETH.selector,
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

        // emit FinalizeDepositETH from L2ETHGateway
        {
            hevm.expectEmit(true, true, false, true);
            emit FinalizeDepositETH(sender, address(recipient), amount, dataToCall);
        }

        // emit RelayedMessage from L2ScrollMessenger
        {
            hevm.expectEmit(true, false, false, true);
            emit RelayedMessage(keccak256(xDomainCalldata));
        }

        uint256 messengerBalance = address(l2Messenger).balance;
        uint256 recipientBalance = address(recipient).balance;
        assertBoolEq(false, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata)));
        hevm.startPrank(AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger)));
        l2Messenger.relayMessage(address(counterpartGateway), address(gateway), amount, 0, message);
        hevm.stopPrank();
        assertEq(messengerBalance - amount, address(l2Messenger).balance);
        assertEq(recipientBalance + amount, address(recipient).balance);
        assertBoolEq(true, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata)));
    }

    function _withdrawETH(
        bool useRouter,
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, address(this).balance / 2);
        gasLimit = bound(gasLimit, 21000, 1000000);
        feePerGas = 0;

        setL1BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL1ETHGateway.finalizeWithdrawETH.selector,
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
            hevm.expectRevert("withdraw zero eth");
            if (useRouter) {
                router.withdrawETH{value: amount}(amount, gasLimit);
            } else {
                gateway.withdrawETH{value: amount}(amount, gasLimit);
            }
        } else {
            // emit AppendMessage from L2MessageQueue
            {
                hevm.expectEmit(false, false, false, true);
                emit AppendMessage(0, keccak256(xDomainCalldata));
            }

            // emit SentMessage from L2ScrollMessenger
            {
                hevm.expectEmit(true, true, false, true);
                emit SentMessage(address(gateway), address(counterpartGateway), amount, 0, gasLimit, message);
            }

            // emit WithdrawETH from L2ETHGateway
            hevm.expectEmit(true, true, false, true);
            emit WithdrawETH(address(this), address(this), amount, new bytes(0));

            uint256 messengerBalance = address(l2Messenger).balance;
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.withdrawETH{value: amount + feeToPay}(amount, gasLimit);
            } else {
                gateway.withdrawETH{value: amount + feeToPay}(amount, gasLimit);
            }
            assertEq(amount + messengerBalance, address(l2Messenger).balance);
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertGt(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        }
    }

    function _withdrawETHWithRecipient(
        bool useRouter,
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, address(this).balance / 2);
        gasLimit = bound(gasLimit, 21000, 1000000);
        feePerGas = 0;

        setL1BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL1ETHGateway.finalizeWithdrawETH.selector,
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
            hevm.expectRevert("withdraw zero eth");
            if (useRouter) {
                router.withdrawETH{value: amount}(recipient, amount, gasLimit);
            } else {
                gateway.withdrawETH{value: amount}(recipient, amount, gasLimit);
            }
        } else {
            // emit AppendMessage from L2MessageQueue
            {
                hevm.expectEmit(false, false, false, true);
                emit AppendMessage(0, keccak256(xDomainCalldata));
            }

            // emit SentMessage from L2ScrollMessenger
            {
                hevm.expectEmit(true, true, false, true);
                emit SentMessage(address(gateway), address(counterpartGateway), amount, 0, gasLimit, message);
            }

            // emit WithdrawETH from L2ETHGateway
            hevm.expectEmit(true, true, false, true);
            emit WithdrawETH(address(this), recipient, amount, new bytes(0));

            uint256 messengerBalance = address(l2Messenger).balance;
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.withdrawETH{value: amount + feeToPay}(recipient, amount, gasLimit);
            } else {
                gateway.withdrawETH{value: amount + feeToPay}(recipient, amount, gasLimit);
            }
            assertEq(amount + messengerBalance, address(l2Messenger).balance);
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertGt(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        }
    }

    function _withdrawETHWithRecipientAndCalldata(
        bool useRouter,
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, address(this).balance / 2);
        gasLimit = bound(gasLimit, 21000, 1000000);
        feePerGas = 0;

        setL1BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL1ETHGateway.finalizeWithdrawETH.selector,
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
            hevm.expectRevert("withdraw zero eth");
            if (useRouter) {
                router.withdrawETHAndCall{value: amount}(recipient, amount, dataToCall, gasLimit);
            } else {
                gateway.withdrawETHAndCall{value: amount}(recipient, amount, dataToCall, gasLimit);
            }
        } else {
            // emit AppendMessage from L2MessageQueue
            {
                hevm.expectEmit(false, false, false, true);
                emit AppendMessage(0, keccak256(xDomainCalldata));
            }

            // emit SentMessage from L2ScrollMessenger
            {
                hevm.expectEmit(true, true, false, true);
                emit SentMessage(address(gateway), address(counterpartGateway), amount, 0, gasLimit, message);
            }

            // emit WithdrawETH from L2ETHGateway
            hevm.expectEmit(true, true, false, true);
            emit WithdrawETH(address(this), recipient, amount, dataToCall);

            uint256 messengerBalance = address(l2Messenger).balance;
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.withdrawETHAndCall{value: amount + feeToPay}(recipient, amount, dataToCall, gasLimit);
            } else {
                gateway.withdrawETHAndCall{value: amount + feeToPay}(recipient, amount, dataToCall, gasLimit);
            }
            assertEq(amount + messengerBalance, address(l2Messenger).balance);
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertGt(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        }
    }

    function _deployGateway(address messenger) internal returns (L2ETHGateway _gateway) {
        _gateway = L2ETHGateway(_deployProxy(address(0)));

        admin.upgrade(
            ITransparentUpgradeableProxy(address(_gateway)),
            address(new L2ETHGateway(address(counterpartGateway), address(router), address(messenger)))
        );
    }
}
