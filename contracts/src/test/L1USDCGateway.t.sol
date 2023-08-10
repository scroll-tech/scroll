// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {MockERC20} from "solmate/test/utils/mocks/MockERC20.sol";

import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {L1GatewayRouter} from "../L1/gateways/L1GatewayRouter.sol";
import {IL1ERC20Gateway, L1USDCGateway} from "../L1/gateways/usdc/L1USDCGateway.sol";
import {IL1ScrollMessenger} from "../L1/IL1ScrollMessenger.sol";
import {IL2ERC20Gateway, L2USDCGateway} from "../L2/gateways/usdc/L2USDCGateway.sol";
import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";

import {L1GatewayTestBase} from "./L1GatewayTestBase.t.sol";
import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {MockGatewayRecipient} from "./mocks/MockGatewayRecipient.sol";

contract L1USDCGatewayTest is L1GatewayTestBase {
    // from L1USDCGateway
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

    MockERC20 private l1USDC;
    MockERC20 private l2USDC;

    L1USDCGateway private gateway;
    L1GatewayRouter private router;

    L2USDCGateway private counterpartGateway;

    function setUp() public {
        setUpBase();

        // Deploy tokens
        l1USDC = new MockERC20("USDC", "USDC", 6);
        l2USDC = new MockERC20("USDC", "USDC", 6);

        // Deploy L1 contracts
        gateway = _deployGateway();
        router = L1GatewayRouter(address(new ERC1967Proxy(address(new L1GatewayRouter()), new bytes(0))));

        // Deploy L2 contracts
        counterpartGateway = new L2USDCGateway(address(l1USDC), address(l2USDC));

        // Initialize L1 contracts
        gateway.initialize(address(counterpartGateway), address(router), address(l1Messenger));
        router.initialize(address(0), address(gateway));

        // Prepare token balances
        l1USDC.mint(address(this), type(uint128).max);
        l1USDC.approve(address(gateway), type(uint256).max);
        l1USDC.approve(address(router), type(uint256).max);
    }

    function testInitialized() public {
        assertEq(address(counterpartGateway), gateway.counterpart());
        assertEq(address(router), gateway.router());
        assertEq(address(l1Messenger), gateway.messenger());
        assertEq(address(l1USDC), gateway.l1USDC());
        assertEq(address(l2USDC), gateway.l2USDC());
        assertEq(address(l2USDC), gateway.getL2ERC20Address(address(l1USDC)));

        hevm.expectRevert("Initializable: contract is already initialized");
        gateway.initialize(address(counterpartGateway), address(router), address(l1Messenger));
    }

    function testDepositPaused() public {
        // non-owner call pause, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.pauseDeposit(false);
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.pauseDeposit(true);
        hevm.stopPrank();

        // pause deposit
        gateway.pauseDeposit(true);

        // deposit paused, should revert
        hevm.expectRevert("deposit paused");
        gateway.depositERC20(address(l1USDC), 1, 0);
        hevm.expectRevert("deposit paused");
        gateway.depositERC20(address(l1USDC), address(this), 1, 0);
        hevm.expectRevert("deposit paused");
        gateway.depositERC20AndCall(address(l1USDC), address(this), 1, new bytes(0), 0);
    }

    function testPauseWithdraw() public {
        // non-owner call pause, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.pauseWithdraw(false);
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.pauseWithdraw(true);
        hevm.stopPrank();
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

    function testFinalizeWithdrawERC20FailedMocking(
        address sender,
        address recipient,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        amount = bound(amount, 1, 100000);

        // revert when caller is not messenger
        hevm.expectRevert("only messenger can call");
        gateway.finalizeWithdrawERC20(address(l1USDC), address(l2USDC), sender, recipient, amount, dataToCall);

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway();
        gateway.initialize(address(counterpartGateway), address(router), address(mockMessenger));

        // only call by conterpart
        hevm.expectRevert("only call by counterpart");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeWithdrawERC20.selector,
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
                gateway.finalizeWithdrawERC20.selector,
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
                gateway.finalizeWithdrawERC20.selector,
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
                gateway.finalizeWithdrawERC20.selector,
                address(l1USDC),
                address(l1USDC),
                sender,
                recipient,
                amount,
                dataToCall
            )
        );

        // withdraw paused
        gateway.pauseWithdraw(true);
        hevm.expectRevert("withdraw paused");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeWithdrawERC20.selector,
                address(l1USDC),
                address(l2USDC),
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

        amount = bound(amount, 1, l1USDC.balanceOf(address(this)));

        // deposit some USDC to L1ScrollMessenger
        gateway.depositERC20(address(l1USDC), amount, 0);

        // do finalize withdraw usdc
        bytes memory message = abi.encodeWithSelector(
            IL1ERC20Gateway.finalizeWithdrawERC20.selector,
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

        prepareL2MessageRoot(keccak256(xDomainCalldata));

        IL1ScrollMessenger.L2MessageProof memory proof;
        proof.batchIndex = rollup.lastFinalizedBatchIndex();

        // conterpart is not L2USDCGateway
        // emit FailedRelayedMessage from L1ScrollMessenger
        hevm.expectEmit(true, false, false, true);
        emit FailedRelayedMessage(keccak256(xDomainCalldata));

        uint256 gatewayBalance = l1USDC.balanceOf(address(gateway));
        uint256 recipientBalance = l1USDC.balanceOf(recipient);
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(
            address(uint160(address(counterpartGateway)) + 1),
            address(gateway),
            0,
            0,
            message,
            proof
        );
        assertEq(gatewayBalance, l1USDC.balanceOf(address(gateway)));
        assertEq(recipientBalance, l1USDC.balanceOf(recipient));
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testFinalizeWithdrawERC20(
        address sender,
        uint256 amount,
        bytes memory dataToCall
    ) public {
        MockGatewayRecipient recipient = new MockGatewayRecipient();

        amount = bound(amount, 1, l1USDC.balanceOf(address(this)));

        // deposit some USDC to gateway
        gateway.depositERC20(address(l1USDC), amount, 0);

        // do finalize withdraw usdc
        bytes memory message = abi.encodeWithSelector(
            IL1ERC20Gateway.finalizeWithdrawERC20.selector,
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

        prepareL2MessageRoot(keccak256(xDomainCalldata));

        IL1ScrollMessenger.L2MessageProof memory proof;
        proof.batchIndex = rollup.lastFinalizedBatchIndex();

        // emit FinalizeWithdrawERC20 from L1USDCGateway
        {
            hevm.expectEmit(true, true, true, true);
            emit FinalizeWithdrawERC20(
                address(l1USDC),
                address(l2USDC),
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

        uint256 gatewayBalance = l1USDC.balanceOf(address(gateway));
        uint256 recipientBalance = l1USDC.balanceOf(address(recipient));
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(address(counterpartGateway), address(gateway), 0, 0, message, proof);
        assertEq(gatewayBalance - amount, l1USDC.balanceOf(address(gateway)));
        assertEq(recipientBalance + amount, l1USDC.balanceOf(address(recipient)));
        assertBoolEq(true, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function _depositERC20(
        bool useRouter,
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) private {
        amount = bound(amount, 0, l1USDC.balanceOf(address(this)));
        gasLimit = bound(gasLimit, 0, 1000000);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
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
            hevm.expectRevert("deposit zero amount");
            if (useRouter) {
                router.depositERC20{value: feeToPay + extraValue}(address(l1USDC), amount, gasLimit);
            } else {
                gateway.depositERC20{value: feeToPay + extraValue}(address(l1USDC), amount, gasLimit);
            }
        } else {
            // token is not l1USDC
            hevm.expectRevert("only USDC is allowed");
            gateway.depositERC20(address(l2USDC), amount, gasLimit);

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

            // emit DepositERC20 from L1USDCGateway
            hevm.expectEmit(true, true, true, true);
            emit DepositERC20(address(l1USDC), address(l2USDC), address(this), address(this), amount, new bytes(0));

            uint256 gatewayBalance = l1USDC.balanceOf(address(gateway));
            uint256 feeVaultBalance = address(feeVault).balance;
            assertBoolEq(false, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
            if (useRouter) {
                router.depositERC20{value: feeToPay + extraValue}(address(l1USDC), amount, gasLimit);
            } else {
                gateway.depositERC20{value: feeToPay + extraValue}(address(l1USDC), amount, gasLimit);
            }
            assertEq(amount + gatewayBalance, l1USDC.balanceOf(address(gateway)));
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
        amount = bound(amount, 0, l1USDC.balanceOf(address(this)));
        gasLimit = bound(gasLimit, 0, 1000000);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);

        uint256 feeToPay = feePerGas * gasLimit;
        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
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
            hevm.expectRevert("deposit zero amount");
            if (useRouter) {
                router.depositERC20{value: feeToPay + extraValue}(address(l1USDC), recipient, amount, gasLimit);
            } else {
                gateway.depositERC20{value: feeToPay + extraValue}(address(l1USDC), recipient, amount, gasLimit);
            }
        } else {
            // token is not l1USDC
            hevm.expectRevert("only USDC is allowed");
            gateway.depositERC20(address(l2USDC), recipient, amount, gasLimit);

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

            // emit DepositERC20 from L1USDCGateway
            hevm.expectEmit(true, true, true, true);
            emit DepositERC20(address(l1USDC), address(l2USDC), address(this), recipient, amount, new bytes(0));

            uint256 gatewayBalance = l1USDC.balanceOf(address(gateway));
            uint256 feeVaultBalance = address(feeVault).balance;
            assertBoolEq(false, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
            if (useRouter) {
                router.depositERC20{value: feeToPay + extraValue}(address(l1USDC), recipient, amount, gasLimit);
            } else {
                gateway.depositERC20{value: feeToPay + extraValue}(address(l1USDC), recipient, amount, gasLimit);
            }
            assertEq(amount + gatewayBalance, l1USDC.balanceOf(address(gateway)));
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertBoolEq(true, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
        }
    }

    function _deployGateway() internal returns (L1USDCGateway) {
        return
            L1USDCGateway(
                payable(new ERC1967Proxy(address(new L1USDCGateway(address(l1USDC), address(l2USDC))), new bytes(0)))
            );
    }
}
