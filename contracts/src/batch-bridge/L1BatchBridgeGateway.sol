// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {AccessControlEnumerableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/AccessControlEnumerableUpgradeable.sol";
import {ReentrancyGuardUpgradeable} from "@openzeppelin/contracts-upgradeable/security/ReentrancyGuardUpgradeable.sol";
import {SafeERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";
import {IERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";
import {AddressUpgradeable} from "@openzeppelin/contracts-upgradeable/utils/AddressUpgradeable.sol";

import {IL1ERC20Gateway} from "../L1/gateways/IL1ERC20Gateway.sol";
import {IL1GatewayRouter} from "../L1/gateways/IL1GatewayRouter.sol";
import {IL1MessageQueue} from "../L1/rollup/IL1MessageQueue.sol";
import {IL1ScrollMessenger} from "../L1/IL1ScrollMessenger.sol";

import {BatchBridgeCodec} from "./BatchBridgeCodec.sol";
import {L2BatchBridgeGateway} from "./L2BatchBridgeGateway.sol";

/// @title L1BatchBridgeGateway
contract L1BatchBridgeGateway is AccessControlEnumerableUpgradeable, ReentrancyGuardUpgradeable {
    using SafeERC20Upgradeable for IERC20Upgradeable;

    /**********
     * Events *
     **********/

    /// @notice Emitted when some user deposited token to this contract.
    /// @param sender The address of token sender.
    /// @param token The address of deposited token.
    /// @param batchIndex The batch index of current deposit.
    /// @param amount The amount of token deposited (including fee).
    /// @param fee The amount of fee charged.
    event Deposit(
        address indexed sender,
        address indexed token,
        uint256 indexed batchIndex,
        uint256 amount,
        uint256 fee
    );

    /// @notice Emitted when a batch deposit is initiated.
    /// @param caller The address of caller who initiate the deposit.
    /// @param l1Token The address of the token in L1 to deposit.
    /// @param batchIndex The index of current batch deposit.
    /// @param l2Token The address of the corresponding token in L2.
    event BatchDeposit(address indexed caller, address indexed l1Token, uint256 indexed batchIndex, address l2Token);

    /**********
     * Errors *
     **********/

    /// @dev Thrown when caller is not `messenger`.
    error ErrorCallerNotMessenger();

    /// @dev Thrown when the deposited amount is smaller than `minAmountPerTx`.
    error ErrorDepositAmountTooSmall();

    /// @dev Thrown when users try to deposit ETH with `depositERC20` method.
    error ErrorIncorrectMethodForETHDeposit();

    /// @dev Thrown when the `msg.value` is not enough for batch deposit fee.
    error ErrorInsufficientMsgValueForBatchDepositFee();

    /// @dev Thrown when the given new batch config is invalid.
    error ErrorInvalidBatchConfig();

    /// @dev Thrown when no pending batch exists.
    error ErrorNoPendingBatch();

    /// @dev Thrown when user deposits unsupported tokens.
    error ErrorTokenNotSupported();

    /// @dev Thrown when ETH transfer failed.
    error ErrorTransferETHFailed();

    /*************
     * Constants *
     *************/

    /// @notice The role for batch deposit keeper.
    bytes32 public constant KEEPER_ROLE = keccak256("KEEPER_ROLE");

    /// @notice The safe gas limit for batch bridge.
    uint256 private constant SAFE_BATCH_BRIDGE_GAS_LIMIT = 200000;

    /// @notice The address of corresponding `L2BatchDepositGateway` contract.
    address public immutable counterpart;

    /// @notice The address of `L1GatewayRouter` contract.
    address public immutable router;

    /// @notice The address of `L1ScrollMessenger` contract.
    address public immutable messenger;

    /// @notice The address of `L1MessageQueue` contract.
    address public immutable queue;

    /***********
     * Structs *
     ***********/

    /// @notice The config for batch token bridge.
    /// @dev Compiler will pack this into a single `bytes32`.
    /// @param feeAmountPerTx The amount of fee charged for each deposit.
    /// @param minAmountPerTx The minimum amount of token for each deposit.
    /// @param maxTxsPerBatch The maximum number of deposit in each batch.
    /// @param maxDelayPerBatch The maximum number of seconds to wait in each batch.
    /// @param safeBridgeGasLimit The safe bridge gas limit for bridging token from L1 to L2.
    struct BatchConfig {
        uint96 feeAmountPerTx;
        uint96 minAmountPerTx;
        uint16 maxTxsPerBatch;
        uint24 maxDelayPerBatch;
        uint24 safeBridgeGasLimit;
    }

    /// @dev Compiler will pack this into two `bytes32`.
    /// @param amount The total amount of token to deposit in current batch.
    /// @param startTime The timestamp of the first deposit.
    /// @param numDeposits The total number of deposits in current batch.
    /// @param hash The hash of current batch.
    ///   Suppose there are `n` deposits in current batch with `senders` and `amounts`. The hash is computed as
    ///   ```text
    ///   hash[0] = concat(token, batch_index)
    ///   hash[i] = keccak(hash[i-1], concat(senders[i], amounts[i]))
    ///   ```
    ///   The type of `token` and `senders` is `address`, while The type of `batch_index` and `amounts[i]` is `uint96`.
    ///   In current way, the hash of each batch among all tokens should be different.
    struct BatchState {
        uint128 amount;
        uint64 startTime;
        uint64 numDeposits;
        bytes32 hash;
    }

    /// @dev Compiler will pack this into a single `bytes32`.
    /// @param pending The total amount of token pending to bridge.
    /// @param currentBatchIndex The index of current batch.
    /// @param pendingBatchIndex The index of pending batch (next batch to bridge).
    struct TokenState {
        uint128 pending;
        uint64 currentBatchIndex;
        uint64 pendingBatchIndex;
    }

    /*************
     * Variables *
     *************/

    /// @notice Mapping from token address to batch bridge config.
    /// @dev The `address(0)` is used for ETH.
    mapping(address => BatchConfig) public configs;

    /// @notice Mapping from token address to batch index to batch state.
    /// @dev The `address(0)` is used for ETH.
    mapping(address => mapping(uint256 => BatchState)) public batches;

    /// @notice Mapping from token address to token state.
    /// @dev The `address(0)` is used for ETH.
    mapping(address => TokenState) public tokens;

    /// @notice The address of fee vault.
    address public feeVault;

    /***************
     * Constructor *
     ***************/

    /// @param _counterpart The address of `L2BatchDepositGateway` contract in L2.
    /// @param _router The address of `L1GatewayRouter` contract in L1.
    /// @param _messenger The address of `L1ScrollMessenger` contract in L1.
    /// @param _queue The address of `L1MessageQueue` contract in L1.
    constructor(
        address _counterpart,
        address _router,
        address _messenger,
        address _queue
    ) {
        _disableInitializers();

        counterpart = _counterpart;
        router = _router;
        messenger = _messenger;
        queue = _queue;
    }

    /// @notice Initialize the storage of `L1BatchDepositGateway`.
    /// @param _feeVault The address of fee vault contract.
    function initialize(address _feeVault) external initializer {
        __Context_init(); // from ContextUpgradeable
        __ERC165_init(); // from ERC165Upgradeable
        __AccessControl_init(); // from AccessControlUpgradeable
        __AccessControlEnumerable_init(); // from AccessControlEnumerableUpgradeable
        __ReentrancyGuard_init(); // from ReentrancyGuardUpgradeable

        feeVault = _feeVault;
        _grantRole(DEFAULT_ADMIN_ROLE, _msgSender());
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Receive refunded ETH from `L1ScrollMessenger`.
    receive() external payable {
        if (_msgSender() != messenger) {
            revert ErrorCallerNotMessenger();
        }
    }

    /// @notice Deposit ETH.
    function depositETH() external payable {
        // no safe cast check here, since no one has so much ETH yet.
        _deposit(address(0), _msgSender(), uint96(msg.value));
    }

    /// @notice Deposit ERC20 token.
    ///
    /// @param token The address of token.
    /// @param amount The amount of token to deposit. We use type `uint96`, since it is enough for most of the major tokens.
    function depositERC20(address token, uint96 amount) external {
        if (token == address(0)) revert ErrorIncorrectMethodForETHDeposit();

        // common practice to handle fee on transfer token.
        uint256 beforeBalance = IERC20Upgradeable(token).balanceOf(address(this));
        IERC20Upgradeable(token).safeTransferFrom(_msgSender(), address(this), amount);
        amount = uint96(IERC20Upgradeable(token).balanceOf(address(this)) - beforeBalance);

        _deposit(token, _msgSender(), amount);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Add or update the batch bridge config for the given token.
    ///
    /// @dev The caller should make sure `safeBridgeGasLimit` is enough for batch bridging.
    ///
    /// @param token The address of token to update.
    /// @param newConfig The new config.
    function setBatchConfig(address token, BatchConfig memory newConfig) external onlyRole(DEFAULT_ADMIN_ROLE) {
        if (
            newConfig.maxTxsPerBatch == 0 ||
            newConfig.maxDelayPerBatch == 0 ||
            newConfig.feeAmountPerTx > newConfig.minAmountPerTx
        ) {
            revert ErrorInvalidBatchConfig();
        }
        configs[token] = newConfig;
    }

    /// @notice Initiate the batch bridge of current pending batch.
    /// @param token The address of the token.
    function executeBatchDeposit(address token) external payable onlyRole(KEEPER_ROLE) {
        BatchConfig memory cachedBatchConfig = configs[token];
        TokenState memory cachedTokenState = tokens[token];
        _tryFinalizeCurrentBatch(token, cachedBatchConfig, cachedTokenState);

        // no batch to bridge
        if (cachedTokenState.currentBatchIndex == cachedTokenState.pendingBatchIndex) {
            revert ErrorNoPendingBatch();
        }

        // check bridge fee
        uint256 depositFee = IL1MessageQueue(queue).estimateCrossDomainMessageFee(cachedBatchConfig.safeBridgeGasLimit);
        uint256 batchBridgeFee = IL1MessageQueue(queue).estimateCrossDomainMessageFee(SAFE_BATCH_BRIDGE_GAS_LIMIT);
        if (msg.value < depositFee + batchBridgeFee) {
            revert ErrorInsufficientMsgValueForBatchDepositFee();
        }

        // take accumulated fee to fee vault
        uint256 accumulatedFee;
        if (token == address(0)) {
            // no uncheck here just in case
            accumulatedFee = address(this).balance - msg.value - cachedTokenState.pending;
        } else {
            // no uncheck here just in case
            accumulatedFee = IERC20Upgradeable(token).balanceOf(address(this)) - cachedTokenState.pending;
        }
        if (accumulatedFee > 0) {
            _transferToken(token, feeVault, accumulatedFee);
        }

        // deposit token to L2
        BatchState memory cachedBatchState = batches[token][cachedTokenState.pendingBatchIndex];
        address l2Token;
        if (token == address(0)) {
            IL1ScrollMessenger(messenger).sendMessage{value: cachedBatchState.amount + depositFee}(
                counterpart,
                cachedBatchState.amount,
                new bytes(0),
                cachedBatchConfig.safeBridgeGasLimit
            );
        } else {
            address gateway = IL1GatewayRouter(router).getERC20Gateway(token);
            l2Token = IL1ERC20Gateway(gateway).getL2ERC20Address(token);
            IERC20Upgradeable(token).safeApprove(gateway, 0);
            IERC20Upgradeable(token).safeApprove(gateway, cachedBatchState.amount);
            IL1ERC20Gateway(gateway).depositERC20{value: depositFee}(
                token,
                counterpart,
                cachedBatchState.amount,
                cachedBatchConfig.safeBridgeGasLimit
            );
        }

        // notify `L2BatchBridgeGateway`
        IL1ScrollMessenger(messenger).sendMessage{value: batchBridgeFee}(
            counterpart,
            0,
            abi.encodeCall(
                L2BatchBridgeGateway.finalizeBatchDeposit,
                (token, l2Token, cachedTokenState.pendingBatchIndex, cachedBatchState.hash)
            ),
            SAFE_BATCH_BRIDGE_GAS_LIMIT
        );

        emit BatchDeposit(_msgSender(), token, cachedTokenState.pendingBatchIndex, l2Token);

        // update token state
        unchecked {
            cachedTokenState.pending -= uint128(cachedBatchState.amount);
            cachedTokenState.pendingBatchIndex += 1;
        }
        tokens[token] = cachedTokenState;

        // refund keeper fee
        unchecked {
            if (msg.value > depositFee + batchBridgeFee) {
                _transferToken(address(0), _msgSender(), msg.value - depositFee - batchBridgeFee);
            }
        }
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to deposit token.
    /// @param token The address of token to deposit.
    /// @param sender The address of token sender.
    /// @param amount The amount of token to deposit.
    function _deposit(
        address token,
        address sender,
        uint96 amount
    ) internal nonReentrant {
        BatchConfig memory cachedBatchConfig = configs[token];
        TokenState memory cachedTokenState = tokens[token];
        _tryFinalizeCurrentBatch(token, cachedBatchConfig, cachedTokenState);

        BatchState memory cachedBatchState = batches[token][cachedTokenState.currentBatchIndex];
        if (amount < cachedBatchConfig.minAmountPerTx) {
            revert ErrorDepositAmountTooSmall();
        }

        emit Deposit(sender, token, cachedTokenState.currentBatchIndex, amount, cachedBatchConfig.feeAmountPerTx);

        // deduct fee and update cached state
        unchecked {
            amount -= cachedBatchConfig.feeAmountPerTx;
            cachedTokenState.pending += amount;
            cachedBatchState.amount += amount;
            cachedBatchState.numDeposits += 1;
        }

        // compute the hash chain
        bytes32 node = BatchBridgeCodec.encodeNode(sender, amount);
        if (cachedBatchState.hash == bytes32(0)) {
            bytes32 initialNode = BatchBridgeCodec.encodeInitialNode(token, cachedTokenState.currentBatchIndex);
            // this is first tx in this batch
            cachedBatchState.hash = BatchBridgeCodec.hash(initialNode, node);
            cachedBatchState.startTime = uint64(block.timestamp);
        } else {
            cachedBatchState.hash = BatchBridgeCodec.hash(cachedBatchState.hash, node);
        }

        batches[token][cachedTokenState.currentBatchIndex] = cachedBatchState;
        tokens[token] = cachedTokenState;
    }

    /// @dev Internal function to finalize current batch.
    ///      This function may change the value of `cachedTokenState`, which can be used in later operation.
    /// @param token The address of token to finalize.
    /// @param cachedBatchConfig The cached batch config in memory.
    /// @param cachedTokenState The cached token state in memory.
    function _tryFinalizeCurrentBatch(
        address token,
        BatchConfig memory cachedBatchConfig,
        TokenState memory cachedTokenState
    ) internal view {
        if (cachedBatchConfig.maxTxsPerBatch == 0) {
            revert ErrorTokenNotSupported();
        }
        BatchState memory cachedBatchState = batches[token][cachedTokenState.currentBatchIndex];
        // return if it is the very first batch
        if (cachedBatchState.numDeposits == 0) return;

        // finalize current batchIndex when `maxTxsPerBatch` or `maxDelayPerBatch` reached.
        if (
            cachedBatchState.numDeposits == cachedBatchConfig.maxTxsPerBatch ||
            block.timestamp - cachedBatchState.startTime > cachedBatchConfig.maxDelayPerBatch
        ) {
            cachedTokenState.currentBatchIndex += 1;
        }
    }

    /// @dev Internal function to transfer token, including ETH.
    /// @param token The address of token.
    /// @param receiver The address of token receiver.
    /// @param amount The amount of token to transfer.
    function _transferToken(
        address token,
        address receiver,
        uint256 amount
    ) private {
        if (token == address(0)) {
            (bool success, ) = receiver.call{value: amount}("");
            if (!success) revert ErrorTransferETHFailed();
        } else {
            IERC20Upgradeable(token).safeTransfer(receiver, amount);
        }
    }
}
