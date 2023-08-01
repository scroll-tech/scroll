// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

interface IETHRateLimiter {
    /**********
     * Events *
     **********/

    /// @notice Emitted when the total limit is updated.
    /// @param oldTotalLimit The previous value of total limit before updating.
    /// @param newTotalLimit The current value of total limit after updating.
    event UpdateTotalLimit(uint256 oldTotalLimit, uint256 newTotalLimit);

    /// @notice Emitted when the default limit per user is updated.
    /// @param oldDefaultUserLimit The previous value of default limit per user before updating.
    /// @param newDefaultUserLimit The current value of default limit per user after updating.
    event UpdateDefaultUserLimit(uint256 oldDefaultUserLimit, uint256 newDefaultUserLimit);

    /// @notice Emitted when the custom limit for some user is updated.
    /// @param account The address of the user updated.
    /// @param oldUserLimit The previous custom user limit before updating.
    /// @param newUserLimit The current custom user limit after updating.
    event UpdateCustomUserLimit(address indexed account, uint256 oldUserLimit, uint256 newUserLimit);

    /**********
     * Errors *
     **********/

    /// @dev Thrown when the `periodDuration` is initialized to zero.
    error PeriodIsZero();

    /// @dev Thrown when the `totalAmount` is initialized to zero.
    error TotalLimitIsZero();

    /// @dev Thrown when the `defaultUserLimit` is initialized to zero.
    error DefaultUserLimitIsZero();

    /// @dev Thrown when an amount breaches the total limit in the period.
    error ExceedTotalLimit();

    /// @dev Thrown when an amount breaches the user limit in the period.
    error ExceedUserLimit();

    /// @dev Thrown when the call is not spender.
    error CallerNotSpender();

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Request some ETH usage for `sender`.
    /// @param _sender The address of the ETH sender.
    /// @param _amount The amount of ETH to use.
    function addUsedAmount(address _sender, uint256 _amount) external;
}
