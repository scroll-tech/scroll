// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {ITransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";

import {L1USDCGateway} from "../L1/gateways/usdc/L1USDCGateway.sol";
import {IL1ERC20Gateway} from "../L1/gateways/IL1ERC20Gateway.sol";
import {L2GatewayRouter} from "../L2/gateways/L2GatewayRouter.sol";
import {IL2ERC20Gateway, L2USDCGateway} from "../L2/gateways/usdc/L2USDCGateway.sol";

import {MockERC20} from "../mocks/MockERC20.sol";

import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";

import {L2GatewayTestBase} from "./L2GatewayTestBase.t.sol";
import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {MockGatewayRecipient} from "./mocks/MockGatewayRecipient.sol";

contract L2USDCGatewayTest is L2GatewayTestBase {
    // from L2USDCGateway
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

    MockERC20 private l1USDC;
    MockERC20 private l2USDC;

    L2USDCGateway private gateway;
    L2GatewayRouter private router;

    L1USDCGateway private counterpartGateway;

    function setUp() public {
        setUpBase();

        // Deploy tokens
        l1USDC = new MockERC20("USDC", "USDC", 6);
        l2USDC = new MockERC20("USDC", "USDC", 6);

        // Deploy L1 contracts
        counterpartGateway = new L1USDCGateway(address(l1USDC), address(l2USDC), address(1), address(1), address(1));

        // Deploy L2 contracts
        router = L2GatewayRouter(_deployProxy(address(new L2GatewayRouter())));
        gateway = _deployGateway(address(l2Messenger));

        // Initialize L2 contracts
        gateway.initialize(address(counterpartGateway), address(router), address(l2Messenger));
        router.initialize(address(0), address(gateway));

        // Prepare token balances
        l2USDC.mint(address(this), type(uint128).max);
        l2USDC.approve(address(gateway), type(uint256).max);
        l2USDC.transferOwnership(address(gateway));
    }

    function testInitialized() public {
        assertEq(l2USDC.owner(), address(gateway));
        assertEq(address(counterpartGateway), gateway.counterpart());
        assertEq(address(router), gateway.router());
        assertEq(address(l2Messenger), gateway.messenger());
        assertEq(address(l1USDC), gateway.l1USDC());
        assertEq(address(l2USDC), gateway.l2USDC());
        assertEq(address(l1USDC), gateway.getL1ERC20Address(address(l2USDC)));
        assertEq(address(l2USDC), gateway.getL2ERC20Address(address(l1USDC)));

        hevm.expectRevert("Initializable: contract is already initialized");
        gateway.initialize(address(counterpartGateway), address(router), address(l2Messenger));
    }

    function testTransferUSDCRoles(address owner) external {
        hevm.assume(owner != address(0));

        // non-whitelisted caller call, should revert
        hevm.expectRevert("only circle caller");
        gateway.transferUSDCRoles(owner);

        // whitelisted caller call
        gateway.updateCircleCaller(address(this));
        assertEq(l2USDC.owner(), address(gateway));
        gateway.transferUSDCRoles(owner);
        assertEq(l2USDC.owner(), owner);
    }

    function testUpdateCircleCaller(address caller) external {
        // non-owner call pause, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.updateCircleCaller(caller);
        hevm.stopPrank();

        // succeed
        assertEq(address(0), gateway.circleCaller());
        gateway.updateCircleCaller(caller);
        assertEq(caller, gateway.circleCaller());
    }

    function testWithdrawPaused() public {
        // non-owner call pause, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.pauseWithdraw(false);
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.pauseWithdraw(true);
        hevm.stopPrank();

        // pause withdraw
        gateway.pauseWithdraw(true);

        // withdraw paused, should revert
        hevm.expectRevert("withdraw paused");
        gateway.withdrawERC20(address(l2USDC), 1, 0);
        hevm.expectRevert("withdraw paused");
        gateway.withdrawERC20(address(l2USDC), address(this), 1, 0);
        hevm.expectRevert("withdraw paused");
        gateway.withdrawERC20AndCall(address(l2USDC), address(this), 1, new bytes(0), 0);
    }

    function testPauseDeposit() public {
        // non-owner call pause, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.pauseDeposit(false);
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.pauseDeposit(true);
        hevm.stopPrank();
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

    function testRouterWithdrawERC20(
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawERC20(true, amount, gasLimit, feePerGas);
    }

    function testRouterWithdrawERC20WithRecipient(
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _withdrawERC20WithRecipient(true, amount, recipient, gasLimit, feePerGas);
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
        gateway.finalizeDepositERC20(address(l1USDC), address(l2USDC), sender, recipient, amount, dataToCall);

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway(address(mockMessenger));
        gateway.initialize(address(counterpartGateway), address(router), address(mockMessenger));

        // only call by counterpart
        hevm.expectRevert(ErrorCallerIsNotCounterpartGateway.selector);
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeDepositERC20.selector,
                address(l1USDC),
                address(l2USDC),
                sender,
                recipient,
                amount,
                dataToCall
            )
        );

        mockMessenger.setXDomainMessageSender(address(counterpartGateway));

        // nonzero msg.value
        hevm.expectRevert("nonzero msg.value");
        mockMessenger.callTarget{value: 1}(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeDepositERC20.selector,
                address(l1USDC),
                address(l2USDC),
                sender,
                recipient,
                amount,
                dataToCall
            )
        );

        // l1 token not USDC
        hevm.expectRevert("l1 token not USDC");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeDepositERC20.selector,
                address(l2USDC),
                address(l2USDC),
                sender,
                recipient,
                amount,
                dataToCall
            )
        );

        // l2 token not USDC
        hevm.expectRevert("l2 token not USDC");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeDepositERC20.selector,
                address(l1USDC),
                address(l1USDC),
                sender,
                recipient,
                amount,
                dataToCall
            )
        );

        // deposit paused
        gateway.pauseDeposit(true);
        hevm.expectRevert("deposit paused");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeDepositERC20.selector,
                address(l1USDC),
                address(l2USDC),
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
        hevm.assume(recipient != address(gateway));

        amount = bound(amount, 1, l2USDC.balanceOf(address(this)));

        // send some USDC to L2ScrollMessenger
        gateway.withdrawERC20(address(l2USDC), amount, 21000);

        // do finalize withdraw eth
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1USDC),
            address(l2USDC),
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

        // conterpart is not L1USDCGateway
        // emit FailedRelayedMessage from L2ScrollMessenger
        hevm.expectEmit(true, false, false, true);
        emit FailedRelayedMessage(keccak256(xDomainCalldata));

        uint256 gatewayBalance = l2USDC.balanceOf(address(gateway));
        uint256 recipientBalance = l2USDC.balanceOf(recipient);
        assertBoolEq(false, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata)));
        hevm.startPrank(AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger)));
        l2Messenger.relayMessage(address(uint160(address(counterpartGateway)) + 1), address(gateway), 0, 0, message);
        hevm.stopPrank();
        assertEq(gatewayBalance, l2USDC.balanceOf(address(gateway)));
        assertEq(recipientBalance, l2USDC.balanceOf(recipient));
        assertBoolEq(false, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testFinalizeDepositERC20(
        address sender,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        MockGatewayRecipient recipient = new MockGatewayRecipient();

        amount = bound(amount, 1, l2USDC.balanceOf(address(this)));

        // send some USDC to L1ScrollMessenger
        gateway.withdrawERC20(address(l2USDC), amount, 21000);

        // do finalize withdraw USDC
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1USDC),
            address(l2USDC),
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

        // emit FinalizeDepositERC20 from L2USDCGateway
        {
            hevm.expectEmit(true, true, true, true);
            emit FinalizeDepositERC20(address(l1USDC), address(l2USDC), sender, address(recipient), amount, dataToCall);
        }

        // emit RelayedMessage from L2ScrollMessenger
        {
            hevm.expectEmit(true, false, false, true);
            emit RelayedMessage(keccak256(xDomainCalldata));
        }

        uint256 gatewayBalance = l2USDC.balanceOf(address(gateway));
        uint256 recipientBalance = l2USDC.balanceOf(address(recipient));
        assertBoolEq(false, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata)));
        hevm.startPrank(AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger)));
        l2Messenger.relayMessage(address(counterpartGateway), address(gateway), 0, 0, message);
        hevm.stopPrank();
        assertEq(gatewayBalance, l2USDC.balanceOf(address(gateway)));
        assertEq(recipientBalance + amount, l2USDC.balanceOf(address(recipient)));
        assertBoolEq(true, l2Messenger.isL1MessageExecuted(keccak256(xDomainCalldata)));
    }

    function _withdrawERC20(
        bool useRouter,
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, l2USDC.balanceOf(address(this)));
        gasLimit = bound(gasLimit, 21000, 1000000);
        feePerGas = 0;

        setL1BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL1ERC20Gateway.finalizeWithdrawERC20.selector,
            address(l1USDC),
            address(l2USDC),
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
                router.withdrawERC20{value: feeToPay}(address(l2USDC), amount, gasLimit);
            } else {
                gateway.withdrawERC20{value: feeToPay}(address(l2USDC), amount, gasLimit);
            }
        } else {
            // token is not l2USDC
            hevm.expectRevert("only USDC is allowed");
            gateway.withdrawERC20(address(l1USDC), amount, gasLimit);

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

            // emit WithdrawERC20 from L2USDCGateway
            hevm.expectEmit(true, true, true, true);
            emit WithdrawERC20(address(l1USDC), address(l2USDC), address(this), address(this), amount, new bytes(0));

            uint256 senderBalance = l2USDC.balanceOf(address(this));
            uint256 gatewayBalance = l2USDC.balanceOf(address(gateway));
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.withdrawERC20{value: feeToPay}(address(l2USDC), amount, gasLimit);
            } else {
                gateway.withdrawERC20{value: feeToPay}(address(l2USDC), amount, gasLimit);
            }
            assertEq(senderBalance - amount, l2USDC.balanceOf(address(this)));
            assertEq(gatewayBalance, l2USDC.balanceOf(address(gateway)));
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
        amount = bound(amount, 0, l2USDC.balanceOf(address(this)));
        gasLimit = bound(gasLimit, 21000, 1000000);
        feePerGas = 0;

        setL1BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL1ERC20Gateway.finalizeWithdrawERC20.selector,
            address(l1USDC),
            address(l2USDC),
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
                router.withdrawERC20{value: feeToPay}(address(l2USDC), recipient, amount, gasLimit);
            } else {
                gateway.withdrawERC20{value: feeToPay}(address(l2USDC), recipient, amount, gasLimit);
            }
        } else {
            // token is not l1USDC
            hevm.expectRevert("only USDC is allowed");
            gateway.withdrawERC20(address(l1USDC), recipient, amount, gasLimit);

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

            // emit WithdrawERC20 from L2USDCGateway
            hevm.expectEmit(true, true, true, true);
            emit WithdrawERC20(address(l1USDC), address(l2USDC), address(this), recipient, amount, new bytes(0));

            uint256 senderBalance = l2USDC.balanceOf(address(this));
            uint256 gatewayBalance = l2USDC.balanceOf(address(gateway));
            uint256 feeVaultBalance = address(feeVault).balance;
            assertEq(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
            if (useRouter) {
                router.withdrawERC20{value: feeToPay}(address(l2USDC), recipient, amount, gasLimit);
            } else {
                gateway.withdrawERC20{value: feeToPay}(address(l2USDC), recipient, amount, gasLimit);
            }
            assertEq(senderBalance - amount, l2USDC.balanceOf(address(this)));
            assertEq(gatewayBalance, l2USDC.balanceOf(address(gateway)));
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertGt(l2Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        }
    }

    function _deployGateway(address messenger) internal returns (L2USDCGateway _gateway) {
        _gateway = L2USDCGateway(_deployProxy(address(0)));

        admin.upgrade(
            ITransparentUpgradeableProxy(address(_gateway)),
            address(
                new L2USDCGateway(
                    address(l1USDC),
                    address(l2USDC),
                    address(counterpartGateway),
                    address(router),
                    address(messenger)
                )
            )
        );
    }
}
