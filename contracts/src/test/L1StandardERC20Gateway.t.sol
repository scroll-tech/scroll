// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {MockERC20} from "solmate/test/utils/mocks/MockERC20.sol";

import {ITransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import {L1GatewayRouter} from "../L1/gateways/L1GatewayRouter.sol";
import {IL1ERC20Gateway, L1StandardERC20Gateway} from "../L1/gateways/L1StandardERC20Gateway.sol";
import {IL1ScrollMessenger} from "../L1/IL1ScrollMessenger.sol";
import {IL2ERC20Gateway, L2StandardERC20Gateway} from "../L2/gateways/L2StandardERC20Gateway.sol";
import {ScrollStandardERC20} from "../libraries/token/ScrollStandardERC20.sol";
import {ScrollStandardERC20Factory} from "../libraries/token/ScrollStandardERC20Factory.sol";
import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";
import {ScrollConstants} from "../libraries/constants/ScrollConstants.sol";

import {L1GatewayTestBase} from "./L1GatewayTestBase.t.sol";
import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {TransferReentrantToken} from "./mocks/tokens/TransferReentrantToken.sol";
import {FeeOnTransferToken} from "./mocks/tokens/FeeOnTransferToken.sol";
import {MockGatewayRecipient} from "./mocks/MockGatewayRecipient.sol";

contract L1StandardERC20GatewayTest is L1GatewayTestBase {
    // from L1StandardERC20Gateway
    event FinalizeWithdrawERC20(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );
    event DepositERC20(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );
    event RefundERC20(address indexed token, address indexed recipient, uint256 amount);

    ScrollStandardERC20 private template;
    ScrollStandardERC20Factory private factory;

    L1StandardERC20Gateway private gateway;
    L1GatewayRouter private router;

    L2StandardERC20Gateway private counterpartGateway;

    MockERC20 private l1Token;
    MockERC20 private l2Token;
    TransferReentrantToken private reentrantToken;
    FeeOnTransferToken private feeToken;

    function setUp() public {
        __L1GatewayTestBase_setUp();

        // Deploy tokens
        l1Token = new MockERC20("Mock", "M", 18);
        reentrantToken = new TransferReentrantToken("Reentrant", "R", 18);
        feeToken = new FeeOnTransferToken("Fee", "F", 18);

        // Deploy L2 contracts
        template = new ScrollStandardERC20();
        factory = new ScrollStandardERC20Factory(address(template));
        counterpartGateway = new L2StandardERC20Gateway(address(1), address(1), address(1), address(factory));

        // Deploy L1 contracts
        router = L1GatewayRouter(_deployProxy(address(new L1GatewayRouter())));
        gateway = _deployGateway(address(l1Messenger));

        // Initialize L1 contracts
        gateway.initialize(
            address(counterpartGateway),
            address(router),
            address(l1Messenger),
            address(template),
            address(factory)
        );
        router.initialize(address(0), address(gateway));

        // Prepare token balances
        l2Token = MockERC20(gateway.getL2ERC20Address(address(l1Token)));
        l1Token.mint(address(this), type(uint128).max);
        l1Token.approve(address(gateway), type(uint256).max);
        l1Token.approve(address(router), type(uint256).max);

        reentrantToken.mint(address(this), type(uint128).max);
        reentrantToken.approve(address(gateway), type(uint256).max);
        reentrantToken.approve(address(router), type(uint256).max);

        feeToken.mint(address(this), type(uint128).max);
        feeToken.approve(address(gateway), type(uint256).max);
        feeToken.approve(address(router), type(uint256).max);
    }

    function testInitialized() public {
        assertEq(address(counterpartGateway), gateway.counterpart());
        assertEq(address(router), gateway.router());
        assertEq(address(l1Messenger), gateway.messenger());
        assertEq(address(template), gateway.l2TokenImplementation());
        assertEq(address(factory), gateway.l2TokenFactory());

        hevm.expectRevert("Initializable: contract is already initialized");
        gateway.initialize(
            address(counterpartGateway),
            address(router),
            address(l1Messenger),
            address(template),
            address(factory)
        );
    }

    function testGetL2ERC20Address(address l1Address) public {
        assertEq(
            gateway.getL2ERC20Address(l1Address),
            factory.computeL2TokenAddress(address(counterpartGateway), l1Address)
        );
    }

    function testDepositERC20(
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _depositERC20(false, amount, gasLimit, feePerGas);
    }

    function testDepositERC20WithRecipient(
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _depositERC20WithRecipient(false, amount, recipient, gasLimit, feePerGas);
    }

    function testDepositERC20WithRecipientAndCalldata(
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _depositERC20WithRecipientAndCalldata(false, amount, recipient, dataToCall, gasLimit, feePerGas);
    }

    function testRouterDepositERC20(
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _depositERC20(true, amount, gasLimit, feePerGas);
    }

    function testRouterDepositERC20WithRecipient(
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _depositERC20WithRecipient(true, amount, recipient, gasLimit, feePerGas);
    }

    function testRouterDepositERC20WithRecipientAndCalldata(
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _depositERC20WithRecipientAndCalldata(true, amount, recipient, dataToCall, gasLimit, feePerGas);
    }

    function testDepositReentrantToken(uint256 amount) public {
        // should revert, reentrant before transfer
        reentrantToken.setReentrantCall(
            address(gateway),
            0,
            abi.encodeWithSignature("depositERC20(address,uint256,uint256)", address(0), 1, 0),
            true
        );
        amount = bound(amount, 1, reentrantToken.balanceOf(address(this)));
        hevm.expectRevert("ReentrancyGuard: reentrant call");
        gateway.depositERC20(address(reentrantToken), amount, defaultGasLimit);

        // should revert, reentrant after transfer
        reentrantToken.setReentrantCall(
            address(gateway),
            0,
            abi.encodeWithSignature("depositERC20(address,uint256,uint256)", address(0), 1, 0),
            false
        );
        amount = bound(amount, 1, reentrantToken.balanceOf(address(this)));
        hevm.expectRevert("ReentrancyGuard: reentrant call");
        gateway.depositERC20(address(reentrantToken), amount, defaultGasLimit);
    }

    function testFeeOnTransferTokenFailed(uint256 amount) public {
        feeToken.setFeeRate(1e9);
        amount = bound(amount, 1, feeToken.balanceOf(address(this)));
        hevm.expectRevert("deposit zero amount");
        gateway.depositERC20(address(feeToken), amount, defaultGasLimit);
    }

    function testFeeOnTransferTokenSucceed(uint256 amount, uint256 feeRate) public {
        feeRate = bound(feeRate, 0, 1e9 - 1);
        amount = bound(amount, 1e9, feeToken.balanceOf(address(this)));
        feeToken.setFeeRate(feeRate);

        // should succeed, for valid amount
        uint256 balanceBefore = feeToken.balanceOf(address(gateway));
        uint256 fee = (amount * feeRate) / 1e9;
        gateway.depositERC20(address(feeToken), amount, defaultGasLimit);
        uint256 balanceAfter = feeToken.balanceOf(address(gateway));
        assertEq(balanceBefore + amount - fee, balanceAfter);
    }

    function testDropMessageMocking() public {
        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway(address(mockMessenger));
        gateway.initialize(
            address(counterpartGateway),
            address(router),
            address(mockMessenger),
            address(template),
            address(factory)
        );

        // only messenger can call, revert
        hevm.expectRevert(ErrorCallerIsNotMessenger.selector);
        gateway.onDropMessage(new bytes(0));

        // only called in drop context, revert
        hevm.expectRevert(ErrorNotInDropMessageContext.selector);
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
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            address(this),
            100,
            new bytes(0)
        );

        // nonzero msg.value, revert
        hevm.expectRevert("nonzero msg.value");
        mockMessenger.callTarget{value: 1}(
            address(gateway),
            abi.encodeWithSelector(gateway.onDropMessage.selector, message)
        );
    }

    function testDropMessage(
        uint256 amount,
        address recipient,
        bytes memory dataToCall
    ) public {
        amount = bound(amount, 1, l1Token.balanceOf(address(this)) / 2);
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            recipient,
            amount,
            abi.encode(true, abi.encode(dataToCall, abi.encode(l1Token.symbol(), l1Token.name(), l1Token.decimals())))
        );
        gateway.depositERC20AndCall(address(l1Token), recipient, amount, dataToCall, defaultGasLimit);
        gateway.depositERC20AndCall(address(l1Token), recipient, amount, dataToCall, defaultGasLimit);

        // skip message 0 and 1
        hevm.startPrank(address(rollup));
        messageQueue.popCrossDomainMessage(0, 2, 0x3);
        assertEq(messageQueue.pendingQueueIndex(), 2);
        hevm.stopPrank();

        // drop message 1
        hevm.expectEmit(true, true, false, true);
        emit RefundERC20(address(l1Token), address(this), amount);

        uint256 balance = l1Token.balanceOf(address(this));
        l1Messenger.dropMessage(address(gateway), address(counterpartGateway), 0, 1, message);
        assertEq(balance + amount, l1Token.balanceOf(address(this)));
    }

    function testFinalizeWithdrawERC20FailedMocking(
        address sender,
        address recipient,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        amount = bound(amount, 1, 100000);

        // revert when caller is not messenger
        hevm.expectRevert(ErrorCallerIsNotMessenger.selector);
        gateway.finalizeWithdrawERC20(address(l1Token), address(l2Token), sender, recipient, amount, dataToCall);

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway(address(mockMessenger));
        gateway.initialize(
            address(counterpartGateway),
            address(router),
            address(mockMessenger),
            address(template),
            address(factory)
        );

        // only call by counterpart
        hevm.expectRevert(ErrorCallerIsNotCounterpartGateway.selector);
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeWithdrawERC20.selector,
                address(l1Token),
                address(l2Token),
                sender,
                recipient,
                amount,
                dataToCall
            )
        );

        mockMessenger.setXDomainMessageSender(address(counterpartGateway));

        // msg.value mismatch
        hevm.expectRevert("nonzero msg.value");
        mockMessenger.callTarget{value: 1}(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeWithdrawERC20.selector,
                address(l1Token),
                address(l2Token),
                sender,
                recipient,
                amount,
                dataToCall
            )
        );
    }

    function testFinalizeWithdrawERC20Failed(
        address sender,
        address recipient,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        // blacklist some addresses
        hevm.assume(recipient != address(0));

        amount = bound(amount, 1, l1Token.balanceOf(address(this)));

        // deposit some token to L1StandardERC20Gateway
        gateway.depositERC20(address(l1Token), amount, defaultGasLimit);

        // do finalize withdraw token
        bytes memory message = abi.encodeWithSelector(
            IL1ERC20Gateway.finalizeWithdrawERC20.selector,
            address(l1Token),
            address(l2Token),
            sender,
            recipient,
            amount,
            dataToCall
        );
        bytes memory xDomainCalldata = abi.encodeWithSignature(
            "relayMessage(address,address,uint256,uint256,bytes)",
            address(uint160(address(counterpartGateway)) + 1),
            address(gateway),
            0,
            0,
            message
        );

        prepareL2MessageRoot(keccak256(xDomainCalldata));

        IL1ScrollMessenger.L2MessageProof memory proof;
        proof.batchIndex = rollup.lastFinalizedBatchIndex();

        // counterpart is not L2WETHGateway
        // emit FailedRelayedMessage from L1ScrollMessenger
        hevm.expectEmit(true, false, false, true);
        emit FailedRelayedMessage(keccak256(xDomainCalldata));

        uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
        uint256 recipientBalance = l1Token.balanceOf(recipient);
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(
            address(uint160(address(counterpartGateway)) + 1),
            address(gateway),
            0,
            0,
            message,
            proof
        );
        assertEq(gatewayBalance, l1Token.balanceOf(address(gateway)));
        assertEq(recipientBalance, l1Token.balanceOf(recipient));
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testFinalizeWithdrawERC20(
        address sender,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        MockGatewayRecipient recipient = new MockGatewayRecipient();

        amount = bound(amount, 1, l1Token.balanceOf(address(this)));

        // deposit some token to L1StandardERC20Gateway
        gateway.depositERC20(address(l1Token), amount, defaultGasLimit);

        // do finalize withdraw token
        bytes memory message = abi.encodeWithSelector(
            IL1ERC20Gateway.finalizeWithdrawERC20.selector,
            address(l1Token),
            address(l2Token),
            sender,
            address(recipient),
            amount,
            dataToCall
        );
        bytes memory xDomainCalldata = abi.encodeWithSignature(
            "relayMessage(address,address,uint256,uint256,bytes)",
            address(counterpartGateway),
            address(gateway),
            0,
            0,
            message
        );

        prepareL2MessageRoot(keccak256(xDomainCalldata));

        IL1ScrollMessenger.L2MessageProof memory proof;
        proof.batchIndex = rollup.lastFinalizedBatchIndex();

        // emit FinalizeWithdrawERC20 from L1StandardERC20Gateway
        {
            hevm.expectEmit(true, true, true, true);
            emit FinalizeWithdrawERC20(
                address(l1Token),
                address(l2Token),
                sender,
                address(recipient),
                amount,
                dataToCall
            );
        }

        // emit RelayedMessage from L1ScrollMessenger
        {
            hevm.expectEmit(true, false, false, true);
            emit RelayedMessage(keccak256(xDomainCalldata));
        }

        uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
        uint256 recipientBalance = l1Token.balanceOf(address(recipient));
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(address(counterpartGateway), address(gateway), 0, 0, message, proof);
        assertEq(gatewayBalance - amount, l1Token.balanceOf(address(gateway)));
        assertEq(recipientBalance + amount, l1Token.balanceOf(address(recipient)));
        assertBoolEq(true, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function _depositERC20(
        bool useRouter,
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, l1Token.balanceOf(address(this)));
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        messageQueue.setL2BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            address(this),
            amount,
            abi.encode(true, abi.encode(new bytes(0), abi.encode(l1Token.symbol(), l1Token.name(), l1Token.decimals())))
        );
        bytes memory xDomainCalldata = abi.encodeWithSignature(
            "relayMessage(address,address,uint256,uint256,bytes)",
            address(gateway),
            address(counterpartGateway),
            0,
            0,
            message
        );

        if (amount == 0) {
            hevm.expectRevert("deposit zero amount");
            if (useRouter) {
                router.depositERC20{value: feeToPay + extraValue}(address(l1Token), amount, gasLimit);
            } else {
                gateway.depositERC20{value: feeToPay + extraValue}(address(l1Token), amount, gasLimit);
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
                emit SentMessage(address(gateway), address(counterpartGateway), 0, 0, gasLimit, message);
            }

            // emit DepositERC20 from L1StandardERC20Gateway
            hevm.expectEmit(true, true, true, true);
            emit DepositERC20(address(l1Token), address(l2Token), address(this), address(this), amount, new bytes(0));

            uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.depositERC20{value: feeToPay + extraValue}(address(l1Token), amount, gasLimit);
            } else {
                gateway.depositERC20{value: feeToPay + extraValue}(address(l1Token), amount, gasLimit);
            }
            assertEq(amount + gatewayBalance, l1Token.balanceOf(address(gateway)));
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertGt(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        }
    }

    function _depositERC20WithRecipient(
        bool useRouter,
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, l1Token.balanceOf(address(this)));
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        messageQueue.setL2BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            recipient,
            amount,
            abi.encode(true, abi.encode(new bytes(0), abi.encode(l1Token.symbol(), l1Token.name(), l1Token.decimals())))
        );
        bytes memory xDomainCalldata = abi.encodeWithSignature(
            "relayMessage(address,address,uint256,uint256,bytes)",
            address(gateway),
            address(counterpartGateway),
            0,
            0,
            message
        );

        if (amount == 0) {
            hevm.expectRevert("deposit zero amount");
            if (useRouter) {
                router.depositERC20{value: feeToPay + extraValue}(address(l1Token), recipient, amount, gasLimit);
            } else {
                gateway.depositERC20{value: feeToPay + extraValue}(address(l1Token), recipient, amount, gasLimit);
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
                emit SentMessage(address(gateway), address(counterpartGateway), 0, 0, gasLimit, message);
            }

            // emit DepositERC20 from L1StandardERC20Gateway
            hevm.expectEmit(true, true, true, true);
            emit DepositERC20(address(l1Token), address(l2Token), address(this), recipient, amount, new bytes(0));

            uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.depositERC20{value: feeToPay + extraValue}(address(l1Token), recipient, amount, gasLimit);
            } else {
                gateway.depositERC20{value: feeToPay + extraValue}(address(l1Token), recipient, amount, gasLimit);
            }
            assertEq(amount + gatewayBalance, l1Token.balanceOf(address(gateway)));
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertGt(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        }
    }

    function _depositERC20WithRecipientAndCalldata(
        bool useRouter,
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, l1Token.balanceOf(address(this)));
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        messageQueue.setL2BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            recipient,
            amount,
            abi.encode(true, abi.encode(dataToCall, abi.encode(l1Token.symbol(), l1Token.name(), l1Token.decimals())))
        );
        bytes memory xDomainCalldata = abi.encodeWithSignature(
            "relayMessage(address,address,uint256,uint256,bytes)",
            address(gateway),
            address(counterpartGateway),
            0,
            0,
            message
        );

        if (amount == 0) {
            hevm.expectRevert("deposit zero amount");
            if (useRouter) {
                router.depositERC20AndCall{value: feeToPay + extraValue}(
                    address(l1Token),
                    recipient,
                    amount,
                    dataToCall,
                    gasLimit
                );
            } else {
                gateway.depositERC20AndCall{value: feeToPay + extraValue}(
                    address(l1Token),
                    recipient,
                    amount,
                    dataToCall,
                    gasLimit
                );
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
                emit SentMessage(address(gateway), address(counterpartGateway), 0, 0, gasLimit, message);
            }

            // emit DepositERC20 from L1StandardERC20Gateway
            hevm.expectEmit(true, true, true, true);
            emit DepositERC20(address(l1Token), address(l2Token), address(this), recipient, amount, dataToCall);

            uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.depositERC20AndCall{value: feeToPay + extraValue}(
                    address(l1Token),
                    recipient,
                    amount,
                    dataToCall,
                    gasLimit
                );
            } else {
                gateway.depositERC20AndCall{value: feeToPay + extraValue}(
                    address(l1Token),
                    recipient,
                    amount,
                    dataToCall,
                    gasLimit
                );
            }
            assertEq(amount + gatewayBalance, l1Token.balanceOf(address(gateway)));
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertGt(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        }
    }

    function _deployGateway(address messenger) internal returns (L1StandardERC20Gateway _gateway) {
        _gateway = L1StandardERC20Gateway(_deployProxy(address(0)));

        admin.upgrade(
            ITransparentUpgradeableProxy(address(_gateway)),
            address(
                new L1StandardERC20Gateway(
                    address(counterpartGateway),
                    address(router),
                    address(messenger),
                    address(template),
                    address(factory)
                )
            )
        );
    }
}
