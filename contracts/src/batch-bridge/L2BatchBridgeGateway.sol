// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {AccessControlEnumerableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/AccessControlEnumerableUpgradeable.sol";
import {SafeERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";
import {IERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";

import {IL2ScrollMessenger} from "../L2/IL2ScrollMessenger.sol";

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

    event FinalizeBatchBridge(address indexed l1Token, address indexed l2Token, uint256 indexed phase);

    event DistributeFailed(address l2Token, uint256 phase, address receiver, uint256 amount);

    /*************
     * Constants *
     *************/

    /// @notice The role for batch deposit keeper.
    bytes32 public constant KEEPER_ROLE = keccak256("KEEPER_ROLE");

    /// @notice The safe gas limit for ETH transfer
    uint256 private constant SAFE_ETH_TRANSFER_GAS_LIMIT = 100000;

    /// @notice The address of corresponding `L1BatchBridgeGateway` contract.
    address public immutable counterpart;

    /// @notice The address of corresponding `L2ScrollMessenger` contract.
    address public immutable messenger;

    /*************
     * Variables *
     *************/

    /// @notice Mapping from l2 token address to l1 token address.
    mapping(address => address) public tokenMapping;

    /// @notice Mapping from L2 token address to phase index to phase hash.
    mapping(address => mapping(uint256 => bytes32)) public phaseHashes;

    mapping(address => uint256) public failedAmount;

    mapping(bytes32 => bool) public isDistributed;

    /***************
     * Constructor *
     ***************/

    constructor(address _counterpart, address _messenger) {
        _disableInitializers();

        counterpart = _counterpart;
        messenger = _messenger;
    }

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

    receive() external payable {
        if (_msgSender() != messenger) revert();
    }

    function finalizeBatchBridge(
        address l1Token,
        address l2Token,
        uint256 phase,
        bytes32 hash
    ) external {
        if (tokenMapping[l2Token] != l1Token) revert();
        if (_msgSender() != messenger) revert();
        if (counterpart != IL2ScrollMessenger(messenger).xDomainMessageSender()) revert();

        phaseHashes[l2Token][phase] = hash;

        emit FinalizeBatchBridge(l1Token, l2Token, phase);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update layer 2 to layer 1 token mapping.
    /// @param l2Token The address of corresponding ERC20 token on layer 2.
    /// @param l1Token The address of ERC20 token on layer 1.
    function updateTokenMapping(address l2Token, address l1Token) external onlyRole(DEFAULT_ADMIN_ROLE) {
        address oldL1Token = tokenMapping[l2Token];
        tokenMapping[l2Token] = l1Token;

        emit UpdateTokenMapping(l2Token, oldL1Token, l1Token);
    }

    function withdrawFailedAmount(address token, address receiver) external onlyRole(DEFAULT_ADMIN_ROLE) {
        uint256 amount = failedAmount[token];
        if (amount == 0) revert();

        _transferToken(token, receiver, amount);
    }

    /// @notice Distribute deposited token to corresponding receivers.
    /// @param l2Token The address of L2 token.
    /// @param phase The phase to distribute.
    /// @param nodes The list of encoded L1 deposits.
    function distribute(
        address l2Token,
        uint256 phase,
        bytes32[] memory nodes
    ) external onlyRole(KEEPER_ROLE) {
        address l1Token = tokenMapping[l2Token];
        bytes32 hash;
        assembly {
            hash := add(shl(96, l1Token), phase)
        }
        for (uint256 i = 0; i < nodes.length; i++) {
            hash = _efficientHash(hash, nodes[i]);
        }
        if (phaseHashes[l2Token][phase] != hash) revert();
        if (isDistributed[hash]) revert();

        // do transfer and allow failure to avoid DDOS attack
        for (uint256 i = 0; i < nodes.length; i++) {
            address receiver = address(uint160(uint256(nodes[i]) >> 96));
            uint256 amount = uint256(nodes[i]) & 0xffffffffffffffffffffffff;
            if (!_transferToken(l2Token, receiver, amount)) {
                failedAmount[l2Token] += amount;

                emit DistributeFailed(l2Token, phase, receiver, amount);
            }
        }

        isDistributed[hash] = true;
    }

    /**********************
     * Internal Functions *
     **********************/

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
    /// @return success Whether the transfer is successful.
    function _transferToken(
        address token,
        address receiver,
        uint256 amount
    ) private returns (bool success) {
        if (token == address(0)) {
            (success, ) = receiver.call{value: amount, gas: SAFE_ETH_TRANSFER_GAS_LIMIT}("");
        } else {
            try IERC20Upgradeable(token).transfer(receiver, amount) {
                success = true;
            } catch {
                success = false;
            }
        }
    }
}
