// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {ScrollGatewayBase} from "../libraries/gateway/ScrollGatewayBase.sol";

// solhint-disable func-name-mixedcase

abstract contract LidoGatewayManager is ScrollGatewayBase {
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

    /// @notice Emitted when the deposits enabler is updated.
    /// @param oldEnabler The address of the previous deposits enabler.
    /// @param newEnabler The address of the current deposits enabler.
    event UpdateDepositsEnabler(address indexed oldEnabler, address indexed newEnabler);

    /// @notice Emitted when the deposits disabler is updated.
    /// @param oldDisabler The address of the previous deposits disabler.
    /// @param newDisabler The address of the current deposits disabler.
    event UpdateDepositsDisabler(address indexed oldDisabler, address indexed newDisabler);

    /// @notice Emitted when the withdrawals enabler is updated.
    /// @param oldEnabler The address of the previous withdrawals enabler.
    /// @param newEnabler The address of the current withdrawals enabler.
    event UpdateWithdrawalsEnabler(address indexed oldEnabler, address indexed newEnabler);

    /// @notice Emitted when the withdrawals disabler is updated.
    /// @param oldDisabler The address of the previous withdrawals disabler.
    /// @param newDisabler The address of the current withdrawals disabler.
    event UpdateWithdrawalsDisabler(address indexed oldDisabler, address indexed newDisabler);

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
    /// @param depositsEnabler The address of user who can enable deposits
    /// @param depositsEnabler The address of user who can disable deposits
    /// @param withdrawalsEnabler The address of user who can enable withdrawals
    /// @param withdrawalsDisabler The address of user who can disable withdrawals
    struct State {
        bool isDepositsEnabled;
        bool isWithdrawalsEnabled;
        address depositsEnabler;
        address depositsDisabler;
        address withdrawalsEnabler;
        address withdrawalsDisabler;
    }

    /*************
     * Constants *
     *************/

    /// @dev The location of the slot with State
    bytes32 private constant STATE_SLOT = keccak256("LidoGatewayManager.bridgingState");

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

        s.depositsDisabler = _depositsEnabler;
        emit UpdateDepositsEnabler(address(0), _depositsEnabler);

        s.depositsDisabler = _depositsDisabler;
        emit UpdateDepositsDisabler(address(0), _depositsDisabler);

        s.withdrawalsEnabler = _withdrawalsEnabler;
        emit UpdateWithdrawalsEnabler(address(0), _withdrawalsEnabler);

        s.withdrawalsDisabler = _withdrawalsDisabler;
        emit UpdateWithdrawalsDisabler(address(0), _withdrawalsDisabler);
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

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Enables the deposits if they are disabled
    function enableDeposits() external {
        if (isDepositsEnabled()) revert ErrorDepositsEnabled();
        if (_msgSender() != _loadState().depositsEnabler) revert ErrorCallerIsNotDepositsEnabler();

        _loadState().isDepositsEnabled = true;
        emit DepositsEnabled(_msgSender());
    }

    /// @notice Disables the deposits if they aren't disabled yet
    function disableDeposits() external whenDepositsEnabled {
        if (_msgSender() != _loadState().depositsDisabler) revert ErrorCallerIsNotDepositsDisabler();

        _loadState().isDepositsEnabled = false;
        emit DepositsDisabled(_msgSender());
    }

    /// @notice Enables the withdrawals if they are disabled
    function enableWithdrawals() external {
        if (isWithdrawalsEnabled()) revert ErrorWithdrawalsEnabled();
        if (_msgSender() != _loadState().withdrawalsEnabler) revert ErrorCallerIsNotWithdrawalsEnabler();

        _loadState().isWithdrawalsEnabled = true;
        emit WithdrawalsEnabled(_msgSender());
    }

    /// @notice Disables the withdrawals if they aren't disabled yet
    function disableWithdrawals() external whenWithdrawalsEnabled {
        if (_msgSender() != _loadState().withdrawalsDisabler) revert ErrorCallerIsNotWithdrawalsDisabler();

        _loadState().isWithdrawalsEnabled = false;
        emit WithdrawalsDisabled(_msgSender());
    }

    /// @notice Update the address of deposits enabler.
    /// @param _newEnabler The address of new deposits enabler.
    function updateDepositsEnabler(address _newEnabler) external onlyOwner {
        State storage s = _loadState();
        address _oldEnabler = s.depositsEnabler;
        s.depositsEnabler = _newEnabler;

        emit UpdateDepositsEnabler(_oldEnabler, _newEnabler);
    }

    /// @notice Update the address of deposits disabler.
    /// @param _newDisabler The address of new deposits disabler.
    function updateDepositsDisabler(address _newDisabler) external onlyOwner {
        State storage s = _loadState();
        address _oldDisabler = s.depositsDisabler;
        s.depositsDisabler = _newDisabler;

        emit UpdateDepositsDisabler(_oldDisabler, _newDisabler);
    }

    /// @notice Update the address of withdrawals enabler.
    /// @param _newEnabler The address of new withdrawals enabler.
    function updateWithdrawalsEnabler(address _newEnabler) external onlyOwner {
        State storage s = _loadState();
        address _oldEnabler = s.withdrawalsEnabler;
        s.withdrawalsEnabler = _newEnabler;

        emit UpdateWithdrawalsEnabler(_oldEnabler, _newEnabler);
    }

    /// @notice Update the address of withdrawals disabler.
    /// @param _newDisabler The address of new withdrawals disabler.
    function updateWithdrawalsDisabler(address _newDisabler) external onlyOwner {
        State storage s = _loadState();
        address _oldDisabler = s.withdrawalsDisabler;
        s.withdrawalsDisabler = _newDisabler;

        emit UpdateWithdrawalsDisabler(_oldDisabler, _newDisabler);
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
}
