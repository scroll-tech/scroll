// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

interface IETHRateLimiter {
    /**********
     * Events *
     **********/

    /// @notice Emitted when the total limit is updated.
    /// @param oldTotalLimit The previous value of total limit before updating.
    /// @param newTotalLimit The current value of total limit after updating.
    event UpdateTotalLimit(uint256 oldTotalLimit, uint256 newTotalLimit);

    /**********
     * Errors *
     **********/

    /// @dev Thrown when the `periodDuration` is initialized to zero.
    error PeriodIsZero();

    /// @dev Thrown when the `totalAmount` is initialized to zero.
    error TotalLimitIsZero();

    /// @dev Thrown when an amount breaches the total limit in the period.
    error ExceedTotalLimit();

    /// @dev Thrown when the call is not spender.
    error CallerNotSpender();

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Request some ETH usage for `sender`.
    /// @param _amount The amount of ETH to use.
    function addUsedAmount(uint256 _amount) external;
}
