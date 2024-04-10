// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {EnumerableSetUpgradeable} from "@openzeppelin/contracts-upgradeable/utils/structs/EnumerableSetUpgradeable.sol";

import {ScrollGatewayBase} from "../libraries/gateway/ScrollGatewayBase.sol";

// solhint-disable func-name-mixedcase

abstract contract LidoGatewayManager is ScrollGatewayBase {
    using EnumerableSetUpgradeable for EnumerableSetUpgradeable.AddressSet;

    /**********
     * Events *
     **********/

    /// @notice Emitted then caller enable deposits.
    /// @param enabler The address of caller.
    event DepositsEnabled(address indexed enabler);

    /// @notice Emitted then caller disable deposits.
    /// @param disabler The address of caller.
    event DepositsDisabled(address indexed disabler);

    /// @notice Emitted then caller enable withdrawals.
    /// @param enabler The address of caller.
    event WithdrawalsEnabled(address indexed enabler);

    /// @notice Emitted then caller disable withdrawals.
    /// @param disabler The address of caller.
    event WithdrawalsDisabled(address indexed disabler);

    /// @notice Emitted when `account` is granted `role`.
    ///
    /// @param role The role granted.
    /// @param account The address of account to grant the role.
    /// @param sender The address of owner.
    event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender);

    /// @notice Emitted when `account` is revoked `role`.
    ///
    /// @param role The role revoked.
    /// @param account The address of account to revoke the role.
    /// @param sender The address of owner.
    event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender);

    /**********
     * Errors *
     **********/

    /// @dev Thrown when deposits are enabled while caller try to enable it again.
    error ErrorDepositsEnabled();

    /// @dev Thrown when deposits are disable while caller try to deposits related operation.
    error ErrorDepositsDisabled();

    /// @dev Thrown when withdrawals are enabled while caller try to enable it again.
    error ErrorWithdrawalsEnabled();

    /// @dev Thrown when withdrawals are disable while caller try to withdrawals related operation.
    error ErrorWithdrawalsDisabled();

    /// @dev Thrown when caller is not deposits enabler.
    error ErrorCallerIsNotDepositsEnabler();

    /// @dev Thrown when caller is not deposits disabler.
    error ErrorCallerIsNotDepositsDisabler();

    /// @dev Thrown when caller is not withdrawals enabler.
    error ErrorCallerIsNotWithdrawalsEnabler();

    /// @dev Thrown when caller is not withdrawals disabler.
    error ErrorCallerIsNotWithdrawalsDisabler();

    /***********
     * Structs *
     ***********/

    /// @dev Stores the state of the bridging
    /// @param isDepositsEnabled Stores the state of the deposits
    /// @param isWithdrawalsEnabled Stores the state of the withdrawals
    /// @param roles Mapping from role to list of role members.
    struct State {
        bool isDepositsEnabled;
        bool isWithdrawalsEnabled;
        mapping(bytes32 => EnumerableSetUpgradeable.AddressSet) roles;
    }

    /*************
     * Constants *
     *************/

    /// @dev The location of the slot with State
    bytes32 private constant STATE_SLOT = keccak256("LidoGatewayManager.bridgingState");

    /// @notice The role for deposits enabler.
    bytes32 public constant DEPOSITS_ENABLER_ROLE = keccak256("BridgingManager.DEPOSITS_ENABLER_ROLE");

    /// @notice The role for deposits disabler.
    bytes32 public constant DEPOSITS_DISABLER_ROLE = keccak256("BridgingManager.DEPOSITS_DISABLER_ROLE");

    /// @notice The role for withdrawals enabler.
    bytes32 public constant WITHDRAWALS_ENABLER_ROLE = keccak256("BridgingManager.WITHDRAWALS_ENABLER_ROLE");

    /// @notice The role for withdrawals disabler.
    bytes32 public constant WITHDRAWALS_DISABLER_ROLE = keccak256("BridgingManager.WITHDRAWALS_DISABLER_ROLE");

    /**********************
     * Function Modifiers *
     **********************/

    /// @dev Validates that deposits are enabled
    modifier whenDepositsEnabled() {
        if (!isDepositsEnabled()) revert ErrorDepositsDisabled();
        _;
    }

    /// @dev Validates that withdrawals are enabled
    modifier whenWithdrawalsEnabled() {
        if (!isWithdrawalsEnabled()) revert ErrorWithdrawalsDisabled();
        _;
    }

    /***************
     * Constructor *
     ***************/

    /// @notice Initialize the storage of LidoGatewayManager.
    /// @param _depositsEnabler The address of user who can enable deposits
    /// @param _depositsEnabler The address of user who can disable deposits
    /// @param _withdrawalsEnabler The address of user who can enable withdrawals
    /// @param _withdrawalsDisabler The address of user who can disable withdrawals
    function __LidoGatewayManager_init(
        address _depositsEnabler,
        address _depositsDisabler,
        address _withdrawalsEnabler,
        address _withdrawalsDisabler
    ) internal onlyInitializing {
        State storage s = _loadState();

        s.isDepositsEnabled = true;
        emit DepositsEnabled(_msgSender());

        s.isWithdrawalsEnabled = true;
        emit WithdrawalsEnabled(_msgSender());

        _grantRole(DEPOSITS_ENABLER_ROLE, _depositsEnabler);
        _grantRole(DEPOSITS_DISABLER_ROLE, _depositsDisabler);
        _grantRole(WITHDRAWALS_ENABLER_ROLE, _withdrawalsEnabler);
        _grantRole(WITHDRAWALS_DISABLER_ROLE, _withdrawalsDisabler);
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @notice Returns whether the deposits are enabled or not
    function isDepositsEnabled() public view returns (bool) {
        return _loadState().isDepositsEnabled;
    }

    /// @notice Returns whether the withdrawals are enabled or not
    function isWithdrawalsEnabled() public view returns (bool) {
        return _loadState().isWithdrawalsEnabled;
    }

    /// @notice Returns `true` if `_account` has been granted `_role`.
    function hasRole(bytes32 _role, address _account) public view returns (bool) {
        return _loadState().roles[_role].contains(_account);
    }

    /// @notice Returns one of the accounts that have `_role`.
    ///
    /// @param _role The role to query.
    /// @param _index The index of account to query. It must be a value between 0 and  {getRoleMemberCount}, non-inclusive.
    function getRoleMember(bytes32 _role, uint256 _index) external view returns (address) {
        return _loadState().roles[_role].at(_index);
    }

    /// @notice Returns the number of accounts that have `role`.
    ///
    /// @dev Can be used together with {getRoleMember} to enumerate all bearers of a role.
    ///
    /// @param _role The role to query.
    function getRoleMemberCount(bytes32 _role) external view returns (uint256) {
        return _loadState().roles[_role].length();
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Enables the deposits if they are disabled
    function enableDeposits() external {
        if (isDepositsEnabled()) revert ErrorDepositsEnabled();
        if (!hasRole(DEPOSITS_ENABLER_ROLE, _msgSender())) {
            revert ErrorCallerIsNotDepositsEnabler();
        }

        _loadState().isDepositsEnabled = true;
        emit DepositsEnabled(_msgSender());
    }

    /// @notice Disables the deposits if they aren't disabled yet
    function disableDeposits() external whenDepositsEnabled {
        if (!hasRole(DEPOSITS_DISABLER_ROLE, _msgSender())) {
            revert ErrorCallerIsNotDepositsDisabler();
        }

        _loadState().isDepositsEnabled = false;
        emit DepositsDisabled(_msgSender());
    }

    /// @notice Enables the withdrawals if they are disabled
    function enableWithdrawals() external {
        if (isWithdrawalsEnabled()) revert ErrorWithdrawalsEnabled();
        if (!hasRole(WITHDRAWALS_ENABLER_ROLE, _msgSender())) {
            revert ErrorCallerIsNotWithdrawalsEnabler();
        }

        _loadState().isWithdrawalsEnabled = true;
        emit WithdrawalsEnabled(_msgSender());
    }

    /// @notice Disables the withdrawals if they aren't disabled yet
    function disableWithdrawals() external whenWithdrawalsEnabled {
        if (!hasRole(WITHDRAWALS_DISABLER_ROLE, _msgSender())) {
            revert ErrorCallerIsNotWithdrawalsDisabler();
        }

        _loadState().isWithdrawalsEnabled = false;
        emit WithdrawalsDisabled(_msgSender());
    }

    /// @notice Grants `_role` from `_account`.
    /// If `account` had been granted `role`, emits a {RoleGranted} event.
    ///
    /// @param _role The role to grant.
    /// @param _account The address of account to grant.
    function grantRole(bytes32 _role, address _account) external onlyOwner {
        _grantRole(_role, _account);
    }

    /// @notice Revokes `_role` from `_account`.
    /// If `account` had been granted `role`, emits a {RoleRevoked} event.
    ///
    /// @param _role The role to revoke.
    /// @param _account The address of account to revoke.
    function revokeRole(bytes32 _role, address _account) external onlyOwner {
        _revokeRole(_role, _account);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Returns the reference to the slot with State struct
    function _loadState() private pure returns (State storage r) {
        bytes32 slot = STATE_SLOT;
        // solhint-disable-next-line no-inline-assembly
        assembly {
            r.slot := slot
        }
    }

    /// @dev Internal function to grant `_role` from `_account`.
    /// If `account` had been granted `role`, emits a {RoleGranted} event.
    ///
    /// @param _role The role to grant.
    /// @param _account The address of account to grant.
    function _grantRole(bytes32 _role, address _account) internal {
        if (_loadState().roles[_role].add(_account)) {
            emit RoleGranted(_role, _account, _msgSender());
        }
    }

    /// @dev Internal function to revoke `_role` from `_account`.
    /// If `account` had been granted `role`, emits a {RoleRevoked} event.
    ///
    /// @param _role The role to revoke.
    /// @param _account The address of account to revoke.
    function _revokeRole(bytes32 _role, address _account) internal {
        if (_loadState().roles[_role].remove(_account)) {
            emit RoleRevoked(_role, _account, _msgSender());
        }
    }
}
