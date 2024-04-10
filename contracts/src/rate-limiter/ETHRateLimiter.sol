// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {SafeCast} from "@openzeppelin/contracts/utils/math/SafeCast.sol";

import {IETHRateLimiter} from "./IETHRateLimiter.sol";

// solhint-disable func-name-mixedcase
// solhint-disable not-rely-on-time

contract ETHRateLimiter is Ownable, IETHRateLimiter {
    /***********
     * Structs *
     ***********/

    struct ETHAmount {
        // The timestamp when the amount is updated.
        uint48 lastUpdateTs;
        // The ETH limit in wei.
        uint104 limit;
        // The amount of ETH in current period.
        uint104 amount;
    }

    /*************
     * Constants *
     *************/

    /// @notice The period length in seconds.
    /// @dev The time frame for the `k`-th period is `[periodDuration * k, periodDuration * (k + 1))`.
    uint256 public immutable periodDuration;

    /// @notice The address of ETH spender.
    address public immutable spender;

    /*************
     * Variables *
     *************/

    /// @notice The ETH amount used in current period.
    ETHAmount public currentPeriod;

    /***************
     * Constructor *
     ***************/

    constructor(
        uint256 _periodDuration,
        address _spender,
        uint104 _totalLimit
    ) {
        if (_periodDuration == 0) {
            revert PeriodIsZero();
        }

        periodDuration = _periodDuration;
        spender = _spender;

        _updateTotalLimit(_totalLimit);
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IETHRateLimiter
    function addUsedAmount(uint256 _amount) external override {
        if (_msgSender() != spender) {
            revert CallerNotSpender();
        }
        if (_amount == 0) return;

        uint256 _currentPeriodStart = (block.timestamp / periodDuration) * periodDuration;

        // check total limit
        uint256 _currentTotalAmount;
        ETHAmount memory _currentPeriod = currentPeriod;

        if (uint256(_currentPeriod.lastUpdateTs) < _currentPeriodStart) {
            _currentTotalAmount = _amount;
        } else {
            _currentTotalAmount = _currentPeriod.amount + _amount;
        }
        if (_currentTotalAmount > _currentPeriod.limit) {
            revert ExceedTotalLimit();
        }

        _currentPeriod.lastUpdateTs = uint48(block.timestamp);
        _currentPeriod.amount = SafeCast.toUint104(_currentTotalAmount);

        currentPeriod = _currentPeriod;
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the total ETH amount limit.
    /// @param _newTotalLimit The new total limit.
    function updateTotalLimit(uint104 _newTotalLimit) external onlyOwner {
        _updateTotalLimit(_newTotalLimit);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to update the total token amount limit.
    /// @param _newTotalLimit The new total limit.
    function _updateTotalLimit(uint104 _newTotalLimit) private {
        if (_newTotalLimit == 0) {
            revert TotalLimitIsZero();
        }

        uint256 _oldTotalLimit = currentPeriod.limit;
        currentPeriod.limit = _newTotalLimit;

        emit UpdateTotalLimit(_oldTotalLimit, _newTotalLimit);
    }
}
