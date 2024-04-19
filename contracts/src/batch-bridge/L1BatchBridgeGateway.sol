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

import {L2BatchBridgeGateway} from "./L2BatchBridgeGateway.sol";

/// @title L1BatchBridgeGateway
contract L1BatchBridgeGateway is AccessControlEnumerableUpgradeable, ReentrancyGuardUpgradeable {
    using SafeERC20Upgradeable for IERC20Upgradeable;

    /**********
     * Events *
     **********/

    event Deposit(address indexed token, uint256 indexed phase, uint256 amount, uint256 fee);

    event BatchBridge(address indexed l1Token, address indexed l2Token, uint256 indexed phase);

    /*************
     * Constants *
     *************/

    /// @notice The role for batch deposit keeper.
    bytes32 public constant KEEPER_ROLE = keccak256("KEEPER_ROLE");

    /// @notice The safe gas limit for batch bridge.
    uint256 private constant SAFE_BATCH_BRIDGE_GAS_LIMIT = 1000000;

    /// @notice The address of corresponding `L2BatchBridgeGateway` contract.
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

    /// @notice The setting for batch token bridging.
    /// @dev Compiler will pack this into a single `bytes32`.
    /// @param feeAmountPerTx The amount of fee charged for each deposit.
    /// @param minAmountPerTx The minimum amount of token for each deposit.
    /// @param maxTxPerBatch The maximum number of deposit in each batch.
    /// @param maxDelayPerBatch The maximum number of seconds to wait in each batch.
    /// @param safeBridgeGasLimit The safe bridge gas limit for bridging token from L1 to L2.
    struct BatchSetting {
        uint96 feeAmountPerTx;
        uint96 minAmountPerTx;
        uint16 maxTxPerBatch;
        uint24 maxDelayPerBatch;
        uint24 safeBridgeGasLimit;
    }

    /// @dev Compiler will pack this into two `bytes32`.
    /// @param amount The total amount of token to deposit in current phase.
    /// @param firstDepositTimestamp The timestamp of first deposit.
    /// @param numDeposits The total number of deposits in current phase.
    /// @param hash The hash of current phase.
    ///   Suppose there are `n` deposits in current phase with `senders` and `amounts`. The hash is computed as
    ///   ```text
    ///   hash[0] = concat(token, phase_index)
    ///   hash[i] = keccak(hash[i-1], concat(senders[i], amounts[i]))
    ///   ```
    ///   The type of `token` and `senders` is `address`, while The type of `phase_index` and `amounts[i]` is `uint96`.
    ///   In current way, the hash of each phase should be different.
    struct PhaseState {
        uint256 amount;
        uint256 firstDepositTimestamp;
        uint256 numDeposits;
        bytes32 hash;
    }

    /// @dev Compiler will pack this into a single `bytes32`.
    struct TokenState {
        uint128 pending;
        uint64 currentPhaseIndex;
        uint64 pendingPhaseIndex;
    }

    /*************
     * Variables *
     *************/

    /// @notice Mapping from token address to batch deposit setting.
    /// @dev The `address(0)` is used for ETH.
    mapping(address => BatchSetting) public settings;

    /// @notice Mapping from token address to phase index to phase state.
    /// @dev The `address(0)` is used for ETH.
    mapping(address => mapping(uint256 => PhaseState)) public phases;

    /// @notice Mapping from token address to token state.
    /// @dev The `address(0)` is used for ETH.
    mapping(address => TokenState) public tokens;

    /// @notice The address of fee vault.
    address public feeVault;

    /***************
     * Constructor *
     ***************/

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

    receive() external payable {}

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
        if (token == address(0)) revert();

        // common practice to handle fee on transfer token.
        uint256 beforeBalance = IERC20Upgradeable(token).balanceOf(address(this));
        IERC20Upgradeable(token).safeTransferFrom(_msgSender(), address(this), amount);
        amount = uint96(IERC20Upgradeable(token).balanceOf(address(this)) - beforeBalance);

        _deposit(token, _msgSender(), amount);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Add or update the batch deposit setting for the given token.
    /// @param token The address of token to update.
    /// @param newSetting The new setting.
    function setTokenSetting(address token, BatchSetting memory newSetting) external onlyRole(DEFAULT_ADMIN_ROLE) {
        settings[token] = newSetting;
    }

    /// @notice Initiate the batch bridge of current pending phase.
    /// @param token The address of the token.
    function batchBridge(address token) external payable onlyRole(KEEPER_ROLE) {
        BatchSetting memory cachedBatchSetting = settings[token];
        TokenState memory cachedTokenState = tokens[token];
        _tryFinalzeCurrentPhase(token, cachedBatchSetting, cachedTokenState);

        // no phase to bridge
        if (cachedTokenState.currentPhaseIndex == cachedTokenState.pendingPhaseIndex) revert();

        // check bridge fee
        uint256 depositFee = IL1MessageQueue(queue).estimateCrossDomainMessageFee(
            cachedBatchSetting.safeBridgeGasLimit
        );
        uint256 batchBridgeFee = IL1MessageQueue(queue).estimateCrossDomainMessageFee(SAFE_BATCH_BRIDGE_GAS_LIMIT);
        if (msg.value < depositFee + batchBridgeFee) revert();

        // take accumulated fee
        uint256 accumulatedFee;
        if (token == address(0)) {
            accumulatedFee = address(this).balance - msg.value - cachedTokenState.pending;
        } else {
            accumulatedFee = IERC20Upgradeable(token).balanceOf(address(this)) - cachedTokenState.pending;
        }
        _transferToken(token, feeVault, accumulatedFee);

        // deposit token to L2
        PhaseState memory cachedPhaseState = phases[token][cachedTokenState.pendingPhaseIndex];
        address l2Token;
        if (token == address(0)) {
            IL1ScrollMessenger(messenger).sendMessage{value: cachedPhaseState.amount + depositFee}(
                counterpart,
                cachedPhaseState.amount,
                new bytes(0),
                cachedBatchSetting.safeBridgeGasLimit
            );
        } else {
            address gateway = IL1GatewayRouter(router).getERC20Gateway(token);
            l2Token = IL1ERC20Gateway(gateway).getL2ERC20Address(token);
            IERC20Upgradeable(token).safeApprove(gateway, 0);
            IERC20Upgradeable(token).safeApprove(gateway, cachedPhaseState.amount);
            IL1ERC20Gateway(gateway).depositERC20{value: depositFee}(
                token,
                cachedPhaseState.amount,
                cachedBatchSetting.safeBridgeGasLimit
            );
        }

        // notify `L2BatchBridgeGateway`
        IL1ScrollMessenger(messenger).sendMessage{value: batchBridgeFee}(
            counterpart,
            0,
            abi.encodeCall(
                L2BatchBridgeGateway.finalizeBatchBridge,
                (token, l2Token, cachedTokenState.pendingPhaseIndex, cachedPhaseState.hash)
            ),
            SAFE_BATCH_BRIDGE_GAS_LIMIT
        );

        emit BatchBridge(token, l2Token, cachedTokenState.pendingPhaseIndex);

        // update token state
        unchecked {
            cachedTokenState.pending -= uint128(cachedPhaseState.amount);
            cachedTokenState.pendingPhaseIndex += 1;
        }
        tokens[token] = cachedTokenState;

        // refund keeper fee
        if (msg.value > depositFee + batchBridgeFee) {
            unchecked {
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
        BatchSetting memory cachedBatchSetting = settings[token];
        TokenState memory cachedTokenState = tokens[token];
        _tryFinalzeCurrentPhase(token, cachedBatchSetting, cachedTokenState);

        PhaseState memory cachedPhaseState = phases[token][cachedTokenState.currentPhaseIndex];
        if (amount < cachedBatchSetting.minAmountPerTx) revert();

        // deduct fee and update cached state
        unchecked {
            amount -= cachedBatchSetting.feeAmountPerTx;
            cachedTokenState.pending += amount;
            cachedPhaseState.amount += amount;
            cachedPhaseState.numDeposits += 1;
        }

        // compute the hash chain
        bytes32 node;
        assembly {
            node := add(shl(96, sender), amount)
        }
        if (cachedPhaseState.hash == bytes32(0)) {
            uint256 currentPhaseIndex = cachedTokenState.currentPhaseIndex;
            bytes32 initialNode;
            assembly {
                initialNode := add(shl(96, token), currentPhaseIndex)
            }
            // this is first tx in this phase
            cachedPhaseState.hash = _efficientHash(initialNode, node);
            cachedPhaseState.firstDepositTimestamp = block.timestamp;
        } else {
            cachedPhaseState.hash = _efficientHash(cachedPhaseState.hash, node);
        }

        phases[token][cachedTokenState.currentPhaseIndex] = cachedPhaseState;
        tokens[token] = cachedTokenState;
    }

    /// @dev Internal function to finalze current phase.
    ///      This function may change the value of `cachedTokenState`, which can be used in later operation.
    /// @param token The address of token to finalze.
    /// @param cachedBatchSetting The cached batch setting in memory.
    /// @param cachedTokenState The cached token state in memory.
    function _tryFinalzeCurrentPhase(
        address token,
        BatchSetting memory cachedBatchSetting,
        TokenState memory cachedTokenState
    ) internal view {
        if (cachedBatchSetting.maxTxPerBatch == 0) revert();
        PhaseState memory cachedPhaseState = phases[token][cachedTokenState.currentPhaseIndex];
        if (cachedPhaseState.numDeposits == 0) return;

        // finalize current phase when `maxTxPerBatch` or `maxDelayPerBatch` reached.
        if (
            cachedPhaseState.numDeposits == cachedBatchSetting.maxTxPerBatch ||
            block.timestamp - cachedPhaseState.firstDepositTimestamp > cachedBatchSetting.maxDelayPerBatch
        ) {
            cachedTokenState.currentPhaseIndex += 1;
        }
    }

    /// @dev Internal function to compute `keccak256(concat(a, b))`.
    function _efficientHash(bytes32 a, bytes32 b) private pure returns (bytes32 value) {
        // solhint-disable-next-line no-inline-assembly
        assembly {
            mstore(0x00, a)
            mstore(0x20, b)
            value := keccak256(0x00, 0x40)
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
            if (!success) revert();
        } else {
            IERC20Upgradeable(token).safeTransfer(receiver, amount);
        }
    }
}
