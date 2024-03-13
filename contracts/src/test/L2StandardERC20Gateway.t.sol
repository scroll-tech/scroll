// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {MockERC20} from "solmate/test/utils/mocks/MockERC20.sol";

import {ITransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import {IL1ERC20Gateway, L1StandardERC20Gateway} from "../L1/gateways/L1StandardERC20Gateway.sol";
import {L2GatewayRouter} from "../L2/gateways/L2GatewayRouter.sol";
import {IL2ERC20Gateway, L2StandardERC20Gateway} from "../L2/gateways/L2StandardERC20Gateway.sol";
import {ScrollStandardERC20} from "../libraries/token/ScrollStandardERC20.sol";
import {ScrollStandardERC20Factory} from "../libraries/token/ScrollStandardERC20Factory.sol";

import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";

import {L2GatewayTestBase} from "./L2GatewayTestBase.t.sol";
import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {MockGatewayRecipient} from "./mocks/MockGatewayRecipient.sol";

contract L2StandardERC20GatewayTest is L2GatewayTestBase {
    // from L2StandardERC20Gateway
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

    ScrollStandardERC20 private template;
    ScrollStandardERC20Factory private factory;

    L2StandardERC20Gateway private gateway;
    L2GatewayRouter private router;

    L1StandardERC20Gateway private counterpartGateway;

    MockERC20 private badToken;
    MockERC20 private l1Token;
    MockERC20 private l2Token;

    function setUp() public {
        setUpBase();

        // Deploy tokens
        l1Token = new MockERC20("L1", "L1", 18);
        badToken = new MockERC20("Mock Bad", "M", 18);
        template = new ScrollStandardERC20();
        factory = new ScrollStandardERC20Factory(address(template));

        // Deploy L1 contracts
        counterpartGateway = new L1StandardERC20Gateway(
            address(1),
            address(1),
            address(1),
            address(template),
            address(factory)
        );

        // Deploy L2 contracts
        router = L2GatewayRouter(_deployProxy(address(new L2GatewayRouter())));
        gateway = _deployGateway(address(l2Messenger));

        // Initialize L2 contracts
        factory.transferOwnership(address(gateway));
        gateway.initialize(address(counterpartGateway), address(router), address(l2Messenger), address(factory));
        router.initialize(address(0), address(gateway));

        // Prepare token balances
        l2Token = MockERC20(gateway.getL2ERC20Address(address(l1Token)));
        hevm.startPrank(AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger)));
        l2Messenger.relayMessage(
            address(counterpartGateway),
            address(gateway),
            0,
            0,
            abi.encodeWithSelector(
                L2StandardERC20Gateway.finalizeDepositERC20.selector,
                address(l1Token),
                address(l2Token),
                address(this),
                address(this),
                type(uint128).max,
                abi.encode(true, abi.encode("", abi.encode("symbol", "name", 18)))
            )
        );
        hevm.stopPrank();
    }

    function testInitialized() public {
        assertEq(address(counterpartGateway), gateway.counterpart());
        assertEq(address(router), gateway.router());
        assertEq(address(l2Messenger), gateway.messenger());
        assertEq(address(factory), gateway.tokenFactory());
        assertEq(address(l1Token), gateway.getL1ERC20Address(address(l2Token)));

        hevm.expectRevert("Initializable: contract is already initialized");
        gateway.initialize(address(counterpartGateway), address(router), address(l1Messenger), address(factory));
    }

    function testGetL2ERC20Address(address l1Address) public {
        assertEq(gateway.getL2ERC20Address(l1Address), factory.computeL2TokenAddress(address(gateway), l1Address));
    }

    function testWithdrawERC20(
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawERC20(false, amount, gasLimit, feePerGas);
    }

    function testWithdrawERC20WithRecipient(
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawERC20WithRecipient(false, amount, recipient, gasLimit, feePerGas);
    }

    function testWithdrawERC20WithRecipientAndCalldata(
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawERC20WithRecipientAndCalldata(false, amount, recipient, dataToCall, gasLimit, feePerGas);
    }

    function testRouterDepositERC20(
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawERC20(true, amount, gasLimit, feePerGas);
    }

    function testRouterDepositERC20WithRecipient(
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawERC20WithRecipient(true, amount, recipient, gasLimit, feePerGas);
    }

    function testRouterDepositERC20WithRecipientAndCalldata(
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawERC20WithRecipientAndCalldata(true, amount, recipient, dataToCall, gasLimit, feePerGas);
    }

    function testFinalizeDepositERC20FailedMocking(
        address sender,
        address recipient,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        amount = bound(amount, 1, 100000);

        // revert when caller is not messenger
        hevm.expectRevert(ErrorCallerIsNotMessenger.selector);
        gateway.finalizeDepositERC20(address(l1Token), address(l2Token), sender, recipient, amount, dataToCall);

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway(address(mockMessenger));
        gateway.initialize(address(counterpartGateway), address(router), address(mockMessenger), address(factory));

        // only call by counterpart
        hevm.expectRevert(ErrorCallerIsNotCounterpartGateway.selector);
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeDepositERC20.selector,
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
                gateway.finalizeDepositERC20.selector,
                address(l1Token),
                address(l2Token),
                sender,
                recipient,
                amount,
                dataToCall
            )
        );

        // l1 token mismatch
        hevm.expectRevert("l2 token mismatch");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeDepositERC20.selector,
                address(l2Token),
                address(l2Token),
                sender,
                recipient,
                amount,
                dataToCall
            )
        );
    }

    function testFinalizeDepositERC20Failed(
        address sender,
        address recipient,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        // blacklist some addresses
        hevm.assume(recipient != address(0));

        amount = bound(amount, 1, l2Token.balanceOf(address(this)));

        // do finalize withdraw token
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
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

        // counterpart is not L2WETHGateway
        // emit FailedRelayedMessage from L1ScrollMessenger
        hevm.expectEmit(true, false, false, true);
        emit FailedRelayedMessage(keccak256(xDomainCalldata));

        uint256 gatewayBalance = l2Token.balanceOf(address(gateway));
        uint256 recipientBalance = l2Token.balanceOf(recipient);
        assertBoolEq(false, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata)));
        hevm.startPrank(AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger)));
        l2Messenger.relayMessage(address(uint160(address(counterpartGateway)) + 1), address(gateway), 0, 0, message);
        hevm.stopPrank();
        assertEq(gatewayBalance, l2Token.balanceOf(address(gateway)));
        assertEq(recipientBalance, l2Token.balanceOf(recipient));
        assertBoolEq(false, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testFinalizeDepositERC20(
        address sender,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        MockGatewayRecipient recipient = new MockGatewayRecipient();

        amount = bound(amount, 1, l2Token.balanceOf(address(this)));

        // do finalize withdraw token
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1Token),
            address(l2Token),
            sender,
            address(recipient),
            amount,
            abi.encode(false, dataToCall)
        );
        bytes memory xDomainCalldata = abi.encodeWithSignature(
            "relayMessage(address,address,uint256,uint256,bytes)",
            address(counterpartGateway),
            address(gateway),
            0,
            0,
            message
        );

        // emit FinalizeDepositERC20 from L2StandardERC20Gateway
        {
            hevm.expectEmit(true, true, true, true);
            emit FinalizeDepositERC20(
                address(l1Token),
                address(l2Token),
                sender,
                address(recipient),
                amount,
                dataToCall
            );
        }

        // emit RelayedMessage from L2ScrollMessenger
        {
            hevm.expectEmit(true, false, false, true);
            emit RelayedMessage(keccak256(xDomainCalldata));
        }

        uint256 gatewayBalance = l2Token.balanceOf(address(gateway));
        uint256 recipientBalance = l2Token.balanceOf(address(recipient));
        assertBoolEq(false, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata)));
        hevm.startPrank(AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger)));
        l2Messenger.relayMessage(address(counterpartGateway), address(gateway), 0, 0, message);
        hevm.stopPrank();
        assertEq(gatewayBalance, l2Token.balanceOf(address(gateway)));
        assertEq(recipientBalance + amount, l2Token.balanceOf(address(recipient)));
        assertBoolEq(true, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata)));
    }

    function _withdrawERC20(
        bool useRouter,
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, l2Token.balanceOf(address(this)));
        gasLimit = bound(gasLimit, 21000, 1000000);
        feePerGas = 0;

        setL1BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL1ERC20Gateway.finalizeWithdrawERC20.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            address(this),
            amount,
            new bytes(0)
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
            hevm.expectRevert("withdraw zero amount");
            if (useRouter) {
                router.withdrawERC20{value: feeToPay}(address(l2Token), amount, gasLimit);
            } else {
                gateway.withdrawERC20{value: feeToPay}(address(l2Token), amount, gasLimit);
            }
        } else {
            hevm.expectRevert("no corresponding l1 token");
            if (useRouter) {
                router.withdrawERC20{value: feeToPay}(address(l1Token), amount, gasLimit);
            } else {
                gateway.withdrawERC20{value: feeToPay}(address(l1Token), amount, gasLimit);
            }

            // emit AppendMessage from L2MessageQueue
            {
                hevm.expectEmit(false, false, false, true);
                emit AppendMessage(0, keccak256(xDomainCalldata));
            }

            // emit SentMessage from L2ScrollMessenger
            {
                hevm.expectEmit(true, true, false, true);
                emit SentMessage(address(gateway), address(counterpartGateway), 0, 0, gasLimit, message);
            }

            // emit WithdrawERC20 from L2StandardERC20Gateway
            hevm.expectEmit(true, true, true, true);
            emit WithdrawERC20(address(l1Token), address(l2Token), address(this), address(this), amount, new bytes(0));

            uint256 gatewayBalance = l2Token.balanceOf(address(gateway));
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.withdrawERC20{value: feeToPay}(address(l2Token), amount, gasLimit);
            } else {
                gateway.withdrawERC20{value: feeToPay}(address(l2Token), amount, gasLimit);
            }
            assertEq(gatewayBalance, l2Token.balanceOf(address(gateway)));
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertGt(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        }
    }

    function _withdrawERC20WithRecipient(
        bool useRouter,
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, l2Token.balanceOf(address(this)));
        gasLimit = bound(gasLimit, 21000, 1000000);
        feePerGas = 0;

        setL1BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL1ERC20Gateway.finalizeWithdrawERC20.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            recipient,
            amount,
            new bytes(0)
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
            hevm.expectRevert("withdraw zero amount");
            if (useRouter) {
                router.withdrawERC20{value: feeToPay}(address(l2Token), recipient, amount, gasLimit);
            } else {
                gateway.withdrawERC20{value: feeToPay}(address(l2Token), recipient, amount, gasLimit);
            }
        } else {
            hevm.expectRevert("no corresponding l1 token");
            if (useRouter) {
                router.withdrawERC20{value: feeToPay}(address(l1Token), recipient, amount, gasLimit);
            } else {
                gateway.withdrawERC20{value: feeToPay}(address(l1Token), recipient, amount, gasLimit);
            }

            // emit AppendMessage from L2MessageQueue
            {
                hevm.expectEmit(false, false, false, true);
                emit AppendMessage(0, keccak256(xDomainCalldata));
            }

            // emit SentMessage from L1ScrollMessenger
            {
                hevm.expectEmit(true, true, false, true);
                emit SentMessage(address(gateway), address(counterpartGateway), 0, 0, gasLimit, message);
            }

            // emit WithdrawERC20 from L1StandardERC20Gateway
            hevm.expectEmit(true, true, true, true);
            emit WithdrawERC20(address(l1Token), address(l2Token), address(this), recipient, amount, new bytes(0));

            uint256 gatewayBalance = l2Token.balanceOf(address(gateway));
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.withdrawERC20{value: feeToPay}(address(l2Token), recipient, amount, gasLimit);
            } else {
                gateway.withdrawERC20{value: feeToPay}(address(l2Token), recipient, amount, gasLimit);
            }
            assertEq(gatewayBalance, l2Token.balanceOf(address(gateway)));
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertGt(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        }
    }

    function _withdrawERC20WithRecipientAndCalldata(
        bool useRouter,
        uint256 amount,
        address recipient,
        bytes memory dataToCall,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, l2Token.balanceOf(address(this)));
        gasLimit = bound(gasLimit, 21000, 1000000);
        feePerGas = 0;

        setL1BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL1ERC20Gateway.finalizeWithdrawERC20.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            recipient,
            amount,
            dataToCall
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
            hevm.expectRevert("withdraw zero amount");
            if (useRouter) {
                router.withdrawERC20AndCall{value: feeToPay}(address(l2Token), recipient, amount, dataToCall, gasLimit);
            } else {
                gateway.withdrawERC20AndCall{value: feeToPay}(
                    address(l2Token),
                    recipient,
                    amount,
                    dataToCall,
                    gasLimit
                );
            }
        } else {
            hevm.expectRevert("no corresponding l1 token");
            if (useRouter) {
                router.withdrawERC20AndCall{value: feeToPay}(address(l1Token), recipient, amount, dataToCall, gasLimit);
            } else {
                gateway.withdrawERC20AndCall{value: feeToPay}(
                    address(l1Token),
                    recipient,
                    amount,
                    dataToCall,
                    gasLimit
                );
            }

            // emit AppendMessage from L2MessageQueue
            {
                hevm.expectEmit(false, false, false, true);
                emit AppendMessage(0, keccak256(xDomainCalldata));
            }

            // emit SentMessage from L1ScrollMessenger
            {
                hevm.expectEmit(true, true, false, true);
                emit SentMessage(address(gateway), address(counterpartGateway), 0, 0, gasLimit, message);
            }

            // emit WithdrawERC20 from L1StandardERC20Gateway
            hevm.expectEmit(true, true, true, true);
            emit WithdrawERC20(address(l1Token), address(l2Token), address(this), recipient, amount, dataToCall);

            uint256 gatewayBalance = l2Token.balanceOf(address(gateway));
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.withdrawERC20AndCall{value: feeToPay}(address(l2Token), recipient, amount, dataToCall, gasLimit);
            } else {
                gateway.withdrawERC20AndCall{value: feeToPay}(
                    address(l2Token),
                    recipient,
                    amount,
                    dataToCall,
                    gasLimit
                );
            }
            assertEq(gatewayBalance, l2Token.balanceOf(address(gateway)));
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertGt(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        }
    }

    function _deployGateway(address messenger) internal returns (L2StandardERC20Gateway _gateway) {
        _gateway = L2StandardERC20Gateway(_deployProxy(address(0)));

        admin.upgrade(
            ITransparentUpgradeableProxy(address(_gateway)),
            address(
                new L2StandardERC20Gateway(
                    address(counterpartGateway),
                    address(router),
                    address(messenger),
                    address(factory)
                )
            )
        );
    }
}
