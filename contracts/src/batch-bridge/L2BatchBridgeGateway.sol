// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {AccessControlEnumerableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/AccessControlEnumerableUpgradeable.sol";
import {IERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";

import {IL2ScrollMessenger} from "../L2/IL2ScrollMessenger.sol";
import {BatchBridgeCodec} from "./BatchBridgeCodec.sol";

/// @title L2BatchBridgeGateway
contract L2BatchBridgeGateway is AccessControlEnumerableUpgradeable {
    /**********
     * Events *
     **********/

    /// @notice Emitted when token mapping for ERC20 token is updated.
    /// @param l2Token The address of corresponding ERC20 token in layer 2.
    /// @param oldL1Token The address of the old corresponding ERC20 token in layer 1.
    /// @param newL1Token The address of the new corresponding ERC20 token in layer 1.
    event UpdateTokenMapping(address indexed l2Token, address indexed oldL1Token, address indexed newL1Token);

    /// @notice Emitted when batch bridge is finalized.
    /// @param l1Token The address of token in L1.
    /// @param l2Token The address of token in L2.
    /// @param batchIndex The index of batch finalized.
    event FinalizeBatchDeposit(address indexed l1Token, address indexed l2Token, uint256 indexed batchIndex);

    /// @notice Emitted when batch distribution finished.
    /// @param l1Token The address of token in L1.
    /// @param l2Token The address of token in L2.
    /// @param batchIndex The index of batch distributed.
    event BatchDistribute(address indexed l1Token, address indexed l2Token, uint256 indexed batchIndex);

    /// @notice Emitted when token distribute failed.
    /// @param l2Token The address of token in L2.
    /// @param batchIndex The index of the batch.
    /// @param receiver The address of token receiver.
    /// @param amount The amount of token to distribute.
    event DistributeFailed(address indexed l2Token, uint256 indexed batchIndex, address receiver, uint256 amount);

    /**********
     * Errors *
     **********/

    /// @dev Thrown when caller is not `messenger`.
    error ErrorCallerNotMessenger();

    /// @dev Thrown when the L1 token mapping mismatch with `finalizeBatchBridge`.
    error ErrorL1TokenMismatched();

    /// @dev Thrown when message sender is not `counterpart`.
    error ErrorMessageSenderNotCounterpart();

    /// @dev Thrown no failed distribution exists.
    error ErrorNoFailedDistribution();

    /// @dev Thrown when the batch hash mismatch.
    error ErrorBatchHashMismatch();

    /// @dev Thrown when distributing the same batch.
    error ErrorBatchDistributed();

    /*************
     * Constants *
     *************/

    /// @notice The role for batch deposit keeper.
    bytes32 public constant KEEPER_ROLE = keccak256("KEEPER_ROLE");

    /// @notice The safe gas limit for ETH transfer
    uint256 private constant SAFE_ETH_TRANSFER_GAS_LIMIT = 50000;

    /// @notice The address of corresponding `L1BatchBridgeGateway` contract.
    address public immutable counterpart;

    /// @notice The address of corresponding `L2ScrollMessenger` contract.
    address public immutable messenger;

    /*************
     * Variables *
     *************/

    /// @notice Mapping from l2 token address to l1 token address.
    mapping(address => address) public tokenMapping;

    /// @notice Mapping from L2 token address to batch index to batch hash.
    mapping(address => mapping(uint256 => bytes32)) public batchHashes;

    /// @notice Mapping from token address to the amount of failed distribution.
    mapping(address => uint256) public failedAmount;

    /// @notice Mapping from batch hash to the distribute status.
    mapping(bytes32 => bool) public isDistributed;

    /*************
     * Modifiers *
     *************/

    modifier onlyMessenger() {
        if (_msgSender() != messenger) {
            revert ErrorCallerNotMessenger();
        }
        _;
    }

    /***************
     * Constructor *
     ***************/

    /// @param _counterpart The address of `L1BatchBridgeGateway` contract in L1.
    /// @param _messenger The address of `L2ScrollMessenger` contract in L2.
    constructor(address _counterpart, address _messenger) {
        _disableInitializers();

        counterpart = _counterpart;
        messenger = _messenger;
    }

    /// @notice Initialize the storage of `L2BatchBridgeGateway`.
    function initialize() external initializer {
        __Context_init(); // from ContextUpgradeable
        __ERC165_init(); // from ERC165Upgradeable
        __AccessControl_init(); // from AccessControlUpgradeable
        __AccessControlEnumerable_init(); // from AccessControlEnumerableUpgradeable

        _grantRole(DEFAULT_ADMIN_ROLE, _msgSender());
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Receive batch bridged ETH from `L2ScrollMessenger`.
    receive() external payable onlyMessenger {
        // empty
    }

    /// @notice Finalize L1 initiated batch token deposit.
    /// @param l1Token The address of the token in L1.
    /// @param l2Token The address of the token in L2.
    /// @param batchIndex The index of this batch bridge.
    /// @param hash The hash of this batch.
    function finalizeBatchDeposit(
        address l1Token,
        address l2Token,
        uint256 batchIndex,
        bytes32 hash
    ) external onlyMessenger {
        if (counterpart != IL2ScrollMessenger(messenger).xDomainMessageSender()) {
            revert ErrorMessageSenderNotCounterpart();
        }

        // trust the messenger and update `tokenMapping` in first call
        // another assumption is this function should never fail due to out of gas
        address storedL1Token = tokenMapping[l2Token];
        if (storedL1Token == address(0) && l1Token != address(0)) {
            tokenMapping[l2Token] = l1Token;
        } else if (storedL1Token != l1Token) {
            // this usually won't happen, check just in case.
            revert ErrorL1TokenMismatched();
        }

        batchHashes[l2Token][batchIndex] = hash;

        emit FinalizeBatchDeposit(l1Token, l2Token, batchIndex);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Withdraw distribution failed tokens.
    /// @param token The address of token to withdraw.
    /// @param receiver The address of token receiver.
    function withdrawFailedAmount(address token, address receiver) external onlyRole(DEFAULT_ADMIN_ROLE) {
        uint256 amount = failedAmount[token];
        if (amount == 0) revert ErrorNoFailedDistribution();
        failedAmount[token] = 0;

        _transferToken(token, receiver, amount);
    }

    /// @notice Distribute deposited token to corresponding receivers.
    /// @param l2Token The address of L2 token.
    /// @param batchIndex The index of batch to distribute.
    /// @param nodes The list of encoded L1 deposits.
    function distribute(
        address l2Token,
        uint64 batchIndex,
        bytes32[] memory nodes
    ) external onlyRole(KEEPER_ROLE) {
        address l1Token = tokenMapping[l2Token];
        bytes32 hash = BatchBridgeCodec.encodeInitialNode(l1Token, batchIndex);
        for (uint256 i = 0; i < nodes.length; i++) {
            hash = BatchBridgeCodec.hash(hash, nodes[i]);
        }
        if (batchHashes[l2Token][batchIndex] != hash) {
            revert ErrorBatchHashMismatch();
        }
        if (isDistributed[hash]) {
            revert ErrorBatchDistributed();
        }
        isDistributed[hash] = true;

        // do transfer and allow failure to avoid DDOS attack
        for (uint256 i = 0; i < nodes.length; i++) {
            (address receiver, uint256 amount) = BatchBridgeCodec.decodeNode(nodes[i]);
            if (!_transferToken(l2Token, receiver, amount)) {
                failedAmount[l2Token] += amount;

                emit DistributeFailed(l2Token, batchIndex, receiver, amount);
            }
        }

        emit BatchDistribute(l1Token, l2Token, batchIndex);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to transfer token, including ETH.
    /// @param token The address of token.
    /// @param receiver The address of token receiver.
    /// @param amount The amount of token to transfer.
    /// @return success Whether the transfer is successful.
    function _transferToken(
        address token,
        address receiver,
        uint256 amount
    ) private returns (bool success) {
        if (token == address(0)) {
            // We add gas limit here to avoid DDOS from malicious receiver.
            (success, ) = receiver.call{value: amount, gas: SAFE_ETH_TRANSFER_GAS_LIMIT}("");
        } else {
            // We perform a low level call here, to bypass Solidity's return data size checking mechanism.
            // Normally, the token is selected that the call would not revert unless out of gas.
            bytes memory returnData;
            (success, returnData) = token.call(abi.encodeCall(IERC20Upgradeable.transfer, (receiver, amount)));
            if (success && returnData.length > 0) {
                success = abi.decode(returnData, (bool));
            }
        }
    }
}
