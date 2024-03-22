// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

interface ITokenRateLimiter {
    /**********
     * Events *
     **********/

    /// @notice Emitted when the total limit is updated.
    /// @param oldTotalLimit The previous value of total limit before updating.
    /// @param newTotalLimit The current value of total limit after updating.
    event UpdateTotalLimit(address indexed token, uint256 oldTotalLimit, uint256 newTotalLimit);

    /**********
     * Errors *
     **********/

    /// @dev Thrown when the `periodDuration` is initialized to zero.
    error PeriodIsZero();

    /// @dev Thrown when the `totalAmount` is initialized to zero.
    /// @param token The address of the token.
    error TotalLimitIsZero(address token);

    /// @dev Thrown when an amount breaches the total limit in the period.
    /// @param token The address of the token.
    error ExceedTotalLimit(address token);

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Request some token usage for `sender`.
    /// @param token The address of the token.
    /// @param amount The amount of token to use.
    function addUsedAmount(address token, uint256 amount) external;
}
