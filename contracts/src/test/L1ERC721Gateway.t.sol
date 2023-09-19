// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {MockERC721} from "solmate/test/utils/mocks/MockERC721.sol";
import {ERC721TokenReceiver} from "solmate/tokens/ERC721.sol";

import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {IL1ERC721Gateway, L1ERC721Gateway} from "../L1/gateways/L1ERC721Gateway.sol";
import {IL1ScrollMessenger} from "../L1/IL1ScrollMessenger.sol";
import {IL2ERC721Gateway, L2ERC721Gateway} from "../L2/gateways/L2ERC721Gateway.sol";
import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";
import {ScrollConstants} from "../libraries/constants/ScrollConstants.sol";

import {L1GatewayTestBase} from "./L1GatewayTestBase.t.sol";
import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {MockERC721Recipient} from "./mocks/MockERC721Recipient.sol";

contract L1ERC721GatewayTest is L1GatewayTestBase, ERC721TokenReceiver {
    // from L1ERC721Gateway
    event FinalizeWithdrawERC721(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _tokenId
    );
    event FinalizeBatchWithdrawERC721(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256[] _tokenIds
    );
    event DepositERC721(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _tokenId
    );
    event BatchDepositERC721(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256[] _tokenIds
    );
    event RefundERC721(address indexed token, address indexed recipient, uint256 tokenId);
    event BatchRefundERC721(address indexed token, address indexed recipient, uint256[] tokenIds);

    uint256 private constant TOKEN_COUNT = 100;

    L1ERC721Gateway private gateway;

    L2ERC721Gateway private counterpartGateway;

    MockERC721 private l1Token;
    MockERC721 private l2Token;
    MockERC721Recipient private mockRecipient;

    function setUp() public {
        setUpBase();

        // Deploy tokens
        l1Token = new MockERC721("Mock L1", "ML1");
        l2Token = new MockERC721("Mock L2", "ML1");

        // Deploy L1 contracts
        gateway = _deployGateway();

        // Deploy L2 contracts
        counterpartGateway = new L2ERC721Gateway();

        // Initialize L1 contracts
        gateway.initialize(address(counterpartGateway), address(l1Messenger));

        // Prepare token balances
        for (uint256 i = 0; i < TOKEN_COUNT; i++) {
            l1Token.mint(address(this), i);
        }
        l1Token.setApprovalForAll(address(gateway), true);

        mockRecipient = new MockERC721Recipient();
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

    function testDepositERC721(
        uint256 tokenId,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _testDepositERC721(tokenId, gasLimit, feePerGas);
    }

    function testDepositERC721WithRecipient(
        uint256 tokenId,
        address to,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _testDepositERC721WithRecipient(tokenId, to, gasLimit, feePerGas);
    }

    function testBatchDepositERC721WithGatewaySuccess(
        uint256 tokenCount,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _testBatchDepositERC721(tokenCount, gasLimit, feePerGas);
    }

    /// @dev batch deposit erc721 with recipient
    function testBatchDepositERC721WithGatewaySuccess(
        uint256 tokenCount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) public {
        _testBatchDepositERC721WithRecipient(tokenCount, recipient, gasLimit, feePerGas);
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
            IL2ERC721Gateway.finalizeDepositERC721.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            address(this),
            0
        );

        // nonzero msg.value, revert
        hevm.expectRevert("nonzero msg.value");
        mockMessenger.callTarget{value: 1}(
            address(gateway),
            abi.encodeWithSelector(gateway.onDropMessage.selector, message)
        );
    }

    function testDropMessage(uint256 tokenId) public {
        gateway.updateTokenMapping(address(l1Token), address(l2Token));

        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        bytes memory message = abi.encodeWithSelector(
            IL2ERC721Gateway.finalizeDepositERC721.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            address(this),
            tokenId
        );
        gateway.depositERC721(address(l1Token), tokenId, defaultGasLimit);

        // skip message 0
        hevm.startPrank(address(rollup));
        messageQueue.popCrossDomainMessage(0, 1, 0x1);
        assertEq(messageQueue.pendingQueueIndex(), 1);
        hevm.stopPrank();

        // drop message 0
        hevm.expectEmit(true, true, false, true);
        emit RefundERC721(address(l1Token), address(this), tokenId);

        assertEq(l1Token.ownerOf(tokenId), address(gateway));
        l1Messenger.dropMessage(address(gateway), address(counterpartGateway), 0, 0, message);
        assertEq(l1Token.ownerOf(tokenId), address(this));
    }

    function testDropMessageBatch(uint256 tokenCount) public {
        tokenCount = bound(tokenCount, 1, TOKEN_COUNT);
        gateway.updateTokenMapping(address(l1Token), address(l2Token));

        uint256[] memory _tokenIds = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            _tokenIds[i] = i;
        }

        bytes memory message = abi.encodeWithSelector(
            IL2ERC721Gateway.finalizeBatchDepositERC721.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            address(this),
            _tokenIds
        );
        gateway.batchDepositERC721(address(l1Token), _tokenIds, defaultGasLimit);

        // skip message 0
        hevm.startPrank(address(rollup));
        messageQueue.popCrossDomainMessage(0, 1, 0x1);
        assertEq(messageQueue.pendingQueueIndex(), 1);
        hevm.stopPrank();

        // drop message 0
        hevm.expectEmit(true, true, false, true);
        emit BatchRefundERC721(address(l1Token), address(this), _tokenIds);
        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(l1Token.ownerOf(_tokenIds[i]), address(gateway));
        }

        l1Messenger.dropMessage(address(gateway), address(counterpartGateway), 0, 0, message);
        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(l1Token.ownerOf(_tokenIds[i]), address(this));
        }
    }

    function testFinalizeWithdrawERC721FailedMocking(
        address sender,
        address recipient,
        uint256 tokenId
    ) public {
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);

        // revert when caller is not messenger
        hevm.expectRevert("only messenger can call");
        gateway.finalizeWithdrawERC721(address(l1Token), address(l2Token), sender, recipient, tokenId);

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway();
        gateway.initialize(address(counterpartGateway), address(mockMessenger));

        // only call by counterpart
        hevm.expectRevert("only call by counterpart");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeWithdrawERC721.selector,
                address(l1Token),
                address(l2Token),
                sender,
                recipient,
                tokenId
            )
        );

        mockMessenger.setXDomainMessageSender(address(counterpartGateway));

        // msg.value mismatch
        hevm.expectRevert("l2 token mismatch");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeWithdrawERC721.selector,
                address(l1Token),
                address(l2Token),
                sender,
                recipient,
                tokenId
            )
        );
    }

    function testFinalizeWithdrawERC721Failed(
        address sender,
        address recipient,
        uint256 tokenId
    ) public {
        hevm.assume(recipient != address(0));
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);

        gateway.updateTokenMapping(address(l1Token), address(l2Token));
        gateway.depositERC721(address(l1Token), tokenId, defaultGasLimit);

        // do finalize withdraw token
        bytes memory message = abi.encodeWithSelector(
            IL1ERC721Gateway.finalizeWithdrawERC721.selector,
            address(l1Token),
            address(l2Token),
            sender,
            recipient,
            tokenId
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

        assertEq(address(gateway), l1Token.ownerOf(tokenId));
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
        assertEq(address(gateway), l1Token.ownerOf(tokenId));
        assertEq(gatewayBalance, l1Token.balanceOf(address(gateway)));
        assertEq(recipientBalance, l1Token.balanceOf(recipient));
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testFinalizeWithdrawERC721(
        address sender,
        address recipient,
        uint256 tokenId
    ) public {
        uint256 size;
        assembly {
            size := extcodesize(recipient)
        }
        hevm.assume(size == 0);
        hevm.assume(recipient != address(0));

        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);

        gateway.updateTokenMapping(address(l1Token), address(l2Token));
        gateway.depositERC721(address(l1Token), tokenId, defaultGasLimit);

        // do finalize withdraw token
        bytes memory message = abi.encodeWithSelector(
            IL1ERC721Gateway.finalizeWithdrawERC721.selector,
            address(l1Token),
            address(l2Token),
            sender,
            recipient,
            tokenId
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

        // emit FinalizeWithdrawERC721 from L1ERC721Gateway
        {
            hevm.expectEmit(true, true, true, true);
            emit FinalizeWithdrawERC721(address(l1Token), address(l2Token), sender, recipient, tokenId);
        }

        // emit RelayedMessage from L1ScrollMessenger
        {
            hevm.expectEmit(true, false, false, true);
            emit RelayedMessage(keccak256(xDomainCalldata));
        }

        assertEq(address(gateway), l1Token.ownerOf(tokenId));
        uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
        uint256 recipientBalance = l1Token.balanceOf(recipient);
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(address(counterpartGateway), address(gateway), 0, 0, message, proof);
        assertEq(recipient, l1Token.ownerOf(tokenId));
        assertEq(gatewayBalance - 1, l1Token.balanceOf(address(gateway)));
        assertEq(recipientBalance + 1, l1Token.balanceOf(recipient));
        assertBoolEq(true, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testFinalizeBatchWithdrawERC721FailedMocking(
        address sender,
        address recipient,
        uint256 tokenCount
    ) public {
        tokenCount = bound(tokenCount, 1, TOKEN_COUNT);
        uint256[] memory _tokenIds = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            _tokenIds[i] = i;
        }

        // revert when caller is not messenger
        hevm.expectRevert("only messenger can call");
        gateway.finalizeBatchWithdrawERC721(address(l1Token), address(l2Token), sender, recipient, _tokenIds);

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway();
        gateway.initialize(address(counterpartGateway), address(mockMessenger));

        // only call by counterpart
        hevm.expectRevert("only call by counterpart");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeBatchWithdrawERC721.selector,
                address(l1Token),
                address(l2Token),
                sender,
                recipient,
                _tokenIds
            )
        );

        mockMessenger.setXDomainMessageSender(address(counterpartGateway));

        // msg.value mismatch
        hevm.expectRevert("l2 token mismatch");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                gateway.finalizeBatchWithdrawERC721.selector,
                address(l1Token),
                address(l2Token),
                sender,
                recipient,
                _tokenIds
            )
        );
    }

    function testFinalizeBatchWithdrawERC721Failed(
        address sender,
        address recipient,
        uint256 tokenCount
    ) public {
        hevm.assume(recipient != address(0));
        tokenCount = bound(tokenCount, 1, TOKEN_COUNT);
        uint256[] memory _tokenIds = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            _tokenIds[i] = i;
        }

        gateway.updateTokenMapping(address(l1Token), address(l2Token));
        gateway.batchDepositERC721(address(l1Token), _tokenIds, defaultGasLimit);

        // do finalize withdraw token
        bytes memory message = abi.encodeWithSelector(
            IL1ERC721Gateway.finalizeBatchWithdrawERC721.selector,
            address(l1Token),
            address(l2Token),
            sender,
            recipient,
            _tokenIds
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

        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(address(gateway), l1Token.ownerOf(_tokenIds[i]));
        }
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
        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(address(gateway), l1Token.ownerOf(_tokenIds[i]));
        }
        assertEq(gatewayBalance, l1Token.balanceOf(address(gateway)));
        assertEq(recipientBalance, l1Token.balanceOf(recipient));
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testFinalizeBatchWithdrawERC721(
        address sender,
        address recipient,
        uint256 tokenCount
    ) public {
        uint256 size;
        assembly {
            size := extcodesize(recipient)
        }
        hevm.assume(size == 0);
        hevm.assume(recipient != address(0));

        tokenCount = bound(tokenCount, 1, TOKEN_COUNT);
        uint256[] memory _tokenIds = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            _tokenIds[i] = i;
        }

        gateway.updateTokenMapping(address(l1Token), address(l2Token));
        gateway.batchDepositERC721(address(l1Token), _tokenIds, defaultGasLimit);

        // do finalize withdraw token
        bytes memory message = abi.encodeWithSelector(
            IL1ERC721Gateway.finalizeBatchWithdrawERC721.selector,
            address(l1Token),
            address(l2Token),
            sender,
            recipient,
            _tokenIds
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

        // emit FinalizeBatchWithdrawERC721 from L1ERC721Gateway
        {
            hevm.expectEmit(true, true, true, true);
            emit FinalizeBatchWithdrawERC721(address(l1Token), address(l2Token), sender, recipient, _tokenIds);
        }

        // emit RelayedMessage from L1ScrollMessenger
        {
            hevm.expectEmit(true, false, false, true);
            emit RelayedMessage(keccak256(xDomainCalldata));
        }

        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(address(gateway), l1Token.ownerOf(_tokenIds[i]));
        }
        uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
        uint256 recipientBalance = l1Token.balanceOf(recipient);
        assertBoolEq(false, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
        l1Messenger.relayMessageWithProof(address(counterpartGateway), address(gateway), 0, 0, message, proof);
        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(recipient, l1Token.ownerOf(_tokenIds[i]));
        }
        assertEq(gatewayBalance - tokenCount, l1Token.balanceOf(address(gateway)));
        assertEq(recipientBalance + tokenCount, l1Token.balanceOf(recipient));
        assertBoolEq(true, l1Messenger.isL2MessageExecuted(keccak256(xDomainCalldata)));
    }

    function testReentranceWhenFinalizeWithdraw(address from, uint256 tokenId) public {
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);

        MockScrollMessenger mockMessenger = new MockScrollMessenger();
        gateway = _deployGateway();
        gateway.initialize(address(counterpartGateway), address(mockMessenger));
        l1Token.setApprovalForAll(address(gateway), true);

        // deposit first
        gateway.updateTokenMapping(address(l1Token), address(l2Token));
        gateway.depositERC721(address(l1Token), tokenId, defaultGasLimit);

        mockRecipient.setCall(
            address(gateway),
            0,
            abi.encodeWithSignature(
                "depositERC721(address,uint256,uint256)",
                address(l1Token),
                tokenId,
                defaultGasLimit
            )
        );
        // finalize withdraw
        mockMessenger.setXDomainMessageSender(address(counterpartGateway));
        hevm.expectRevert("ReentrancyGuard: reentrant call");
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L1ERC721Gateway.finalizeWithdrawERC721.selector,
                address(l1Token),
                address(l2Token),
                from,
                address(mockRecipient),
                tokenId
            )
        );

        // finalize batch withdraw
        mockMessenger.setXDomainMessageSender(address(counterpartGateway));
        hevm.expectRevert("ReentrancyGuard: reentrant call");
        uint256[] memory tokenIds = new uint256[](1);
        tokenIds[0] = tokenId;
        mockMessenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L1ERC721Gateway.finalizeBatchWithdrawERC721.selector,
                address(l1Token),
                address(l2Token),
                from,
                address(mockRecipient),
                tokenIds
            )
        );
    }

    function _testDepositERC721(
        uint256 tokenId,
        uint256 gasLimit,
        uint256 feePerGas
    ) internal {
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);
        uint256 feeToPay = feePerGas * gasLimit;

        hevm.expectRevert("no corresponding l2 token");
        gateway.depositERC721(address(l1Token), tokenId, gasLimit);

        bytes memory message = abi.encodeWithSelector(
            IL2ERC721Gateway.finalizeDepositERC721.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            address(this),
            tokenId
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

        // emit FinalizeWithdrawERC721 from L1ERC721Gateway
        hevm.expectEmit(true, true, true, true);
        emit DepositERC721(address(l1Token), address(l2Token), address(this), address(this), tokenId);

        assertEq(l1Token.ownerOf(tokenId), address(this));
        uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
        uint256 feeVaultBalance = address(feeVault).balance;
        assertEq(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        gateway.depositERC721{value: feeToPay + extraValue}(address(l1Token), tokenId, gasLimit);
        assertEq(address(gateway), l1Token.ownerOf(tokenId));
        assertEq(1 + gatewayBalance, l1Token.balanceOf(address(gateway)));
        assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
        assertGt(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
    }

    function _testDepositERC721WithRecipient(
        uint256 tokenId,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) internal {
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);
        uint256 feeToPay = feePerGas * gasLimit;

        hevm.expectRevert("no corresponding l2 token");
        gateway.depositERC721(address(l1Token), tokenId, gasLimit);

        bytes memory message = abi.encodeWithSelector(
            IL2ERC721Gateway.finalizeDepositERC721.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            recipient,
            tokenId
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

        // emit FinalizeWithdrawERC721 from L1ERC721Gateway
        hevm.expectEmit(true, true, true, true);
        emit DepositERC721(address(l1Token), address(l2Token), address(this), recipient, tokenId);

        assertEq(l1Token.ownerOf(tokenId), address(this));
        uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
        uint256 feeVaultBalance = address(feeVault).balance;
        assertEq(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        gateway.depositERC721{value: feeToPay + extraValue}(address(l1Token), recipient, tokenId, gasLimit);
        assertEq(address(gateway), l1Token.ownerOf(tokenId));
        assertEq(1 + gatewayBalance, l1Token.balanceOf(address(gateway)));
        assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
        assertGt(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
    }

    function _testBatchDepositERC721(
        uint256 tokenCount,
        uint256 gasLimit,
        uint256 feePerGas
    ) internal {
        tokenCount = bound(tokenCount, 1, TOKEN_COUNT);
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);
        uint256 feeToPay = feePerGas * gasLimit;

        uint256[] memory _tokenIds = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            _tokenIds[i] = i;
        }

        hevm.expectRevert("no token to deposit");
        gateway.batchDepositERC721(address(l1Token), new uint256[](0), gasLimit);

        hevm.expectRevert("no corresponding l2 token");
        gateway.batchDepositERC721(address(l1Token), _tokenIds, gasLimit);

        bytes memory message = abi.encodeWithSelector(
            IL2ERC721Gateway.finalizeBatchDepositERC721.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            address(this),
            _tokenIds
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

        // emit FinalizeWithdrawERC721 from L1ERC721Gateway
        hevm.expectEmit(true, true, true, true);
        emit BatchDepositERC721(address(l1Token), address(l2Token), address(this), address(this), _tokenIds);

        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(l1Token.ownerOf(i), address(this));
        }
        uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
        uint256 feeVaultBalance = address(feeVault).balance;
        assertEq(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        gateway.batchDepositERC721{value: feeToPay + extraValue}(address(l1Token), _tokenIds, gasLimit);
        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(l1Token.ownerOf(i), address(gateway));
        }
        assertEq(tokenCount + gatewayBalance, l1Token.balanceOf(address(gateway)));
        assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
        assertGt(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
    }

    function _testBatchDepositERC721WithRecipient(
        uint256 tokenCount,
        address recipient,
        uint256 gasLimit,
        uint256 feePerGas
    ) internal {
        tokenCount = bound(tokenCount, 1, TOKEN_COUNT);
        gasLimit = bound(gasLimit, defaultGasLimit / 2, defaultGasLimit);
        feePerGas = bound(feePerGas, 0, 1000);

        gasOracle.setL2BaseFee(feePerGas);
        uint256 feeToPay = feePerGas * gasLimit;

        uint256[] memory _tokenIds = new uint256[](tokenCount);
        for (uint256 i = 0; i < tokenCount; i++) {
            _tokenIds[i] = i;
        }

        hevm.expectRevert("no token to deposit");
        gateway.batchDepositERC721(address(l1Token), recipient, new uint256[](0), gasLimit);

        hevm.expectRevert("no corresponding l2 token");
        gateway.batchDepositERC721(address(l1Token), recipient, _tokenIds, gasLimit);

        bytes memory message = abi.encodeWithSelector(
            IL2ERC721Gateway.finalizeBatchDepositERC721.selector,
            address(l1Token),
            address(l2Token),
            address(this),
            recipient,
            _tokenIds
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

        // emit FinalizeWithdrawERC721 from L1ERC721Gateway
        hevm.expectEmit(true, true, true, true);
        emit BatchDepositERC721(address(l1Token), address(l2Token), address(this), recipient, _tokenIds);

        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(l1Token.ownerOf(i), address(this));
        }
        uint256 gatewayBalance = l1Token.balanceOf(address(gateway));
        uint256 feeVaultBalance = address(feeVault).balance;
        assertEq(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
        gateway.batchDepositERC721{value: feeToPay + extraValue}(address(l1Token), recipient, _tokenIds, gasLimit);
        for (uint256 i = 0; i < tokenCount; i++) {
            assertEq(l1Token.ownerOf(i), address(gateway));
        }
        assertEq(tokenCount + gatewayBalance, l1Token.balanceOf(address(gateway)));
        assertEq(feeToPay + feeVaultBalance, address(feeVault).balance);
        assertGt(l1Messenger.messageSendTimestamp(keccak256(xDomainCalldata)), 0);
    }

    function _deployGateway() internal returns (L1ERC721Gateway) {
        return L1ERC721Gateway(address(new ERC1967Proxy(address(new L1ERC721Gateway()), new bytes(0))));
    }
}
