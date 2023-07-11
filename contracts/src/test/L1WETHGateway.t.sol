// SPDX-License-Identifier: MIT

pragma solidity =0.8.20;

import {WETH} from "solmate/tokens/WETH.sol";

import {L1GatewayRouter} from "../L1/gateways/L1GatewayRouter.sol";
import {IL1ERC20Gateway, L1WETHGateway} from "../L1/gateways/L1WETHGateway.sol";
import {IL1ScrollMessenger} from "../L1/IL1ScrollMessenger.sol";
import {IL2ERC20Gateway, L2WETHGateway} from "../L2/gateways/L2WETHGateway.sol";
import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";
import {ScrollConstants} from "../libraries/constants/ScrollConstants.sol";

import {L1GatewayTestBase} from "./L1GatewayTestBase.t.sol";
import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {MockGatewayRecipient} from "./mocks/MockGatewayRecipient.sol";

contract L1WETHGatewayTest is L1GatewayTestBase {
    // from L1WETHGateway
    event FinalizeWithdrawERC20(
        address indexed _l1weth,
        address indexed _l2weth,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );
    event DepositERC20(
        address indexed _l1weth,
        address indexed _l2weth,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );
    event RefundERC20(address indexed token, address indexed recipient, uint256 amount);

    WETH private l1weth;
    WETH private l2weth;

    L1WETHGateway private gateway;
    L1GatewayRouter private router;

    L2WETHGateway private counterpartGateway;

    function setUp() public {
        setUpBase();

        // Deploy tokens
        l1weth = new WETH();
        l2weth = new WETH();

        // Deploy L1 contracts
        gateway = new L1WETHGateway(address(l1weth), address(l2weth));
        router = new L1GatewayRouter();

        // Deploy L2 contracts
        counterpartGateway = new L2WETHGateway(address(l2weth), address(l1weth));

        // Initialize L1 contracts
        gateway.initialize(address(counterpartGateway), address(router), address(l1Messenger));
        router.initialize(address(0), address(gateway));

        // Prepare token balances
        l1weth.deposit{value: address(this).balance / 2}();
        l1weth.approve(address(gateway), type(uint256).max);
        l1weth.approve(address(router), type(uint256).max);
    }

    function testInitialized() public {
        assertEq(address(counterpartGateway), gateway.counterpart());
        assertEq(address(router), gateway.router());
        assertEq(address(l1Messenger), gateway.messenger());
        assertEq(address(l1weth), gateway.WETH());
        assertEq(address(l2weth), gateway.l2WETH());
        assertEq(address(l2weth), gateway.getL2ERC20Address(address(l1weth)));

        hevm.expectRevert("Initializable: contract is already initialized");
        gateway.initialize(address(counterpartGateway), address(router), address(l1Messenger));
    }

    function testDirectTransferETH(uint256 amount) public {
        amount = bound(amount, 0, address(this).balance);
        // solhint-disable-next-line avoid-low-level-calls
        (bool success, bytes memory result) = address(gateway).call{value: amount}("");
        assertBoolEq(success, false);
        assertEq(string(result), string(abi.encodeWithSignature("Error(string)", "only WETH")));
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

    function testDropMessageMocking() public {
        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = new L1WETHGateway(address(l1weth), address(l2weth));
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
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1weth),
            address(l2weth),
            address(this),
            address(this),
            100,
            new bytes(0)
        );

        // token not WETH, revert
        hevm.expectRevert("token not WETH");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.onDropMessage.selector,
                abi.encodeWithSelector(
                    IL2ERC20Gateway.finalizeDepositERC20.selector,
                    address(l2weth),
                    address(l2weth),
                    address(this),
                    address(this),
                    100,
                    new bytes(0)
                )
            )
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
        amount = bound(amount, 1, l1weth.balanceOf(address(this)));
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1weth),
            address(l2weth),
            address(this),
            recipient,
            amount,
            dataToCall
        );
        gateway.depositERC20AndCall(address(l1weth), recipient, amount, dataToCall, 0);

        // skip message 0
        hevm.startPrank(address(rollup));
        messageQueue.popCrossDomainMessage(0, 1, 0x1);
        assertEq(messageQueue.pendingQueueIndex(), 1);
        hevm.stopPrank();

        // drop message 0
        hevm.expectEmit(true, true, false, true);
        emit RefundERC20(address(l1weth), address(this), amount);

        uint256 balance = l1weth.balanceOf(address(this));
        l1Messenger.dropMessage(address(gateway), address(counterpartGateway), amount, 0, message);
        assertEq(balance + amount, l1weth.balanceOf(address(this)));
    }

    function testFinalizeWithdrawERC20FailedMocking(
        address sender,
        address recipient,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        amount = bound(amount, 1, 100000);

        // revert when caller is not messenger
        hevm.expectRevert("only messenger can call");
        gateway.finalizeWithdrawERC20(address(l1weth), address(l2weth), sender, recipient, amount, dataToCall);

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = new L1WETHGateway(address(l1weth), address(l2weth));
        gateway.initialize(address(counterpartGateway), address(router), address(mockMessenger));

        // only call by counterpart
        hevm.expectRevert("only call by counterpart");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeWithdrawERC20.selector,
                address(l1weth),
                address(l2weth),
                sender,
                recipient,
                amount,
                dataToCall
            )
        );

        mockMessenger.setXDomainMessageSender(address(counterpartGateway));

        // l1 token not WETH
        hevm.expectRevert("l1 token not WETH");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeWithdrawERC20.selector,
                address(l2weth),
                address(l2weth),
                sender,
                recipient,
                amount,
                dataToCall
            )
        );

        // l2 token not WETH
        hevm.expectRevert("l2 token not WETH");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeWithdrawERC20.selector,
                address(l1weth),
                address(l1weth),
                sender,
                recipient,
                amount,
                dataToCall
            )
        );

        // msg.value mismatch
        hevm.expectRevert("msg.value mismatch");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeWithdrawERC20.selector,
                address(l1weth),
                address(l2weth),
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
        hevm.assume(recipient != address(gateway));

        amount = bound(amount, 1, l1weth.balanceOf(address(this)));

        // deposit some WETH to L1ScrollMessenger
        gateway.depositERC20(address(l1weth), amount, 0);

        // do finalize withdraw eth
        bytes memory message = abi.encodeWithSelector(
            IL1ERC20Gateway.finalizeWithdrawERC20.selector,
            address(l1weth),
            address(l2weth),
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

        // counterpart is not L2WETHGateway
        // emit FailedRelayedMessage from L1ScrollMessenger
        hevm.expectEmit(true, false, false, true);
        emit FailedRelayedMessage(keccak256(xDomainCalldata));

        uint256 messengerBalance = address(l1Messenger).balance;
        uint256 recipientBalance = l1weth.balanceOf(recipient);
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
        assertEq(recipientBalance, l1weth.balanceOf(recipient));
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testFinalizeWithdrawERC20(
        address sender,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        MockGatewayRecipient recipient = new MockGatewayRecipient();

        amount = bound(amount, 1, l1weth.balanceOf(address(this)));

        // deposit some WETH to L1ScrollMessenger
        gateway.depositERC20(address(l1weth), amount, 0);

        // do finalize withdraw eth
        bytes memory message = abi.encodeWithSelector(
            IL1ERC20Gateway.finalizeWithdrawERC20.selector,
            address(l1weth),
            address(l2weth),
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

        // emit FinalizeWithdrawERC20 from L1WETHGateway
        {
            hevm.expectEmit(true, true, true, true);
            emit FinalizeWithdrawERC20(
                address(l1weth),
                address(l2weth),
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

        uint256 messengerBalance = address(l1Messenger).balance;
        uint256 recipientBalance = l1weth.balanceOf(address(recipient));
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(address(counterpartGateway), address(gateway), amount, 0, message, proof);
        assertEq(messengerBalance - amount, address(l1Messenger).balance);
        assertEq(recipientBalance + amount, l1weth.balanceOf(address(recipient)));
        assertBoolEq(true, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function _depositERC20(
        bool useRouter,
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, l1weth.balanceOf(address(this)));
        gasLimit = bound(gasLimit, 0, 1000000);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1weth),
            address(l2weth),
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
            hevm.expectRevert("deposit zero amount");
            if (useRouter) {
                router.depositERC20{value: feeToPay + extraValue}(address(l1weth), amount, gasLimit);
            } else {
                gateway.depositERC20{value: feeToPay + extraValue}(address(l1weth), amount, gasLimit);
            }
        } else {
            // token is not l1WETH
            hevm.expectRevert("only WETH is allowed");
            gateway.depositERC20(address(l2weth), amount, gasLimit);

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

            // emit DepositERC20 from L1WETHGateway
            hevm.expectEmit(true, true, true, true);
            emit DepositERC20(address(l1weth), address(l2weth), address(this), address(this), amount, new bytes(0));

            uint256 messengerBalance = address(l1Messenger).balance;
            uint256 feeVaultBalance = address(feeVault).balance;
            assertBoolEq(false, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
            if (useRouter) {
                router.depositERC20{value: feeToPay + extraValue}(address(l1weth), amount, gasLimit);
            } else {
                gateway.depositERC20{value: feeToPay + extraValue}(address(l1weth), amount, gasLimit);
            }
            assertEq(amount + messengerBalance, address(l1Messenger).balance);
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertBoolEq(true, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
        }
    }

    function _depositERC20WithRecipient(
        bool useRouter,
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, l1weth.balanceOf(address(this)));
        gasLimit = bound(gasLimit, 0, 1000000);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1weth),
            address(l2weth),
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
            hevm.expectRevert("deposit zero amount");
            if (useRouter) {
                router.depositERC20{value: feeToPay + extraValue}(address(l1weth), recipient, amount, gasLimit);
            } else {
                gateway.depositERC20{value: feeToPay + extraValue}(address(l1weth), recipient, amount, gasLimit);
            }
        } else {
            // token is not l1WETH
            hevm.expectRevert("only WETH is allowed");
            gateway.depositERC20(address(l2weth), recipient, amount, gasLimit);

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

            // emit DepositERC20 from L1WETHGateway
            hevm.expectEmit(true, true, true, true);
            emit DepositERC20(address(l1weth), address(l2weth), address(this), recipient, amount, new bytes(0));

            uint256 messengerBalance = address(l1Messenger).balance;
            uint256 feeVaultBalance = address(feeVault).balance;
            assertBoolEq(false, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
            if (useRouter) {
                router.depositERC20{value: feeToPay + extraValue}(address(l1weth), recipient, amount, gasLimit);
            } else {
                gateway.depositERC20{value: feeToPay + extraValue}(address(l1weth), recipient, amount, gasLimit);
            }
            assertEq(amount + messengerBalance, address(l1Messenger).balance);
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertBoolEq(true, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
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
        amount = bound(amount, 0, l1weth.balanceOf(address(this)));
        gasLimit = bound(gasLimit, 0, 1000000);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1weth),
            address(l2weth),
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
            hevm.expectRevert("deposit zero amount");
            if (useRouter) {
                router.depositERC20AndCall{value: feeToPay + extraValue}(
                    address(l1weth),
                    recipient,
                    amount,
                    dataToCall,
                    gasLimit
                );
            } else {
                gateway.depositERC20AndCall{value: feeToPay + extraValue}(
                    address(l1weth),
                    recipient,
                    amount,
                    dataToCall,
                    gasLimit
                );
            }
        } else {
            // token is not l1WETH
            hevm.expectRevert("only WETH is allowed");
            gateway.depositERC20AndCall(address(l2weth), recipient, amount, dataToCall, gasLimit);

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

            // emit DepositERC20 from L1WETHGateway
            hevm.expectEmit(true, true, true, true);
            emit DepositERC20(address(l1weth), address(l2weth), address(this), recipient, amount, dataToCall);

            uint256 messengerBalance = address(l1Messenger).balance;
            uint256 feeVaultBalance = address(feeVault).balance;
            assertBoolEq(false, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
            if (useRouter) {
                router.depositERC20AndCall{value: feeToPay + extraValue}(
                    address(l1weth),
                    recipient,
                    amount,
                    dataToCall,
                    gasLimit
                );
            } else {
                gateway.depositERC20AndCall{value: feeToPay + extraValue}(
                    address(l1weth),
                    recipient,
                    amount,
                    dataToCall,
                    gasLimit
                );
            }
            assertEq(amount + messengerBalance, address(l1Messenger).balance);
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertBoolEq(true, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
        }
    }
}
