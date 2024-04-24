// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {MockERC20} from "solmate/test/utils/mocks/MockERC20.sol";

import {ITransparentUpgradeableProxy} from "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";
import {Strings} from "@openzeppelin/contracts/utils/Strings.sol";

import {L1BatchBridgeGateway} from "../../batch-bridge/L1BatchBridgeGateway.sol";
import {L2BatchBridgeGateway} from "../../batch-bridge/L2BatchBridgeGateway.sol";
import {BatchBridgeCodec} from "../../batch-bridge/BatchBridgeCodec.sol";
import {IL1ERC20Gateway, L1CustomERC20Gateway} from "../../L1/gateways/L1CustomERC20Gateway.sol";
import {L1GatewayRouter} from "../../L1/gateways/L1GatewayRouter.sol";
import {IL2ERC20Gateway, L2CustomERC20Gateway} from "../../L2/gateways/L2CustomERC20Gateway.sol";
import {AddressAliasHelper} from "../../libraries/common/AddressAliasHelper.sol";
import {ScrollConstants} from "../../libraries/constants/ScrollConstants.sol";

import {L1GatewayTestBase} from "../L1GatewayTestBase.t.sol";

contract L1BatchBridgeGatewayTest is L1GatewayTestBase {
    event Deposit(
        address indexed sender,
        address indexed token,
        uint256 indexed batchIndex,
        uint256 amount,
        uint256 fee
    );
    event BatchDeposit(address indexed caller, address indexed l1Token, uint256 indexed batchIndex, address l2Token);
    event DepositERC20(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    uint24 private constant SAFE_BATCH_BRIDGE_GAS_LIMIT = 200000;
    uint24 ETH_DEPOSIT_SAFE_GAS_LIMIT = 300000;
    uint24 ERC20_DEPOSIT_SAFE_GAS_LIMIT = 200000;

    uint256 private constant L2_GAS_PRICE = 10;

    L1BatchBridgeGateway private batch;
    L1CustomERC20Gateway private gateway;
    L1GatewayRouter private router;

    L2CustomERC20Gateway private counterpartGateway;
    L2BatchBridgeGateway private counterpartBatch;

    MockERC20 private l1Token;
    MockERC20 private l2Token;

    address private batchFeeVault;

    function setUp() public {
        __L1GatewayTestBase_setUp();

        batchFeeVault = address(uint160(address(this)) - 2);

        // Deploy tokens
        l1Token = new MockERC20("Mock L1", "ML1", 18);
        l2Token = new MockERC20("Mock L2", "ML2", 18);

        // Deploy L2 contracts
        counterpartGateway = new L2CustomERC20Gateway(address(1), address(1), address(1));
        counterpartBatch = new L2BatchBridgeGateway(address(1), address(1));

        // Deploy L1 contracts
        router = L1GatewayRouter(_deployProxy(address(new L1GatewayRouter())));
        gateway = L1CustomERC20Gateway(_deployProxy(address(0)));
        batch = L1BatchBridgeGateway(payable(_deployProxy(address(0))));

        // Initialize L1 contracts
        admin.upgrade(
            ITransparentUpgradeableProxy(address(gateway)),
            address(new L1CustomERC20Gateway(address(counterpartGateway), address(router), address(l1Messenger)))
        );
        gateway.initialize(address(counterpartGateway), address(router), address(l1Messenger));
        admin.upgrade(
            ITransparentUpgradeableProxy(address(batch)),
            address(
                new L1BatchBridgeGateway(
                    address(counterpartBatch),
                    address(router),
                    address(l1Messenger),
                    address(messageQueue)
                )
            )
        );
        batch.initialize(batchFeeVault);
        router.initialize(address(0), address(gateway));
        messageQueue.setL2BaseFee(L2_GAS_PRICE);

        // Prepare token balances
        l1Token.mint(address(this), type(uint128).max);
        gateway.updateTokenMapping(address(l1Token), address(l2Token));
        hevm.warp(1000000);
    }

    function testInitialized() external {
        assertBoolEq(true, batch.hasRole(bytes32(0), address(this)));
        assertEq(address(counterpartBatch), batch.counterpart());
        assertEq(address(router), batch.router());
        assertEq(address(l1Messenger), batch.messenger());
        assertEq(address(messageQueue), batch.queue());

        hevm.expectRevert("Initializable: contract is already initialized");
        batch.initialize(address(0));
    }

    function testSetTokenSetting() external {
        // revert not admin
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0x0000000000000000000000000000000000000000000000000000000000000000"
        );
        batch.setBatchConfig(address(0), L1BatchBridgeGateway.BatchConfig(0, 0, 0, 0, 0));
        hevm.stopPrank();

        // revert maxTxsPerBatch = 0
        hevm.expectRevert(L1BatchBridgeGateway.ErrorInvalidBatchConfig.selector);
        batch.setBatchConfig(address(0), L1BatchBridgeGateway.BatchConfig(0, 0, 0, 0, 0));

        // revert maxDelayPerBatch = 0
        hevm.expectRevert(L1BatchBridgeGateway.ErrorInvalidBatchConfig.selector);
        batch.setBatchConfig(address(0), L1BatchBridgeGateway.BatchConfig(0, 0, 1, 0, 0));

        // revert feeAmountPerTx > minAmountPerTx
        hevm.expectRevert(L1BatchBridgeGateway.ErrorInvalidBatchConfig.selector);
        batch.setBatchConfig(address(0), L1BatchBridgeGateway.BatchConfig(1, 0, 1, 1, 0));

        // succeed
        batch.setBatchConfig(address(0), L1BatchBridgeGateway.BatchConfig(1, 2, 3, 4, 5));
        (
            uint96 feeAmountPerTx,
            uint96 minAmountPerTx,
            uint16 maxTxsPerBatch,
            uint24 maxDelayPerBatch,
            uint24 safeBridgeGasLimit
        ) = batch.configs(address(0));
        assertEq(feeAmountPerTx, 1);
        assertEq(minAmountPerTx, 2);
        assertEq(maxTxsPerBatch, 3);
        assertEq(maxDelayPerBatch, 4);
        assertEq(safeBridgeGasLimit, 5);
    }

    function testSetTokenSettingFuzzing(address token, L1BatchBridgeGateway.BatchConfig memory config) external {
        hevm.assume(config.maxTxsPerBatch > 0);
        hevm.assume(config.maxDelayPerBatch > 0);
        hevm.assume(config.feeAmountPerTx <= config.minAmountPerTx);

        (
            uint96 feeAmountPerTx,
            uint96 minAmountPerTx,
            uint16 maxTxsPerBatch,
            uint24 maxDelayPerBatch,
            uint24 safeBridgeGasLimit
        ) = batch.configs(token);
        assertEq(feeAmountPerTx, 0);
        assertEq(minAmountPerTx, 0);
        assertEq(maxTxsPerBatch, 0);
        assertEq(maxDelayPerBatch, 0);
        assertEq(safeBridgeGasLimit, 0);
        batch.setBatchConfig(token, config);
        (feeAmountPerTx, minAmountPerTx, maxTxsPerBatch, maxDelayPerBatch, safeBridgeGasLimit) = batch.configs(token);
        assertEq(feeAmountPerTx, config.feeAmountPerTx);
        assertEq(minAmountPerTx, config.minAmountPerTx);
        assertEq(maxTxsPerBatch, config.maxTxsPerBatch);
        assertEq(maxDelayPerBatch, config.maxDelayPerBatch);
        assertEq(safeBridgeGasLimit, config.safeBridgeGasLimit);
    }

    function checkBatchState(
        address token,
        uint256 phase,
        L1BatchBridgeGateway.BatchState memory expected
    ) private {
        (uint128 amount, uint64 startTime, uint64 numDeposits, bytes32 hash) = batch.batches(token, phase);
        assertEq(amount, expected.amount);
        assertEq(startTime, expected.startTime);
        assertEq(numDeposits, expected.numDeposits);
        // assertEq(hash, expected.hash);
    }

    function checkTokenState(address token, L1BatchBridgeGateway.TokenState memory expected) private {
        (uint128 pending, uint64 currentBatchIndex, uint64 pendingBatchIndex) = batch.tokens(token);
        assertEq(pending, expected.pending);
        assertEq(currentBatchIndex, expected.currentBatchIndex);
        assertEq(pendingBatchIndex, expected.pendingBatchIndex);
    }

    function testDepositETH() external {
        // revert token not supported
        hevm.expectRevert(L1BatchBridgeGateway.ErrorTokenNotSupported.selector);
        batch.depositETH();

        // revert deposit amount too small
        batch.setBatchConfig(address(0), L1BatchBridgeGateway.BatchConfig(0, 100, 2, 100, ETH_DEPOSIT_SAFE_GAS_LIMIT));
        hevm.expectRevert(L1BatchBridgeGateway.ErrorDepositAmountTooSmall.selector);
        batch.depositETH{value: 10}();

        // no fee
        batch.setBatchConfig(address(0), L1BatchBridgeGateway.BatchConfig(0, 0, 2, 100, ETH_DEPOSIT_SAFE_GAS_LIMIT));
        assertEq(0, address(batch).balance);
        checkBatchState(address(0), 0, L1BatchBridgeGateway.BatchState(0, 0, 0, bytes32(0)));
        checkTokenState(address(0), L1BatchBridgeGateway.TokenState(0, 0, 0));

        hevm.warp(1000001);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(0), 0, 1000, 0);
        batch.depositETH{value: 1000}();
        assertEq(1000, address(batch).balance);
        checkBatchState(address(0), 0, L1BatchBridgeGateway.BatchState(1000, 1000001, 1, bytes32(0)));
        checkTokenState(address(0), L1BatchBridgeGateway.TokenState(1000, 0, 0));

        hevm.warp(1000002);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(0), 0, 2000, 0);
        batch.depositETH{value: 2000}();
        assertEq(3000, address(batch).balance);
        checkBatchState(address(0), 0, L1BatchBridgeGateway.BatchState(3000, 1000001, 2, bytes32(0)));
        checkTokenState(address(0), L1BatchBridgeGateway.TokenState(3000, 0, 0));

        hevm.warp(1000003);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(0), 1, 3000, 0);
        batch.depositETH{value: 3000}();
        assertEq(6000, address(batch).balance);
        checkBatchState(address(0), 1, L1BatchBridgeGateway.BatchState(3000, 1000003, 1, bytes32(0)));
        checkTokenState(address(0), L1BatchBridgeGateway.TokenState(6000, 1, 0));

        // with fee
        batch.setBatchConfig(
            address(0),
            L1BatchBridgeGateway.BatchConfig(100, 1000, 2, 100, ETH_DEPOSIT_SAFE_GAS_LIMIT)
        );

        hevm.warp(1000004);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(0), 1, 1000, 100);
        batch.depositETH{value: 1000}();
        assertEq(7000, address(batch).balance);
        checkBatchState(address(0), 1, L1BatchBridgeGateway.BatchState(3900, 1000003, 2, bytes32(0)));
        checkTokenState(address(0), L1BatchBridgeGateway.TokenState(6900, 1, 0));

        hevm.warp(1000005);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(0), 2, 2000, 100);
        batch.depositETH{value: 2000}();
        assertEq(9000, address(batch).balance);
        checkBatchState(address(0), 2, L1BatchBridgeGateway.BatchState(1900, 1000005, 1, bytes32(0)));
        checkTokenState(address(0), L1BatchBridgeGateway.TokenState(8800, 2, 0));

        hevm.warp(1000006);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(0), 2, 3000, 100);
        batch.depositETH{value: 3000}();
        assertEq(12000, address(batch).balance);
        checkBatchState(address(0), 2, L1BatchBridgeGateway.BatchState(4800, 1000005, 2, bytes32(0)));
        checkTokenState(address(0), L1BatchBridgeGateway.TokenState(11700, 2, 0));

        // switch phase by timestamp
        batch.setBatchConfig(address(0), L1BatchBridgeGateway.BatchConfig(0, 0, 100, 100, ETH_DEPOSIT_SAFE_GAS_LIMIT));

        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(0), 2, 1000, 0);
        batch.depositETH{value: 1000}();
        assertEq(13000, address(batch).balance);
        checkBatchState(address(0), 2, L1BatchBridgeGateway.BatchState(5800, 1000005, 3, bytes32(0)));
        checkTokenState(address(0), L1BatchBridgeGateway.TokenState(12700, 2, 0));

        hevm.warp(1000005 + 100 + 1);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(0), 3, 1000, 0);
        batch.depositETH{value: 1000}();
        assertEq(14000, address(batch).balance);
        checkBatchState(address(0), 3, L1BatchBridgeGateway.BatchState(1000, 1000005 + 100 + 1, 1, bytes32(0)));
        checkTokenState(address(0), L1BatchBridgeGateway.TokenState(13700, 3, 0));
    }

    function testDepositERC20() external {
        // revert token is zero
        hevm.expectRevert(L1BatchBridgeGateway.ErrorIncorrectMethodForETHDeposit.selector);
        batch.depositERC20(address(0), 0);

        // revert token not supported
        hevm.expectRevert(L1BatchBridgeGateway.ErrorTokenNotSupported.selector);
        batch.depositERC20(address(l1Token), 0);

        // revert deposit amount too small
        batch.setBatchConfig(
            address(l1Token),
            L1BatchBridgeGateway.BatchConfig(0, 100, 2, 100, ERC20_DEPOSIT_SAFE_GAS_LIMIT)
        );
        l1Token.approve(address(batch), 10);
        hevm.expectRevert(L1BatchBridgeGateway.ErrorDepositAmountTooSmall.selector);
        batch.depositERC20(address(l1Token), 10);

        // no fee
        batch.setBatchConfig(
            address(l1Token),
            L1BatchBridgeGateway.BatchConfig(0, 0, 2, 100, ERC20_DEPOSIT_SAFE_GAS_LIMIT)
        );
        assertEq(0, l1Token.balanceOf(address(batch)));
        checkBatchState(address(l1Token), 0, L1BatchBridgeGateway.BatchState(0, 0, 0, bytes32(0)));
        checkTokenState(address(l1Token), L1BatchBridgeGateway.TokenState(0, 0, 0));

        hevm.warp(1000001);
        l1Token.approve(address(batch), 1000);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(l1Token), 0, 1000, 0);
        batch.depositERC20(address(l1Token), 1000);
        assertEq(1000, l1Token.balanceOf(address(batch)));
        checkBatchState(address(l1Token), 0, L1BatchBridgeGateway.BatchState(1000, 1000001, 1, bytes32(0)));
        checkTokenState(address(l1Token), L1BatchBridgeGateway.TokenState(1000, 0, 0));

        hevm.warp(1000002);
        l1Token.approve(address(batch), 2000);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(l1Token), 0, 2000, 0);
        batch.depositERC20(address(l1Token), 2000);
        assertEq(3000, l1Token.balanceOf(address(batch)));
        checkBatchState(address(l1Token), 0, L1BatchBridgeGateway.BatchState(3000, 1000001, 2, bytes32(0)));
        checkTokenState(address(l1Token), L1BatchBridgeGateway.TokenState(3000, 0, 0));

        hevm.warp(1000003);
        l1Token.approve(address(batch), 3000);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(l1Token), 1, 3000, 0);
        batch.depositERC20(address(l1Token), 3000);
        assertEq(6000, l1Token.balanceOf(address(batch)));
        checkBatchState(address(l1Token), 1, L1BatchBridgeGateway.BatchState(3000, 1000003, 1, bytes32(0)));
        checkTokenState(address(l1Token), L1BatchBridgeGateway.TokenState(6000, 1, 0));

        // with fee
        batch.setBatchConfig(
            address(l1Token),
            L1BatchBridgeGateway.BatchConfig(100, 1000, 2, 100, ERC20_DEPOSIT_SAFE_GAS_LIMIT)
        );

        hevm.warp(1000004);
        l1Token.approve(address(batch), 1000);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(l1Token), 1, 1000, 100);
        batch.depositERC20(address(l1Token), 1000);
        assertEq(7000, l1Token.balanceOf(address(batch)));
        checkBatchState(address(l1Token), 1, L1BatchBridgeGateway.BatchState(3900, 1000003, 2, bytes32(0)));
        checkTokenState(address(l1Token), L1BatchBridgeGateway.TokenState(6900, 1, 0));

        hevm.warp(1000005);
        l1Token.approve(address(batch), 2000);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(l1Token), 2, 2000, 100);
        batch.depositERC20(address(l1Token), 2000);
        assertEq(9000, l1Token.balanceOf(address(batch)));
        checkBatchState(address(l1Token), 2, L1BatchBridgeGateway.BatchState(1900, 1000005, 1, bytes32(0)));
        checkTokenState(address(l1Token), L1BatchBridgeGateway.TokenState(8800, 2, 0));

        hevm.warp(1000006);
        l1Token.approve(address(batch), 3000);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(l1Token), 2, 3000, 100);
        batch.depositERC20(address(l1Token), 3000);
        assertEq(12000, l1Token.balanceOf(address(batch)));
        checkBatchState(address(l1Token), 2, L1BatchBridgeGateway.BatchState(4800, 1000005, 2, bytes32(0)));
        checkTokenState(address(l1Token), L1BatchBridgeGateway.TokenState(11700, 2, 0));

        // switch phase by timestamp
        batch.setBatchConfig(
            address(l1Token),
            L1BatchBridgeGateway.BatchConfig(0, 0, 100, 100, ERC20_DEPOSIT_SAFE_GAS_LIMIT)
        );

        l1Token.approve(address(batch), 1000);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(l1Token), 2, 1000, 0);
        batch.depositERC20(address(l1Token), 1000);
        assertEq(13000, l1Token.balanceOf(address(batch)));
        checkBatchState(address(l1Token), 2, L1BatchBridgeGateway.BatchState(5800, 1000005, 3, bytes32(0)));
        checkTokenState(address(l1Token), L1BatchBridgeGateway.TokenState(12700, 2, 0));

        hevm.warp(1000005 + 100 + 1);
        l1Token.approve(address(batch), 1000);
        hevm.expectEmit(true, true, true, true);
        emit Deposit(address(this), address(l1Token), 3, 1000, 0);
        batch.depositERC20(address(l1Token), 1000);
        assertEq(14000, l1Token.balanceOf(address(batch)));
        checkBatchState(address(l1Token), 3, L1BatchBridgeGateway.BatchState(1000, 1000005 + 100 + 1, 1, bytes32(0)));
        checkTokenState(address(l1Token), L1BatchBridgeGateway.TokenState(13700, 3, 0));
    }

    function testBatchBridgeFailure() external {
        // revert not keeper
        hevm.startPrank(address(1));
        hevm.expectRevert(
            "AccessControl: account 0x0000000000000000000000000000000000000001 is missing role 0xfc8737ab85eb45125971625a9ebdb75cc78e01d5c1fa80c4c6e5203f47bc4fab"
        );
        batch.batchDeposit(address(0));
        hevm.stopPrank();

        batch.grantRole(batch.KEEPER_ROLE(), address(this));

        // revert token not supported
        hevm.expectRevert(L1BatchBridgeGateway.ErrorTokenNotSupported.selector);
        batch.batchDeposit(address(0));

        batch.setBatchConfig(address(0), L1BatchBridgeGateway.BatchConfig(0, 0, 1, 1, ETH_DEPOSIT_SAFE_GAS_LIMIT));

        // revert no pending
        hevm.expectRevert(L1BatchBridgeGateway.ErrorNoPendingBatch.selector);
        batch.batchDeposit(address(0));

        // revert insufficient msg.value
        batch.depositETH{value: 1000}();
        hevm.expectRevert(L1BatchBridgeGateway.ErrorInsufficientMsgValueForBatchDepositFee.selector);
        batch.batchDeposit(address(0));

        hevm.expectRevert(L1BatchBridgeGateway.ErrorInsufficientMsgValueForBatchDepositFee.selector);
        batch.batchDeposit{value: L2_GAS_PRICE * ETH_DEPOSIT_SAFE_GAS_LIMIT}(address(0));

        hevm.expectRevert(L1BatchBridgeGateway.ErrorInsufficientMsgValueForBatchDepositFee.selector);
        batch.batchDeposit{value: L2_GAS_PRICE * (SAFE_BATCH_BRIDGE_GAS_LIMIT + ETH_DEPOSIT_SAFE_GAS_LIMIT) - 1}(
            address(0)
        );

        // succeed
        batch.batchDeposit{value: L2_GAS_PRICE * (SAFE_BATCH_BRIDGE_GAS_LIMIT + ETH_DEPOSIT_SAFE_GAS_LIMIT)}(
            address(0)
        );
    }

    function testBatchBridgeETH() external {
        batch.grantRole(batch.KEEPER_ROLE(), address(this));

        // no deposit fee
        batch.setBatchConfig(address(0), L1BatchBridgeGateway.BatchConfig(0, 0, 1, 1, ETH_DEPOSIT_SAFE_GAS_LIMIT));
        batch.depositETH{value: 1000}();
        checkTokenState(address(0), L1BatchBridgeGateway.TokenState(1000, 0, 0));

        // emit SentMessage by deposit ETH
        hevm.expectEmit(true, true, false, true);
        emit SentMessage(address(batch), address(counterpartBatch), 1000, 0, ETH_DEPOSIT_SAFE_GAS_LIMIT, "");

        // emit SentMessage by batchBridge
        hevm.expectEmit(true, true, false, true);
        emit SentMessage(
            address(batch),
            address(counterpartBatch),
            0,
            1,
            SAFE_BATCH_BRIDGE_GAS_LIMIT,
            abi.encodeCall(
                L2BatchBridgeGateway.finalizeBatchDeposit,
                (
                    address(0),
                    address(0),
                    0,
                    BatchBridgeCodec.hash(
                        BatchBridgeCodec.encodeInitialNode(address(0), 0),
                        BatchBridgeCodec.encodeNode(address(this), 1000)
                    )
                )
            )
        );

        // emit BatchBridge
        hevm.expectEmit(true, true, true, true);
        emit BatchDeposit(address(this), address(0), 0, address(0));

        uint256 batchFeeVaultBefore = batchFeeVault.balance;
        uint256 messengerBefore = address(l1Messenger).balance;
        batch.batchDeposit{value: 1 ether}(address(0));
        checkTokenState(address(0), L1BatchBridgeGateway.TokenState(0, 1, 1));
        assertEq(batchFeeVaultBefore, batchFeeVault.balance);
        assertEq(messengerBefore + 1000, address(l1Messenger).balance);

        // has deposit fee = 100
        batch.setBatchConfig(address(0), L1BatchBridgeGateway.BatchConfig(100, 1000, 1, 1, ETH_DEPOSIT_SAFE_GAS_LIMIT));

        batch.depositETH{value: 1000}();
        checkTokenState(address(0), L1BatchBridgeGateway.TokenState(900, 1, 1));

        // emit SentMessage by deposit ETH
        hevm.expectEmit(true, true, false, true);
        emit SentMessage(address(batch), address(counterpartBatch), 900, 2, ETH_DEPOSIT_SAFE_GAS_LIMIT, "");

        // emit SentMessage by batchBridge
        hevm.expectEmit(true, true, false, true);
        emit SentMessage(
            address(batch),
            address(counterpartBatch),
            0,
            3,
            SAFE_BATCH_BRIDGE_GAS_LIMIT,
            abi.encodeCall(
                L2BatchBridgeGateway.finalizeBatchDeposit,
                (
                    address(0),
                    address(0),
                    1,
                    BatchBridgeCodec.hash(
                        BatchBridgeCodec.encodeInitialNode(address(0), 1),
                        BatchBridgeCodec.encodeNode(address(this), 900)
                    )
                )
            )
        );

        // emit BatchBridge
        hevm.expectEmit(true, true, true, true);
        emit BatchDeposit(address(this), address(0), 1, address(0));

        batchFeeVaultBefore = batchFeeVault.balance;
        messengerBefore = address(l1Messenger).balance;
        batch.batchDeposit{value: 1 ether}(address(0));
        checkTokenState(address(0), L1BatchBridgeGateway.TokenState(0, 2, 2));
        assertEq(batchFeeVaultBefore + 100, batchFeeVault.balance);
        assertEq(messengerBefore + 900, address(l1Messenger).balance);
    }

    function testBatchBridgeERC20() external {
        batch.grantRole(batch.KEEPER_ROLE(), address(this));

        // no deposit fee
        batch.setBatchConfig(
            address(l1Token),
            L1BatchBridgeGateway.BatchConfig(0, 0, 1, 1, ERC20_DEPOSIT_SAFE_GAS_LIMIT)
        );
        l1Token.approve(address(batch), 1000);
        batch.depositERC20(address(l1Token), 1000);
        checkTokenState(address(l1Token), L1BatchBridgeGateway.TokenState(1000, 0, 0));

        bytes memory message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1Token),
            address(l2Token),
            address(batch),
            address(counterpartBatch),
            1000,
            new bytes(0)
        );
        // emit SentMessage by deposit ERC20
        hevm.expectEmit(true, true, false, true);
        emit SentMessage(address(gateway), address(counterpartGateway), 0, 0, ERC20_DEPOSIT_SAFE_GAS_LIMIT, message);
        // emit SentMessage by batchBridge
        hevm.expectEmit(true, true, false, true);
        emit SentMessage(
            address(batch),
            address(counterpartBatch),
            0,
            1,
            SAFE_BATCH_BRIDGE_GAS_LIMIT,
            abi.encodeCall(
                L2BatchBridgeGateway.finalizeBatchDeposit,
                (
                    address(l1Token),
                    address(l2Token),
                    0,
                    BatchBridgeCodec.hash(
                        BatchBridgeCodec.encodeInitialNode(address(l1Token), 0),
                        BatchBridgeCodec.encodeNode(address(this), 1000)
                    )
                )
            )
        );
        // emit BatchBridge
        hevm.expectEmit(true, true, true, true);
        emit BatchDeposit(address(this), address(l1Token), 0, address(l2Token));

        uint256 batchFeeVaultBefore = l1Token.balanceOf(batchFeeVault);
        uint256 gatewayBefore = l1Token.balanceOf(address(gateway));
        batch.batchDeposit{value: 1 ether}(address(l1Token));
        checkTokenState(address(l1Token), L1BatchBridgeGateway.TokenState(0, 1, 1));
        assertEq(batchFeeVaultBefore, l1Token.balanceOf(batchFeeVault));
        assertEq(gatewayBefore + 1000, l1Token.balanceOf(address(gateway)));

        // has deposit fee = 100
        batch.setBatchConfig(
            address(l1Token),
            L1BatchBridgeGateway.BatchConfig(100, 1000, 1, 1, ERC20_DEPOSIT_SAFE_GAS_LIMIT)
        );

        l1Token.approve(address(batch), 1000);
        batch.depositERC20(address(l1Token), 1000);
        checkTokenState(address(l1Token), L1BatchBridgeGateway.TokenState(900, 1, 1));

        message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            address(l1Token),
            address(l2Token),
            address(batch),
            address(counterpartBatch),
            900,
            new bytes(0)
        );
        // emit SentMessage by deposit ERC20
        hevm.expectEmit(true, true, false, true);
        emit SentMessage(address(gateway), address(counterpartGateway), 0, 2, ERC20_DEPOSIT_SAFE_GAS_LIMIT, message);
        // emit SentMessage by batchBridge
        hevm.expectEmit(true, true, false, true);
        emit SentMessage(
            address(batch),
            address(counterpartBatch),
            0,
            3,
            SAFE_BATCH_BRIDGE_GAS_LIMIT,
            abi.encodeCall(
                L2BatchBridgeGateway.finalizeBatchDeposit,
                (
                    address(l1Token),
                    address(l2Token),
                    1,
                    BatchBridgeCodec.hash(
                        BatchBridgeCodec.encodeInitialNode(address(l1Token), 1),
                        BatchBridgeCodec.encodeNode(address(this), 900)
                    )
                )
            )
        );
        // emit BatchBridge
        hevm.expectEmit(true, true, true, true);
        emit BatchDeposit(address(this), address(l1Token), 1, address(l2Token));

        batchFeeVaultBefore = l1Token.balanceOf(batchFeeVault);
        gatewayBefore = l1Token.balanceOf(address(gateway));
        batch.batchDeposit{value: 1 ether}(address(l1Token));
        checkTokenState(address(l1Token), L1BatchBridgeGateway.TokenState(0, 2, 2));
        assertEq(batchFeeVaultBefore + 100, l1Token.balanceOf(batchFeeVault));
        assertEq(gatewayBefore + 900, l1Token.balanceOf(address(gateway)));
    }
}
