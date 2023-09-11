// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";
import {MockERC1155} from "solmate/test/utils/mocks/MockERC1155.sol";
import {ERC1155TokenReceiver} from "solmate/tokens/ERC1155.sol";

import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {IL1ERC1155Gateway, L1ERC1155Gateway} from "../L1/gateways/L1ERC1155Gateway.sol";
import {IL1ScrollMessenger} from "../L1/IL1ScrollMessenger.sol";
import {IL2ERC1155Gateway, L2ERC1155Gateway} from "../L2/gateways/L2ERC1155Gateway.sol";
import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";
import {ScrollConstants} from "../libraries/constants/ScrollConstants.sol";

import {L1GatewayTestBase} from "./L1GatewayTestBase.t.sol";
import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {MockERC1155Recipient} from "./mocks/MockERC1155Recipient.sol";

contract L1ERC1155GatewayTest is L1GatewayTestBase, ERC1155TokenReceiver {
    // from L1ERC1155Gateway
    event FinalizeWithdrawERC1155(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _tokenId,
        uint256 _amount
    );
    event FinalizeBatchWithdrawERC1155(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256[] _tokenIds,
        uint256[] _amounts
    );
    event DepositERC1155(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _tokenId,
        uint256 _amount
    );
    event BatchDepositERC1155(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256[] _tokenIds,
        uint256[] _amounts
    );
    event RefundERC1155(address indexed token, address indexed recipient, uint256 tokenId, uint256 amount);
    event BatchRefundERC1155(address indexed token, address indexed recipient, uint256[] tokenIds, uint256[] amounts);

    uint256 private constant TOKEN_COUNT = 100;
    uint256 private constant MAX_TOKEN_BALANCE = 1000000000;

    L1ERC1155Gateway private gateway;

    L2ERC1155Gateway private counterpartGateway;

    MockERC1155 private l1Token;
    MockERC1155 private l2Token;
    MockERC1155Recipient private mockRecipient;

    function setUp() public {
        setUpBase();

        // Deploy tokens
        l1Token = new MockERC1155();
        l2Token = new MockERC1155();

        // Deploy L1 contracts
        gateway = _deployGateway();

        // Deploy L2 contracts
        counterpartGateway = new L2ERC1155Gateway();

        // Initialize L1 contracts
        gateway.initialize(address(counterpartGateway), address(l1Messenger));

        // Prepare token balances
        for (uint256 i = 0; i < TOKEN_COUNT; i++) {
            l1Token.mint(address(this), i, MAX_TOKEN_BALANCE, "");
        }
        l1Token.setApprovalForAll(address(gateway), true);

        mockRecipient = new MockERC1155Recipient();
    }

    function testInitialized() public {
        assertEq(address(counterpartGateway), gateway.counterpart());
        assertEq(address(0), gateway.router());
        assertEq(address(l1Messenger), gateway.messenger());

        assertEq(address(0), gateway.tokenMapping(address(l1Token)));

        hevm.expectRevert("Initializable: contract is already initialized");
        gateway.initialize(address(1), address(l1Messenger));
    }

    function testUpdateTokenMappingFailed(address token1) public {
        // call by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        gateway.updateTokenMapping(token1, token1);
        hevm.stopPrank();

        // l2 token is zero, should revert
        hevm.expectRevert("token address cannot be 0");
        gateway.updateTokenMapping(token1, address(0));
    }

    function testUpdateTokenMappingSuccess(address token1, address token2) public {
        hevm.assume(token2 != address(0));

        assertEq(gateway.tokenMapping(token1), address(0));
        gateway.updateTokenMapping(token1, token2);
        assertEq(gateway.tokenMapping(token1), token2);
    }

    function testDepositERC1155(
        uint256 tokenId,
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _testDepositERC1155(tokenId, amount, gasLimit, feePerGas);
    }

    function testDepositERC1155WithRecipient(
        uint256 tokenId,
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _testDepositERC1155WithRecipient(tokenId, amount, recipient, gasLimit, feePerGas);
    }

    function testBatchDepositERC1155(
        uint256 tokenCount,
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _testBatchDepositERC1155(tokenCount, amount, gasLimit, feePerGas);
    }

    function testBatchDepositERC1155WithRecipient(
        uint256 tokenCount,
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _testBatchDepositERC1155WithRecipient(tokenCount, amount, recipient, gasLimit, feePerGas);
    }

    function testDropMessageMocking() public {
        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway();
        gateway.initialize(address(counterpartGateway), address(mockMessenger));

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
            IL2ERC1155Gateway.finalizeDepositERC1155.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            address(this),
            0,
            0
        );

        // nonzero msg.value, revert
        hevm.expectRevert("nonzero msg.value");
        mockMessenger.callTarget{value: 1}(
            address(gateway),
            abi.encodeWithSelector(gateway.onDropMessage.selector, message)
        );
    }

    function testDropMessage(uint256 tokenId, uint256 amount) public {
        gateway.updateTokenMapping(address(l1Token), address(l2Token));

        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        amount = bound(amount, 1, MAX_TOKEN_BALANCE);
        bytes memory message = abi.encodeWithSelector(
            IL2ERC1155Gateway.finalizeDepositERC1155.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            address(this),
            tokenId,
            amount
        );
        gateway.depositERC1155(address(l1Token), tokenId, amount, defaultGasLimit);

        // skip message 0
        hevm.startPrank(address(rollup));
        messageQueue.popCrossDomainMessage(0, 1, 0x1);
        assertEq(messageQueue.pendingQueueIndex(), 1);
        hevm.stopPrank();

        // drop message 0
        hevm.expectEmit(true, true, false, true);
        emit RefundERC1155(address(l1Token), address(this), tokenId, amount);

        uint256 balance = l1Token.balanceOf(address(this), tokenId);
        l1Messenger.dropMessage(address(gateway), address(counterpartGateway), 0, 0, message);
        assertEq(balance + amount, l1Token.balanceOf(address(this), tokenId));
    }

    function testDropMessageBatch(uint256 tokenCount, uint256 amount) public {
        tokenCount = bound(tokenCount, 1, TOKEN_COUNT);
        amount = bound(amount, 1, MAX_TOKEN_BALANCE);
        gateway.updateTokenMapping(address(l1Token), address(l2Token));

        uint256[] memory _tokenIds = new uint256[](tokenCount);
        uint256[] memory _amounts = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            _tokenIds[i] = i;
            _amounts[i] = amount;
        }

        bytes memory message = abi.encodeWithSelector(
            IL2ERC1155Gateway.finalizeBatchDepositERC1155.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            address(this),
            _tokenIds,
            _amounts
        );
        gateway.batchDepositERC1155(address(l1Token), _tokenIds, _amounts, defaultGasLimit);

        // skip message 0
        hevm.startPrank(address(rollup));
        messageQueue.popCrossDomainMessage(0, 1, 0x1);
        assertEq(messageQueue.pendingQueueIndex(), 1);
        hevm.stopPrank();

        // drop message 0
        hevm.expectEmit(true, true, false, true);
        emit BatchRefundERC1155(address(l1Token), address(this), _tokenIds, _amounts);

        uint256[] memory balances = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            balances[i] = l1Token.balanceOf(address(this), _tokenIds[i]);
        }
        l1Messenger.dropMessage(address(gateway), address(counterpartGateway), 0, 0, message);
        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(balances[i] + _amounts[i], l1Token.balanceOf(address(this), _tokenIds[i]));
        }
    }

    function testFinalizeWithdrawERC1155FailedMocking(
        address sender,
        address recipient,
        uint256 tokenId,
        uint256 amount
    ) public {
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        amount = bound(amount, 1, MAX_TOKEN_BALANCE);

        // revert when caller is not messenger
        hevm.expectRevert("only messenger can call");
        gateway.finalizeWithdrawERC1155(address(l1Token), address(l2Token), sender, recipient, tokenId, amount);

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway();
        gateway.initialize(address(counterpartGateway), address(mockMessenger));

        // only call by counterpart
        hevm.expectRevert("only call by counterpart");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeWithdrawERC1155.selector,
                address(l1Token),
                address(l2Token),
                sender,
                recipient,
                tokenId,
                amount
            )
        );

        mockMessenger.setXDomainMessageSender(address(counterpartGateway));

        // msg.value mismatch
        hevm.expectRevert("l2 token mismatch");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeWithdrawERC1155.selector,
                address(l1Token),
                address(l2Token),
                sender,
                recipient,
                tokenId,
                amount
            )
        );
    }

    function testFinalizeWithdrawERC1155Failed(
        address sender,
        address recipient,
        uint256 tokenId,
        uint256 amount
    ) public {
        hevm.assume(recipient != address(0));
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        amount = bound(amount, 1, MAX_TOKEN_BALANCE);

        gateway.updateTokenMapping(address(l1Token), address(l2Token));
        gateway.depositERC1155(address(l1Token), tokenId, amount, defaultGasLimit);

        // do finalize withdraw token
        bytes memory message = abi.encodeWithSelector(
            IL1ERC1155Gateway.finalizeWithdrawERC1155.selector,
            address(l1Token),
            address(l2Token),
            sender,
            recipient,
            tokenId,
            amount
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

        uint256 gatewayBalance = l1Token.balanceOf(address(gateway), tokenId);
        uint256 recipientBalance = l1Token.balanceOf(recipient, tokenId);
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(
            address(uint160(address(counterpartGateway)) + 1),
            address(gateway),
            0,
            0,
            message,
            proof
        );
        assertEq(gatewayBalance, l1Token.balanceOf(address(gateway), tokenId));
        assertEq(recipientBalance, l1Token.balanceOf(recipient, tokenId));
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testFinalizeWithdrawERC1155(
        address sender,
        address recipient,
        uint256 tokenId,
        uint256 amount
    ) public {
        uint256 size;
        assembly {
            size := extcodesize(recipient)
        }
        hevm.assume(size == 0);
        hevm.assume(recipient != address(0));

        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        amount = bound(amount, 1, MAX_TOKEN_BALANCE);

        gateway.updateTokenMapping(address(l1Token), address(l2Token));
        gateway.depositERC1155(address(l1Token), tokenId, amount, defaultGasLimit);

        // do finalize withdraw token
        bytes memory message = abi.encodeWithSelector(
            IL1ERC1155Gateway.finalizeWithdrawERC1155.selector,
            address(l1Token),
            address(l2Token),
            sender,
            recipient,
            tokenId,
            amount
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

        // emit FinalizeWithdrawERC1155 from L1ERC1155Gateway
        {
            hevm.expectEmit(true, true, true, true);
            emit FinalizeWithdrawERC1155(address(l1Token), address(l2Token), sender, recipient, tokenId, amount);
        }

        // emit RelayedMessage from L1ScrollMessenger
        {
            hevm.expectEmit(true, false, false, true);
            emit RelayedMessage(keccak256(xDomainCalldata));
        }

        uint256 gatewayBalance = l1Token.balanceOf(address(gateway), tokenId);
        uint256 recipientBalance = l1Token.balanceOf(recipient, tokenId);
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(address(counterpartGateway), address(gateway), 0, 0, message, proof);
        assertEq(gatewayBalance - amount, l1Token.balanceOf(address(gateway), tokenId));
        assertEq(recipientBalance + amount, l1Token.balanceOf(recipient, tokenId));
        assertBoolEq(true, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testFinalizeBatchWithdrawERC1155FailedMocking(
        address sender,
        address recipient,
        uint256 tokenCount,
        uint256 amount
    ) public {
        tokenCount = bound(tokenCount, 1, TOKEN_COUNT);
        amount = bound(amount, 1, MAX_TOKEN_BALANCE);
        uint256[] memory _tokenIds = new uint256[](tokenCount);
        uint256[] memory _amounts = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            _tokenIds[i] = i;
            _amounts[i] = amount;
        }

        // revert when caller is not messenger
        hevm.expectRevert("only messenger can call");
        gateway.finalizeBatchWithdrawERC1155(
            address(l1Token),
            address(l2Token),
            sender,
            recipient,
            _tokenIds,
            _amounts
        );

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway();
        gateway.initialize(address(counterpartGateway), address(mockMessenger));

        // only call by counterpart
        hevm.expectRevert("only call by counterpart");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeBatchWithdrawERC1155.selector,
                address(l1Token),
                address(l2Token),
                sender,
                recipient,
                _tokenIds,
                _amounts
            )
        );

        mockMessenger.setXDomainMessageSender(address(counterpartGateway));

        // msg.value mismatch
        hevm.expectRevert("l2 token mismatch");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeBatchWithdrawERC1155.selector,
                address(l1Token),
                address(l2Token),
                sender,
                recipient,
                _tokenIds,
                _amounts
            )
        );
    }

    function testFinalizeBatchWithdrawERC1155Failed(
        address sender,
        address recipient,
        uint256 tokenCount,
        uint256 amount
    ) public {
        hevm.assume(recipient != address(0));
        tokenCount = bound(tokenCount, 1, TOKEN_COUNT);
        amount = bound(amount, 1, MAX_TOKEN_BALANCE);
        uint256[] memory _tokenIds = new uint256[](tokenCount);
        uint256[] memory _amounts = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            _tokenIds[i] = i;
            _amounts[i] = amount;
        }

        gateway.updateTokenMapping(address(l1Token), address(l2Token));
        gateway.batchDepositERC1155(address(l1Token), _tokenIds, _amounts, defaultGasLimit);

        // do finalize withdraw token
        bytes memory message = abi.encodeWithSelector(
            IL1ERC1155Gateway.finalizeBatchWithdrawERC1155.selector,
            address(l1Token),
            address(l2Token),
            sender,
            recipient,
            _tokenIds,
            _amounts
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

        uint256[] memory gatewayBalances = new uint256[](tokenCount);
        uint256[] memory recipientBalances = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            gatewayBalances[i] = l1Token.balanceOf(address(gateway), i);
            recipientBalances[i] = l1Token.balanceOf(recipient, i);
        }
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(
            address(uint160(address(counterpartGateway)) + 1),
            address(gateway),
            0,
            0,
            message,
            proof
        );
        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(gatewayBalances[i], l1Token.balanceOf(address(gateway), i));
            assertEq(recipientBalances[i], l1Token.balanceOf(recipient, i));
        }
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testFinalizeBatchWithdrawERC1155(
        address sender,
        address recipient,
        uint256 tokenCount,
        uint256 amount
    ) public {
        uint256 size;
        assembly {
            size := extcodesize(recipient)
        }
        hevm.assume(size == 0);
        hevm.assume(recipient != address(0));

        tokenCount = bound(tokenCount, 1, TOKEN_COUNT);
        amount = bound(amount, 1, MAX_TOKEN_BALANCE);
        uint256[] memory _tokenIds = new uint256[](tokenCount);
        uint256[] memory _amounts = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            _tokenIds[i] = i;
            _amounts[i] = amount;
        }

        gateway.updateTokenMapping(address(l1Token), address(l2Token));
        gateway.batchDepositERC1155(address(l1Token), _tokenIds, _amounts, defaultGasLimit);

        // do finalize withdraw token
        bytes memory message = abi.encodeWithSelector(
            IL1ERC1155Gateway.finalizeBatchWithdrawERC1155.selector,
            address(l1Token),
            address(l2Token),
            sender,
            recipient,
            _tokenIds,
            _amounts
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

        // emit FinalizeBatchWithdrawERC1155 from L1ERC1155Gateway
        {
            hevm.expectEmit(true, true, true, true);
            emit FinalizeBatchWithdrawERC1155(
                address(l1Token),
                address(l2Token),
                sender,
                recipient,
                _tokenIds,
                _amounts
            );
        }

        // emit RelayedMessage from L1ScrollMessenger
        {
            hevm.expectEmit(true, false, false, true);
            emit RelayedMessage(keccak256(xDomainCalldata));
        }

        uint256[] memory gatewayBalances = new uint256[](tokenCount);
        uint256[] memory recipientBalances = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            gatewayBalances[i] = l1Token.balanceOf(address(gateway), i);
            recipientBalances[i] = l1Token.balanceOf(recipient, i);
        }
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(address(counterpartGateway), address(gateway), 0, 0, message, proof);

        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(gatewayBalances[i] - _amounts[i], l1Token.balanceOf(address(gateway), i));
            assertEq(recipientBalances[i] + _amounts[i], l1Token.balanceOf(recipient, i));
        }
        assertBoolEq(true, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testReentranceWhenFinalizeWithdraw(
        address from,
        uint256 tokenId,
        uint256 amount
    ) public {
        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway();
        gateway.initialize(address(counterpartGateway), address(mockMessenger));
        l1Token.setApprovalForAll(address(gateway), true);

        // deposit first
        gateway.updateTokenMapping(address(l1Token), address(l2Token));
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        amount = bound(amount, 1, MAX_TOKEN_BALANCE);
        gateway.depositERC1155(address(l1Token), tokenId, amount, defaultGasLimit);

        mockRecipient.setCall(
            address(gateway),
            0,
            abi.encodeWithSignature(
                "depositERC1155(address,uint256,uint256,uint256)",
                address(l1Token),
                tokenId,
                amount,
                0
            )
        );

        // finalize withdraw
        mockMessenger.setXDomainMessageSender(address(counterpartGateway));
        hevm.expectRevert("ReentrancyGuard: reentrant call");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                IL1ERC1155Gateway.finalizeWithdrawERC1155.selector,
                address(l1Token),
                address(l2Token),
                from,
                address(mockRecipient),
                tokenId,
                amount
            )
        );

        // finalize batch withdraw
        mockMessenger.setXDomainMessageSender(address(counterpartGateway));
        hevm.expectRevert("ReentrancyGuard: reentrant call");
        uint256[] memory tokenIds = new uint256[](1);
        uint256[] memory amounts = new uint256[](1);
        tokenIds[0] = tokenId;
        amounts[0] = amount;
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                IL1ERC1155Gateway.finalizeBatchWithdrawERC1155.selector,
                address(l1Token),
                address(l2Token),
                from,
                address(mockRecipient),
                tokenIds,
                amounts
            )
        );
    }

    function _testDepositERC1155(
        uint256 tokenId,
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) internal {
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        amount = bound(amount, 0, MAX_TOKEN_BALANCE);
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);
        uint256 feeToPay = feePerGas * gasLimit;

        bytes memory message = abi.encodeWithSelector(
            IL2ERC1155Gateway.finalizeDepositERC1155.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            address(this),
            tokenId,
            amount
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
            gateway.depositERC1155{value: feeToPay + extraValue}(address(l1Token), tokenId, amount, gasLimit);
        } else {
            hevm.expectRevert("no corresponding l2 token");
            gateway.depositERC1155(address(l1Token), tokenId, amount, gasLimit);

            gateway.updateTokenMapping(address(l1Token), address(l2Token));
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

            // emit FinalizeWithdrawERC1155 from L1ERC1155Gateway
            hevm.expectEmit(true, true, true, true);
            emit DepositERC1155(address(l1Token), address(l2Token), address(this), address(this), tokenId, amount);

            uint256 gatewayBalance = l1Token.balanceOf(address(gateway), tokenId);
            uint256 feeVaultBalance = address(feeVault).balance;
            assertBoolEq(false, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
            gateway.depositERC1155{value: feeToPay + extraValue}(address(l1Token), tokenId, amount, gasLimit);
            assertEq(amount + gatewayBalance, l1Token.balanceOf(address(gateway), tokenId));
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertBoolEq(true, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
        }
    }

    function _testDepositERC1155WithRecipient(
        uint256 tokenId,
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) internal {
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        amount = bound(amount, 0, MAX_TOKEN_BALANCE);
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);
        uint256 feeToPay = feePerGas * gasLimit;

        bytes memory message = abi.encodeWithSelector(
            IL2ERC1155Gateway.finalizeDepositERC1155.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            recipient,
            tokenId,
            amount
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
            gateway.depositERC1155{value: feeToPay + extraValue}(
                address(l1Token),
                recipient,
                tokenId,
                amount,
                gasLimit
            );
        } else {
            hevm.expectRevert("no corresponding l2 token");
            gateway.depositERC1155(address(l1Token), tokenId, amount, gasLimit);

            gateway.updateTokenMapping(address(l1Token), address(l2Token));
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

            // emit FinalizeWithdrawERC1155 from L1ERC1155Gateway
            hevm.expectEmit(true, true, true, true);
            emit DepositERC1155(address(l1Token), address(l2Token), address(this), recipient, tokenId, amount);

            uint256 gatewayBalance = l1Token.balanceOf(address(gateway), tokenId);
            uint256 feeVaultBalance = address(feeVault).balance;
            assertBoolEq(false, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
            gateway.depositERC1155{value: feeToPay + extraValue}(
                address(l1Token),
                recipient,
                tokenId,
                amount,
                gasLimit
            );
            assertEq(amount + gatewayBalance, l1Token.balanceOf(address(gateway), tokenId));
            assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
            assertBoolEq(true, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
        }
    }

    function _testBatchDepositERC1155(
        uint256 tokenCount,
        uint256 amount,
        uint256 gasLimit,
        uint256 feePerGas
    ) internal {
        tokenCount = bound(tokenCount, 1, TOKEN_COUNT);
        amount = bound(amount, 1, MAX_TOKEN_BALANCE);
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);
        uint256 feeToPay = feePerGas * gasLimit;

        uint256[] memory _tokenIds = new uint256[](tokenCount);
        uint256[] memory _amounts = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            _tokenIds[i] = i;
            _amounts[i] = amount;
        }

        hevm.expectRevert("no token to deposit");
        gateway.batchDepositERC1155(address(l1Token), new uint256[](0), new uint256[](0), gasLimit);

        hevm.expectRevert("length mismatch");
        gateway.batchDepositERC1155(address(l1Token), new uint256[](1), new uint256[](0), gasLimit);

        hevm.expectRevert("deposit zero amount");
        gateway.batchDepositERC1155(address(l1Token), _tokenIds, new uint256[](tokenCount), gasLimit);

        hevm.expectRevert("no corresponding l2 token");
        gateway.batchDepositERC1155(address(l1Token), _tokenIds, _amounts, gasLimit);

        bytes memory message = abi.encodeWithSelector(
            IL2ERC1155Gateway.finalizeBatchDepositERC1155.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            address(this),
            _tokenIds,
            _amounts
        );
        bytes memory xDomainCalldata = abi.encodeWithSignature(
            "relayMessage(address,address,uint256,uint256,bytes)",
            address(gateway),
            address(counterpartGateway),
            0,
            0,
            message
        );

        gateway.updateTokenMapping(address(l1Token), address(l2Token));

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

        // emit FinalizeWithdrawERC1155 from L1ERC1155Gateway
        hevm.expectEmit(true, true, true, true);
        emit BatchDepositERC1155(address(l1Token), address(l2Token), address(this), address(this), _tokenIds, _amounts);

        uint256[] memory gatewayBalances = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            gatewayBalances[i] = l1Token.balanceOf(address(gateway), i);
        }
        uint256 feeVaultBalance = address(feeVault).balance;
        assertBoolEq(false, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
        gateway.batchDepositERC1155{value: feeToPay + extraValue}(address(l1Token), _tokenIds, _amounts, gasLimit);
        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(gatewayBalances[i] + amount, l1Token.balanceOf(address(gateway), i));
        }
        assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
        assertBoolEq(true, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
    }

    function _testBatchDepositERC1155WithRecipient(
        uint256 tokenCount,
        uint256 amount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) internal {
        tokenCount = bound(tokenCount, 1, TOKEN_COUNT);
        amount = bound(amount, 1, MAX_TOKEN_BALANCE);
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);
        uint256 feeToPay = feePerGas * gasLimit;

        uint256[] memory _tokenIds = new uint256[](tokenCount);
        uint256[] memory _amounts = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            _tokenIds[i] = i;
            _amounts[i] = amount;
        }

        hevm.expectRevert("no token to deposit");
        gateway.batchDepositERC1155(address(l1Token), recipient, new uint256[](0), new uint256[](0), gasLimit);

        hevm.expectRevert("length mismatch");
        gateway.batchDepositERC1155(address(l1Token), recipient, new uint256[](1), new uint256[](0), gasLimit);

        hevm.expectRevert("deposit zero amount");
        gateway.batchDepositERC1155(address(l1Token), recipient, _tokenIds, new uint256[](tokenCount), gasLimit);

        hevm.expectRevert("no corresponding l2 token");
        gateway.batchDepositERC1155(address(l1Token), recipient, _tokenIds, _amounts, gasLimit);

        bytes memory message = abi.encodeWithSelector(
            IL2ERC1155Gateway.finalizeBatchDepositERC1155.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            recipient,
            _tokenIds,
            _amounts
        );
        bytes memory xDomainCalldata = abi.encodeWithSignature(
            "relayMessage(address,address,uint256,uint256,bytes)",
            address(gateway),
            address(counterpartGateway),
            0,
            0,
            message
        );

        gateway.updateTokenMapping(address(l1Token), address(l2Token));

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

        // emit FinalizeWithdrawERC1155 from L1ERC1155Gateway
        hevm.expectEmit(true, true, true, true);
        emit BatchDepositERC1155(address(l1Token), address(l2Token), address(this), recipient, _tokenIds, _amounts);

        uint256[] memory gatewayBalances = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            gatewayBalances[i] = l1Token.balanceOf(address(gateway), i);
        }
        uint256 feeVaultBalance = address(feeVault).balance;
        assertBoolEq(false, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
        gateway.batchDepositERC1155{value: feeToPay + extraValue}(
            address(l1Token),
            recipient,
            _tokenIds,
            _amounts,
            gasLimit
        );
        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(gatewayBalances[i] + amount, l1Token.balanceOf(address(gateway), i));
        }
        assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
        assertBoolEq(true, l1Messenger.isL1MessageSent(keccak256(xDomainCalldata)));
    }

    function _deployGateway() internal returns (L1ERC1155Gateway) {
        return L1ERC1155Gateway(address(new ERC1967Proxy(address(new L1ERC1155Gateway()), new bytes(0))));
    }
}
