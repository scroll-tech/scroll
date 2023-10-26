// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {MockERC20} from "solmate/test/utils/mocks/MockERC20.sol";

import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {IL1ERC20Gateway} from "../L1/gateways/IL1ERC20Gateway.sol";
import {IL2ERC20Gateway} from "../L2/gateways/IL2ERC20Gateway.sol";
import {L2GatewayRouter} from "../L2/gateways/L2GatewayRouter.sol";
import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";
import {ScrollStandardERC20} from "../libraries/token/ScrollStandardERC20.sol";
import {L1LidoGateway} from "../lido/L1LidoGateway.sol";

import {L2GatewayTestBase} from "./L2GatewayTestBase.t.sol";
import {MockGatewayRecipient} from "./mocks/MockGatewayRecipient.sol";
import {MockL2LidoGateway} from "./mocks/MockL2LidoGateway.sol";
import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";

contract L2LidoGatewayTest is L2GatewayTestBase {
    // events from L2LidoGateway
    event WithdrawERC20(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );
    event FinalizeDepositERC20(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );
    event DepositsEnabled(address indexed enabler);
    event DepositsDisabled(address indexed disabler);
    event WithdrawalsEnabled(address indexed enabler);
    event WithdrawalsDisabled(address indexed disabler);
    event UpdateDepositsEnabler(address indexed oldEnabler, address indexed newEnabler);
    event UpdateDepositsDisabler(address indexed oldDisabler, address indexed newDisabler);
    event UpdateWithdrawalsEnabler(address indexed oldEnabler, address indexed newEnabler);
    event UpdateWithdrawalsDisabler(address indexed oldDisabler, address indexed newDisabler);

    // errors from L2LidoGateway
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
    error ErrorWithdrawZeroAmount();

    MockL2LidoGateway private gateway;
    L2GatewayRouter private router;

    L1LidoGateway private counterpartGateway;

    MockERC20 private l1Token;
    ScrollStandardERC20 private l2Token;

    function setUp() public {
        setUpBase();
        // Deploy tokens
        l1Token = new MockERC20("Mock L1", "ML1", 18);
        l2Token = ScrollStandardERC20(address(new ERC1967Proxy(address(new ScrollStandardERC20()), new bytes(0))));

        // Deploy L2 contracts
        gateway = _deployGateway();
        router = L2GatewayRouter(address(new ERC1967Proxy(address(new L2GatewayRouter()), new bytes(0))));

        // Deploy L1 contracts
        counterpartGateway = new L1LidoGateway(address(l1Token), address(l2Token));

        // Initialize L2 contracts
        gateway.initialize(address(counterpartGateway), address(router), address(l2Messenger));
        gateway.initializeV2();
        router.initialize(address(0), address(gateway));
        l2Token.initialize("Mock L2", "ML2", 18, address(gateway), address(l1Token));

        // Prepare token balances
        hevm.startPrank(address(gateway));
        l2Token.mint(address(this), type(uint128).max);
        hevm.stopPrank();
    }

    function testInitialized() external {
        // state in ScrollGatewayBase
        assertEq(address(this), gateway.owner());
        assertEq(address(counterpartGateway), gateway.counterpart());
        assertEq(address(router), gateway.router());
        assertEq(address(l2Messenger), gateway.messenger());

        // state in LidoBridgeableTokens
        assertEq(address(l1Token), gateway.l1Token());
        assertEq(address(l2Token), gateway.l2Token());

        // state in LidoGatewayManager
        assertBoolEq(true, gateway.isDepositsEnabled());
        assertBoolEq(true, gateway.isWithdrawalsEnabled());

        // state in L2LidoGateway
        assertEq(address(l1Token), gateway.getL1ERC20Address(address(l2Token)));
        assertEq(address(l2Token), gateway.getL2ERC20Address(address(l1Token)));

        hevm.expectRevert("Initializable: contract is already initialized");
        gateway.initialize(address(counterpartGateway), address(router), address(l2Messenger));

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
     * Functions from L2LidoGateway *
     ********************************/

    function testGetL1ERC20Address(address token) external {
        hevm.assume(token != address(l2Token));
        hevm.expectRevert(ErrorUnsupportedL2Token.selector);
        gateway.getL1ERC20Address(token);
    }

    function testGetL2ERC20Address(address token) external {
        hevm.assume(token != address(l1Token));
        hevm.expectRevert(ErrorUnsupportedL1Token.selector);
        gateway.getL2ERC20Address(token);
    }

    function testWithdrawERC20(uint256 amount, uint256 gasLimit) external {
        _withdrawERC20(false, 0, amount, address(this), new bytes(0), gasLimit);
    }

    function testWithdrawERC20WithRecipient(
        uint256 amount,
        address recipient,
        uint256 gasLimit
    ) external {
        _withdrawERC20(false, 1, amount, recipient, new bytes(0), gasLimit);
    }

    function testWithdrawERC20WithRecipientAndCalldata(
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit
    ) external {
        _withdrawERC20(false, 2, amount, recipient, dataToCall, gasLimit);
    }

    function testWithdrawERC20ByRouter(uint256 amount, uint256 gasLimit) external {
        _withdrawERC20(true, 0, amount, address(this), new bytes(0), gasLimit);
    }

    function testWithdrawERC20WithRecipientByRouter(
        uint256 amount,
        address recipient,
        uint256 gasLimit
    ) external {
        _withdrawERC20(true, 1, amount, recipient, new bytes(0), gasLimit);
    }

    function testWithdrawERC20WithRecipientAndCalldataByRouter(
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit
    ) external {
        _withdrawERC20(true, 2, amount, recipient, dataToCall, gasLimit);
    }

    function testFinalizeDepositERC20(
        address sender,
        uint256 amount,
        bytes memory dataToCall
    ) external {
        amount = bound(amount, 1, l2Token.balanceOf(address(this)));
        MockGatewayRecipient recipient = new MockGatewayRecipient();
        bytes memory message = abi.encodeCall(
            IL2ERC20Gateway.finalizeDepositERC20,
            (address(l1Token), address(l2Token), sender, address(recipient), amount, dataToCall)
        );

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        MockL2LidoGateway mockGateway = _deployGateway();
        mockGateway.initialize(address(counterpartGateway), address(router), address(mockMessenger));
        mockGateway.initializeV2();

        // revert caller is not messenger
        hevm.expectRevert("only messenger can call");
        mockGateway.finalizeDepositERC20(
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
                IL2ERC20Gateway.finalizeDepositERC20,
                (address(l2Token), address(l2Token), sender, address(recipient), amount, dataToCall)
            )
        );

        // revert when l2 token not supported
        hevm.expectRevert(ErrorUnsupportedL2Token.selector);
        mockMessenger.callTarget(
            address(mockGateway),
            abi.encodeCall(
                IL2ERC20Gateway.finalizeDepositERC20,
                (address(l1Token), address(l1Token), sender, address(recipient), amount, dataToCall)
            )
        );

        // revert when deposits disabled
        mockGateway.updateDepositsDisabler(address(this));
        mockGateway.disableDeposits();
        hevm.expectRevert(ErrorDepositsDisabled.selector);
        mockMessenger.callTarget(address(mockGateway), message);

        // revert when nonzero msg.value
        mockGateway.updateDepositsEnabler(address(this));
        mockGateway.enableDeposits();
        hevm.expectRevert(ErrorNonZeroMsgValue.selector);
        mockMessenger.callTarget{value: 1}(address(mockGateway), message);

        // succeed when finialize
        bytes memory xDomainCalldata = abi.encodeCall(
            l2Messenger.relayMessage,
            (address(counterpartGateway), address(gateway), 0, 0, message)
        );

        // should emit FinalizeDepositERC20 from L2LidoGateway
        hevm.expectEmit(true, true, true, true);
        emit FinalizeDepositERC20(address(l1Token), address(l2Token), sender, address(recipient), amount, dataToCall);

        // should emit RelayedMessage from L2ScrollMessenger
        hevm.expectEmit(true, false, false, true);
        emit RelayedMessage(keccak256(xDomainCalldata));

        uint256 gatewayBalance = l2Token.balanceOf(address(gateway));
        uint256 recipientBalance = l2Token.balanceOf(address(recipient));
        assertBoolEq(false, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata)));
        hevm.startPrank(AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger)));
        l2Messenger.relayMessage(address(counterpartGateway), address(gateway), 0, 0, message);
        hevm.stopPrank();
        assertBoolEq(true, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata))); // executed
        assertEq(recipientBalance + amount, l2Token.balanceOf(address(recipient))); // mint token
        assertEq(gatewayBalance, l2Token.balanceOf(address(gateway))); // gateway balance unchanged
    }

    function _withdrawERC20(
        bool useRouter,
        uint256 methodType,
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit
    ) private {
        hevm.assume(recipient != address(0));
        amount = bound(amount, 1, l2Token.balanceOf(address(this)));

        // revert when reentrant
        hevm.expectRevert("ReentrancyGuard: reentrant call");
        bytes memory reentrantData;
        if (methodType == 0) {
            reentrantData = abi.encodeWithSignature(
                "withdrawERC20(address,uint256,uint256)",
                address(l2Token),
                amount,
                gasLimit
            );
        } else if (methodType == 1) {
            reentrantData = abi.encodeWithSignature(
                "withdrawERC20(address,address,uint256,uint256)",
                address(l2Token),
                recipient,
                amount,
                gasLimit
            );
        } else if (methodType == 2) {
            reentrantData = abi.encodeCall(
                IL2ERC20Gateway.withdrawERC20AndCall,
                (address(l2Token), recipient, amount, dataToCall, gasLimit)
            );
        }
        gateway.reentrantCall(useRouter ? address(router) : address(gateway), reentrantData);

        // revert when l2 token not support
        hevm.expectRevert(ErrorUnsupportedL2Token.selector);
        _invokeWithdrawERC20Call(useRouter, methodType, address(l1Token), amount, recipient, dataToCall, gasLimit);

        // revert when to is zero address
        if (methodType != 0) {
            hevm.expectRevert(ErrorAccountIsZeroAddress.selector);
            _invokeWithdrawERC20Call(useRouter, methodType, address(l2Token), amount, address(0), dataToCall, gasLimit);
        }

        // revert when withdrawals disabled
        gateway.updateWithdrawalsDisabler(address(this));
        gateway.disableWithdrawals();
        hevm.expectRevert(ErrorWithdrawalsDisabled.selector);
        _invokeWithdrawERC20Call(useRouter, methodType, address(l2Token), amount, recipient, dataToCall, gasLimit);

        // revert when withdraw zero amount
        gateway.updateWithdrawalsEnabler(address(this));
        gateway.enableWithdrawals();
        hevm.expectRevert(ErrorWithdrawZeroAmount.selector);
        _invokeWithdrawERC20Call(useRouter, methodType, address(l2Token), 0, recipient, dataToCall, gasLimit);

        // succeed to withdraw
        bytes memory message = abi.encodeCall(
            IL1ERC20Gateway.finalizeWithdrawERC20,
            (address(l1Token), address(l2Token), address(this), recipient, amount, dataToCall)
        );
        bytes memory xDomainCalldata = abi.encodeCall(
            l2Messenger.relayMessage,
            (address(gateway), address(counterpartGateway), 0, 0, message)
        );
        // should emit AppendMessage from L2MessageQueue
        hevm.expectEmit(false, false, false, true);
        emit AppendMessage(0, keccak256(xDomainCalldata));

        // should emit SentMessage from L2ScrollMessenger
        hevm.expectEmit(true, true, false, true);
        emit SentMessage(address(gateway), address(counterpartGateway), 0, 0, gasLimit, message);

        // should emit WithdrawERC20 from L2LidoGateway
        hevm.expectEmit(true, true, true, true);
        emit WithdrawERC20(address(l1Token), address(l2Token), address(this), recipient, amount, dataToCall);

        uint256 gatewayBalance = l2Token.balanceOf(address(gateway));
        uint256 thisBalance = l2Token.balanceOf(address(this));
        assertEq(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        _invokeWithdrawERC20Call(useRouter, methodType, address(l2Token), amount, recipient, dataToCall, gasLimit);
        assertGt(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        assertEq(thisBalance - amount, l2Token.balanceOf(address(this)));
        assertEq(gatewayBalance, l2Token.balanceOf(address(gateway)));
    }

    function _invokeWithdrawERC20Call(
        bool useRouter,
        uint256 methodType,
        address token,
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit
    ) private {
        if (useRouter) {
            if (methodType == 0) {
                router.withdrawERC20(token, amount, gasLimit);
            } else if (methodType == 1) {
                router.withdrawERC20(token, recipient, amount, gasLimit);
            } else if (methodType == 2) {
                router.withdrawERC20AndCall(token, recipient, amount, dataToCall, gasLimit);
            }
        } else {
            if (methodType == 0) {
                gateway.withdrawERC20(token, amount, gasLimit);
            } else if (methodType == 1) {
                gateway.withdrawERC20(token, recipient, amount, gasLimit);
            } else if (methodType == 2) {
                gateway.withdrawERC20AndCall(token, recipient, amount, dataToCall, gasLimit);
            }
        }
    }

    function _deployGateway() internal returns (MockL2LidoGateway) {
        return
            MockL2LidoGateway(
                address(
                    new ERC1967Proxy(address(new MockL2LidoGateway(address(l1Token), address(l2Token))), new bytes(0))
                )
            );
    }
}
