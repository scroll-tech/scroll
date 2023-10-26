// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {MockERC20} from "solmate/test/utils/mocks/MockERC20.sol";

import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {IL1ERC20Gateway} from "../L1/gateways/IL1ERC20Gateway.sol";
import {L1GatewayRouter} from "../L1/gateways/L1GatewayRouter.sol";
import {IL1ScrollMessenger} from "../L1/IL1ScrollMessenger.sol";
import {IL2ERC20Gateway} from "../L2/gateways/IL2ERC20Gateway.sol";
import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";
import {ScrollConstants} from "../libraries/constants/ScrollConstants.sol";
import {L2LidoGateway} from "../lido/L2LidoGateway.sol";

import {L1GatewayTestBase} from "./L1GatewayTestBase.t.sol";
import {MockL1LidoGateway} from "./mocks/MockL1LidoGateway.sol";
import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {MockGatewayRecipient} from "./mocks/MockGatewayRecipient.sol";

contract L1LidoGatewayTest is L1GatewayTestBase {
    // events from L1LidoGateway
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
    event DepositsEnabled(address indexed enabler);
    event DepositsDisabled(address indexed disabler);
    event WithdrawalsEnabled(address indexed enabler);
    event WithdrawalsDisabled(address indexed disabler);
    event UpdateDepositsEnabler(address indexed oldEnabler, address indexed newEnabler);
    event UpdateDepositsDisabler(address indexed oldDisabler, address indexed newDisabler);
    event UpdateWithdrawalsEnabler(address indexed oldEnabler, address indexed newEnabler);
    event UpdateWithdrawalsDisabler(address indexed oldDisabler, address indexed newDisabler);

    // errors from L1LidoGateway
    error ErrorDepositsEnabled();
    error ErrorDepositsDisabled();
    error ErrorWithdrawalsEnabled();
    error ErrorWithdrawalsDisabled();
    error ErrorCallerIsNotDepositsEnabler();
    error ErrorCallerIsNotDepositsDisabler();
    error ErrorCallerIsNotWithdrawalsEnabler();
    error ErrorCallerIsNotWithdrawalsDisabler();
    error ErrorUnsupportedL1Token();
    error ErrorUnsupportedL2Token();
    error ErrorAccountIsZeroAddress();
    error ErrorNonZeroMsgValue();
    error ErrorDepositZeroAmount();

    MockL1LidoGateway private gateway;
    L1GatewayRouter private router;

    L2LidoGateway private counterpartGateway;

    MockERC20 private l1Token;
    MockERC20 private l2Token;

    function setUp() public {
        setUpBase();
        // Deploy tokens
        l1Token = new MockERC20("Mock L1", "ML1", 18);
        l2Token = new MockERC20("Mock L2", "ML2", 18);

        // Deploy L1 contracts
        gateway = _deployGateway();
        router = L1GatewayRouter(address(new ERC1967Proxy(address(new L1GatewayRouter()), new bytes(0))));

        // Deploy L2 contracts
        counterpartGateway = new L2LidoGateway(address(l1Token), address(l2Token));

        // Initialize L1 contracts
        gateway.initialize(address(counterpartGateway), address(router), address(l1Messenger));
        gateway.initializeV2();
        router.initialize(address(0), address(gateway));

        // Prepare token balances
        l1Token.mint(address(this), type(uint128).max);
        l1Token.approve(address(gateway), type(uint256).max);
        l1Token.approve(address(router), type(uint256).max);
    }

    function testInitialized() public {
        // state in ScrollGatewayBase
        assertEq(address(this), gateway.owner());
        assertEq(address(counterpartGateway), gateway.counterpart());
        assertEq(address(router), gateway.router());
        assertEq(address(l1Messenger), gateway.messenger());

        // state in LidoBridgeableTokens
        assertEq(address(l1Token), gateway.l1Token());
        assertEq(address(l2Token), gateway.l2Token());

        // state in LidoGatewayManager
        assertBoolEq(true, gateway.isDepositsEnabled());
        assertBoolEq(true, gateway.isWithdrawalsEnabled());

        // state in L1LidoGateway
        assertEq(address(l2Token), gateway.getL2ERC20Address(address(l1Token)));

        hevm.expectRevert("Initializable: contract is already initialized");
        gateway.initialize(address(counterpartGateway), address(router), address(l1Messenger));

        hevm.expectRevert("Initializable: contract is already initialized");
        gateway.initializeV2();
    }

    /*************************************
     * Functions from LidoGatewayManager *
     *************************************/

    function testEnableDeposits() external {
        // revert when already enabled
        hevm.expectRevert(ErrorDepositsEnabled.selector);
        gateway.enableDeposits();

        // revert when caller is not deposits enabler
        gateway.updateDepositsDisabler(address(this));
        gateway.disableDeposits();
        hevm.expectRevert(ErrorCallerIsNotDepositsEnabler.selector);
        gateway.enableDeposits();

        // succeed
        gateway.updateDepositsEnabler(address(this));
        assertBoolEq(false, gateway.isDepositsEnabled());
        hevm.expectEmit(true, false, false, true);
        emit DepositsEnabled(address(this));
        gateway.enableDeposits();
        assertBoolEq(true, gateway.isDepositsEnabled());
    }

    function testDisableDeposits() external {
        // revert when already disabled
        gateway.updateDepositsDisabler(address(this));
        gateway.disableDeposits();
        assertBoolEq(false, gateway.isDepositsEnabled());
        hevm.expectRevert(ErrorDepositsDisabled.selector);
        gateway.disableDeposits();

        // revert when caller is not deposits disabler
        gateway.updateDepositsEnabler(address(this));
        gateway.enableDeposits();
        assertBoolEq(true, gateway.isDepositsEnabled());
        gateway.updateDepositsDisabler(address(0));
        hevm.expectRevert(ErrorCallerIsNotDepositsDisabler.selector);
        gateway.disableDeposits();

        // succeed
        gateway.updateDepositsDisabler(address(this));
        assertBoolEq(true, gateway.isDepositsEnabled());
        hevm.expectEmit(true, false, false, true);
        emit DepositsDisabled(address(this));
        gateway.disableDeposits();
        assertBoolEq(false, gateway.isDepositsEnabled());
    }

    function testEnableWithdrawals() external {
        // revert when already enabled
        hevm.expectRevert(ErrorWithdrawalsEnabled.selector);
        gateway.enableWithdrawals();

        // revert when caller is not deposits enabler
        gateway.updateWithdrawalsDisabler(address(this));
        gateway.disableWithdrawals();
        hevm.expectRevert(ErrorCallerIsNotWithdrawalsEnabler.selector);
        gateway.enableWithdrawals();

        // succeed
        gateway.updateWithdrawalsEnabler(address(this));
        assertBoolEq(false, gateway.isWithdrawalsEnabled());
        hevm.expectEmit(true, false, false, true);
        emit WithdrawalsEnabled(address(this));
        gateway.enableWithdrawals();
        assertBoolEq(true, gateway.isWithdrawalsEnabled());
    }

    function testDisableWithdrawals() external {
        // revert when already disabled
        gateway.updateWithdrawalsDisabler(address(this));
        gateway.disableWithdrawals();
        assertBoolEq(false, gateway.isWithdrawalsEnabled());
        hevm.expectRevert(ErrorWithdrawalsDisabled.selector);
        gateway.disableWithdrawals();

        // revert when caller is not deposits disabler
        gateway.updateWithdrawalsEnabler(address(this));
        gateway.enableWithdrawals();
        assertBoolEq(true, gateway.isWithdrawalsEnabled());
        gateway.updateWithdrawalsDisabler(address(0));
        hevm.expectRevert(ErrorCallerIsNotWithdrawalsDisabler.selector);
        gateway.disableWithdrawals();

        // succeed
        gateway.updateWithdrawalsDisabler(address(this));
        assertBoolEq(true, gateway.isWithdrawalsEnabled());
        hevm.expectEmit(true, false, false, true);
        emit WithdrawalsDisabled(address(this));
        gateway.disableWithdrawals();
        assertBoolEq(false, gateway.isWithdrawalsEnabled());
    }

    function testUpdateDepositsEnabler(address _enabler) external {
        hevm.assume(_enabler != address(0));

        // revert caller is not owner
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.updateDepositsEnabler(address(0));
        hevm.stopPrank();

        gateway.updateDepositsDisabler(address(this));
        gateway.disableDeposits();

        // succeed to set
        hevm.startPrank(_enabler);
        hevm.expectRevert(ErrorCallerIsNotDepositsEnabler.selector);
        gateway.enableDeposits();
        hevm.stopPrank();

        hevm.expectEmit(true, true, false, true);
        emit UpdateDepositsEnabler(address(0), _enabler);
        gateway.updateDepositsEnabler(_enabler);

        assertBoolEq(false, gateway.isDepositsEnabled());
        hevm.startPrank(_enabler);
        gateway.enableDeposits();
        hevm.stopPrank();
        assertBoolEq(true, gateway.isDepositsEnabled());
    }

    function testUpdateDepositsDisabler(address _disabler) external {
        hevm.assume(_disabler != address(0));

        // revert caller is not owner
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.updateDepositsDisabler(address(0));
        hevm.stopPrank();

        // succeed to set
        hevm.startPrank(_disabler);
        hevm.expectRevert(ErrorCallerIsNotDepositsDisabler.selector);
        gateway.disableDeposits();
        hevm.stopPrank();

        hevm.expectEmit(true, true, false, true);
        emit UpdateDepositsDisabler(address(0), _disabler);
        gateway.updateDepositsDisabler(_disabler);

        assertBoolEq(true, gateway.isDepositsEnabled());
        hevm.startPrank(_disabler);
        gateway.disableDeposits();
        hevm.stopPrank();
        assertBoolEq(false, gateway.isDepositsEnabled());
    }

    function testUpdateWithdrawalsEnabler(address _enabler) external {
        hevm.assume(_enabler != address(0));

        // revert caller is not owner
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.updateWithdrawalsEnabler(address(0));
        hevm.stopPrank();

        gateway.updateWithdrawalsDisabler(address(this));
        gateway.disableWithdrawals();

        // succeed to set
        hevm.startPrank(_enabler);
        hevm.expectRevert(ErrorCallerIsNotWithdrawalsEnabler.selector);
        gateway.enableWithdrawals();
        hevm.stopPrank();

        hevm.expectEmit(true, true, false, true);
        emit UpdateWithdrawalsEnabler(address(0), _enabler);
        gateway.updateWithdrawalsEnabler(_enabler);

        assertBoolEq(false, gateway.isWithdrawalsEnabled());
        hevm.startPrank(_enabler);
        gateway.enableWithdrawals();
        hevm.stopPrank();
        assertBoolEq(true, gateway.isWithdrawalsEnabled());
    }

    function testUpdateWithdrawalsDisabler(address _disabler) external {
        hevm.assume(_disabler != address(0));

        // revert caller is not owner
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.updateWithdrawalsDisabler(address(0));
        hevm.stopPrank();

        // succeed to set
        hevm.startPrank(_disabler);
        hevm.expectRevert(ErrorCallerIsNotWithdrawalsDisabler.selector);
        gateway.disableWithdrawals();
        hevm.stopPrank();

        hevm.expectEmit(true, true, false, true);
        emit UpdateWithdrawalsDisabler(address(0), _disabler);
        gateway.updateWithdrawalsDisabler(_disabler);

        assertBoolEq(true, gateway.isWithdrawalsEnabled());
        hevm.startPrank(_disabler);
        gateway.disableWithdrawals();
        hevm.stopPrank();
        assertBoolEq(false, gateway.isWithdrawalsEnabled());
    }

    /********************************
     * Functions from L1LidoGateway *
     ********************************/

    function testDepositERC20(
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) external {
        _depositERC20(false, 0, amount, address(this), new bytes(0), gasLimit, feePerGas);
    }

    function testDepositERC20WithRecipient(
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) external {
        _depositERC20(false, 1, amount, recipient, new bytes(0), gasLimit, feePerGas);
    }

    function testDepositERC20WithRecipientAndCalldata(
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) external {
        _depositERC20(false, 2, amount, recipient, dataToCall, gasLimit, feePerGas);
    }

    function testDepositERC20ByRouter(
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) external {
        _depositERC20(true, 0, amount, address(this), new bytes(0), gasLimit, feePerGas);
    }

    function testDepositERC20WithRecipientByRouter(
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) external {
        _depositERC20(true, 1, amount, recipient, new bytes(0), gasLimit, feePerGas);
    }

    function testDepositERC20WithRecipientAndCalldataByRouter(
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) external {
        _depositERC20(true, 2, amount, recipient, dataToCall, gasLimit, feePerGas);
    }

    function testDropMessage(
        uint256 amount,
        address recipient,
        bytes memory dataToCall
    ) public {
        hevm.assume(recipient != address(0));

        amount = bound(amount, 1, l1Token.balanceOf(address(this)));
        bytes memory message = abi.encodeCall(
            IL2ERC20Gateway.finalizeDepositERC20,
            (address(l1Token), address(l2Token), address(this), recipient, amount, dataToCall)
        );
        gateway.depositERC20AndCall(address(l1Token), recipient, amount, dataToCall, defaultGasLimit);

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        MockL1LidoGateway mockGateway = _deployGateway();
        mockGateway.initialize(address(counterpartGateway), address(router), address(mockMessenger));
        mockGateway.initializeV2();

        // revert caller is not messenger
        hevm.expectRevert("only messenger can call");
        mockGateway.onDropMessage(new bytes(0));

        // revert not in drop context
        hevm.expectRevert("only called in drop context");
        mockMessenger.callTarget(address(mockGateway), abi.encodeCall(mockGateway.onDropMessage, (new bytes(0))));

        // revert when reentrant
        mockMessenger.setXDomainMessageSender(ScrollConstants.DROP_XDOMAIN_MESSAGE_SENDER);
        hevm.expectRevert("ReentrancyGuard: reentrant call");
        mockGateway.reentrantCall(
            address(mockMessenger),
            abi.encodeCall(
                mockMessenger.callTarget,
                (address(mockGateway), abi.encodeCall(mockGateway.onDropMessage, (message)))
            )
        );

        // revert when invalid selector
        hevm.expectRevert("invalid selector");
        mockMessenger.callTarget(address(mockGateway), abi.encodeCall(mockGateway.onDropMessage, (new bytes(4))));

        // revert when l1 token not supported
        hevm.expectRevert(ErrorUnsupportedL1Token.selector);
        mockMessenger.callTarget(
            address(mockGateway),
            abi.encodeCall(
                mockGateway.onDropMessage,
                (
                    abi.encodeCall(
                        IL2ERC20Gateway.finalizeDepositERC20,
                        (address(l2Token), address(l2Token), address(this), recipient, amount, dataToCall)
                    )
                )
            )
        );

        // revert when nonzero msg.value
        hevm.expectRevert(ErrorNonZeroMsgValue.selector);
        mockMessenger.callTarget{value: 1}(
            address(mockGateway),
            abi.encodeWithSelector(mockGateway.onDropMessage.selector, message)
        );

        // succeed on drop
        // skip message 0
        hevm.startPrank(address(rollup));
        messageQueue.popCrossDomainMessage(0, 1, 0x1);
        assertEq(messageQueue.pendingQueueIndex(), 1);
        hevm.stopPrank();

        // should emit RefundERC20
        hevm.expectEmit(true, true, false, true);
        emit RefundERC20(address(l1Token), address(this), amount);

        uint256 balance = l1Token.balanceOf(address(this));
        uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
        l1Messenger.dropMessage(address(gateway), address(counterpartGateway), 0, 0, message);
        assertEq(gatewayBalance - amount, l1Token.balanceOf(address(gateway)));
        assertEq(balance + amount, l1Token.balanceOf(address(this)));
    }

    function testFinalizeWithdrawERC20(
        address sender,
        uint256 amount,
        bytes memory dataToCall
    ) external {
        amount = bound(amount, 1, l1Token.balanceOf(address(this)));
        MockGatewayRecipient recipient = new MockGatewayRecipient();
        bytes memory message = abi.encodeCall(
            IL1ERC20Gateway.finalizeWithdrawERC20,
            (address(l1Token), address(l2Token), sender, address(recipient), amount, dataToCall)
        );
        gateway.depositERC20(address(l1Token), amount, defaultGasLimit); // deposit some token to L1LidoGateway

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        MockL1LidoGateway mockGateway = _deployGateway();
        mockGateway.initialize(address(counterpartGateway), address(router), address(mockMessenger));
        mockGateway.initializeV2();

        // revert caller is not messenger
        hevm.expectRevert("only messenger can call");
        mockGateway.finalizeWithdrawERC20(
            address(l1Token),
            address(l2Token),
            sender,
            address(recipient),
            amount,
            dataToCall
        );

        // revert not called by counterpart
        hevm.expectRevert("only call by counterpart");
        mockMessenger.callTarget(address(mockGateway), message);

        // revert when reentrant
        mockMessenger.setXDomainMessageSender(address(counterpartGateway));
        hevm.expectRevert("ReentrancyGuard: reentrant call");
        mockGateway.reentrantCall(
            address(mockMessenger),
            abi.encodeCall(mockMessenger.callTarget, (address(mockGateway), message))
        );

        // revert when l1 token not supported
        hevm.expectRevert(ErrorUnsupportedL1Token.selector);
        mockMessenger.callTarget(
            address(mockGateway),
            abi.encodeCall(
                IL1ERC20Gateway.finalizeWithdrawERC20,
                (address(l2Token), address(l2Token), sender, address(recipient), amount, dataToCall)
            )
        );

        // revert when l2 token not supported
        hevm.expectRevert(ErrorUnsupportedL2Token.selector);
        mockMessenger.callTarget(
            address(mockGateway),
            abi.encodeCall(
                IL1ERC20Gateway.finalizeWithdrawERC20,
                (address(l1Token), address(l1Token), sender, address(recipient), amount, dataToCall)
            )
        );

        // revert when withdrawals disabled
        mockGateway.updateWithdrawalsDisabler(address(this));
        mockGateway.disableWithdrawals();
        hevm.expectRevert(ErrorWithdrawalsDisabled.selector);
        mockMessenger.callTarget(address(mockGateway), message);

        // revert when nonzero msg.value
        mockGateway.updateWithdrawalsEnabler(address(this));
        mockGateway.enableWithdrawals();
        hevm.expectRevert(ErrorNonZeroMsgValue.selector);
        mockMessenger.callTarget{value: 1}(address(mockGateway), message);

        // succeed when finialize
        bytes memory xDomainCalldata = abi.encodeCall(
            l2Messenger.relayMessage,
            (address(counterpartGateway), address(gateway), 0, 0, message)
        );
        prepareL2MessageRoot(keccak256(xDomainCalldata));
        IL1ScrollMessenger.L2MessageProof memory proof;
        proof.batchIndex = rollup.lastFinalizedBatchIndex();

        // should emit FinalizeWithdrawERC20 from L1StandardERC20Gateway
        hevm.expectEmit(true, true, true, true);
        emit FinalizeWithdrawERC20(address(l1Token), address(l2Token), sender, address(recipient), amount, dataToCall);

        // should emit RelayedMessage from L1ScrollMessenger
        hevm.expectEmit(true, false, false, true);
        emit RelayedMessage(keccak256(xDomainCalldata));

        uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
        uint256 recipientBalance = l1Token.balanceOf(address(recipient));
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(address(counterpartGateway), address(gateway), 0, 0, message, proof);
        assertBoolEq(true, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        assertEq(recipientBalance + amount, l1Token.balanceOf(address(recipient)));
        assertEq(gatewayBalance - amount, l1Token.balanceOf(address(gateway)));
    }

    function _depositERC20(
        bool useRouter,
        uint256 methodType,
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        hevm.assume(recipient != address(0));
        amount = bound(amount, 1, l1Token.balanceOf(address(this)));
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);
        gasOracle.setL2BaseFee(feePerGas);
        feePerGas = feePerGas * gasLimit;

        // revert when reentrant
        hevm.expectRevert("ReentrancyGuard: reentrant call");
        {
            bytes memory reentrantData;
            if (methodType == 0) {
                reentrantData = abi.encodeWithSignature(
                    "depositERC20(address,uint256,uint256)",
                    address(l1Token),
                    amount,
                    gasLimit
                );
            } else if (methodType == 1) {
                reentrantData = abi.encodeWithSignature(
                    "depositERC20(address,address,uint256,uint256)",
                    address(l1Token),
                    recipient,
                    amount,
                    gasLimit
                );
            } else if (methodType == 2) {
                reentrantData = abi.encodeCall(
                    IL1ERC20Gateway.depositERC20AndCall,
                    (address(l1Token), recipient, amount, dataToCall, gasLimit)
                );
            }
            gateway.reentrantCall(useRouter ? address(router) : address(gateway), reentrantData);
        }

        // revert when l1 token not support
        hevm.expectRevert(ErrorUnsupportedL1Token.selector);
        _invokeDepositERC20Call(
            useRouter,
            methodType,
            address(l2Token),
            amount,
            recipient,
            dataToCall,
            gasLimit,
            feePerGas
        );

        // revert when to is zero address
        if (methodType != 0) {
            hevm.expectRevert(ErrorAccountIsZeroAddress.selector);
            _invokeDepositERC20Call(
                useRouter,
                methodType,
                address(l1Token),
                amount,
                address(0),
                dataToCall,
                gasLimit,
                feePerGas
            );
        }

        // revert when deposits disabled
        gateway.updateDepositsDisabler(address(this));
        gateway.disableDeposits();
        hevm.expectRevert(ErrorDepositsDisabled.selector);
        _invokeDepositERC20Call(
            useRouter,
            methodType,
            address(l1Token),
            amount,
            recipient,
            dataToCall,
            gasLimit,
            feePerGas
        );

        // revert when deposit zero amount
        gateway.updateDepositsEnabler(address(this));
        gateway.enableDeposits();
        hevm.expectRevert(ErrorDepositZeroAmount.selector);
        _invokeDepositERC20Call(useRouter, methodType, address(l1Token), 0, recipient, dataToCall, gasLimit, feePerGas);

        // succeed to withdraw
        bytes memory message = abi.encodeCall(
            IL2ERC20Gateway.finalizeDepositERC20,
            (address(l1Token), address(l2Token), address(this), recipient, amount, dataToCall)
        );
        bytes memory xDomainCalldata = abi.encodeCall(
            l2Messenger.relayMessage,
            (address(gateway), address(counterpartGateway), 0, 0, message)
        );
        // should emit QueueTransaction from L1MessageQueue
        hevm.expectEmit(true, true, false, true);
        address sender = AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger));
        emit QueueTransaction(sender, address(l2Messenger), 0, 0, gasLimit, xDomainCalldata);

        // should emit SentMessage from L1ScrollMessenger
        hevm.expectEmit(true, true, false, true);
        emit SentMessage(address(gateway), address(counterpartGateway), 0, 0, gasLimit, message);

        // should emit DepositERC20 from L1CustomERC20Gateway
        hevm.expectEmit(true, true, true, true);
        emit DepositERC20(address(l1Token), address(l2Token), address(this), recipient, amount, dataToCall);

        uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
        uint256 feeVaultBalance = address(feeVault).balance;
        uint256 thisBalance = l1Token.balanceOf(address(this));
        assertEq(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        uint256 balance = address(this).balance;
        _invokeDepositERC20Call(
            useRouter,
            methodType,
            address(l1Token),
            amount,
            recipient,
            dataToCall,
            gasLimit,
            feePerGas
        );
        assertEq(balance - feePerGas, address(this).balance); // extra value is transfered back
        assertGt(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        assertEq(thisBalance - amount, l1Token.balanceOf(address(this)));
        assertEq(feeVaultBalance + feePerGas, address(feeVault).balance);
        assertEq(gatewayBalance + amount, l1Token.balanceOf(address(gateway)));
    }

    function _invokeDepositERC20Call(
        bool useRouter,
        uint256 methodType,
        address token,
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feeToPay
    ) private {
        uint256 value = feeToPay + extraValue;
        if (useRouter) {
            if (methodType == 0) {
                router.depositERC20{value: value}(token, amount, gasLimit);
            } else if (methodType == 1) {
                router.depositERC20{value: value}(token, recipient, amount, gasLimit);
            } else if (methodType == 2) {
                router.depositERC20AndCall{value: value}(token, recipient, amount, dataToCall, gasLimit);
            }
        } else {
            if (methodType == 0) {
                gateway.depositERC20{value: value}(token, amount, gasLimit);
            } else if (methodType == 1) {
                gateway.depositERC20{value: value}(token, recipient, amount, gasLimit);
            } else if (methodType == 2) {
                gateway.depositERC20AndCall{value: value}(token, recipient, amount, dataToCall, gasLimit);
            }
        }
    }

    function _deployGateway() internal returns (MockL1LidoGateway) {
        return
            MockL1LidoGateway(
                address(
                    new ERC1967Proxy(address(new MockL1LidoGateway(address(l1Token), address(l2Token))), new bytes(0))
                )
            );
    }
}
