// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {SafeCast} from "@openzeppelin/contracts/utils/math/SafeCast.sol";

import {IETHRateLimiter} from "./IETHRateLimiter.sol";

// solhint-disable func-name-mixedcase
// solhint-disable not-rely-on-time

contract ETHRateLimiter is Ownable, IETHRateLimiter {
    /***********
     * Structs *
     ***********/

    struct TokenAmount {
        // The ETH limit in wei.
        uint128 limit;
        // The amount of ETH in current period.
        uint128 currentPeriodAmount;
    }

    /*************
     * Constants *
     *************/

    /// @notice The period length in seconds.
    uint256 public immutable periodDuration;

    /// @notice The address of ETH spender.
    address public immutable spender;

    /*************
     * Variables *
     *************/

    /// @notice The time at which the current period ends at.
    uint256 public currentPeriodEnd;

    /// @notice The total token amount limit.
    uint256 public totalLimit;

    /// @notice The total amounts used in current period.
    uint256 public currentPeriodAmount;

    /// @notice The default token amount limit per user.
    uint256 public defaultUserLimit;

    /// @notice Mapping from user address to the amounts used in current period and custom token amount limit.
    mapping(address => TokenAmount) public userAmount;

    /***************
     * Constructor *
     ***************/

    constructor(
        uint256 _periodDuration,
        address _spender,
        uint256 _totalLimit,
        uint256 _defaultUserLimit
    ) {
        if (_periodDuration == 0) {
            revert PeriodIsZero();
        }

        if (_totalLimit == 0) {
            revert TotalLimitIsZero();
        }

        if (_defaultUserLimit == 0) {
            revert DefaultUserLimitIsZero();
        }

        periodDuration = _periodDuration;
        spender = _spender;

        currentPeriodEnd = block.timestamp + _periodDuration;

        totalLimit = _totalLimit;
        defaultUserLimit = _defaultUserLimit;
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @notice Return the token limit for specific user.
    /// @param _account The address of the user to query.
    function getUserCustomLimit(address _account) external view returns (uint256) {
        uint256 _userLimit = defaultUserLimit;
        TokenAmount memory _userAmount = userAmount[_account];
        if (_userAmount.limit != 0) {
            _userLimit = _userAmount.limit;
        }
        return _userLimit;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IETHRateLimiter
    function addUsedAmount(address _sender, uint256 _amount) external override {
        if (msg.sender != spender) {
            revert CallerNotSpender();
        }

        if (_amount == 0) return;

        uint256 _currentTotal;
        uint256 _currentUser;

        TokenAmount memory _userAmount = userAmount[_sender];
        if (currentPeriodEnd < block.timestamp) {
            currentPeriodEnd = block.timestamp + periodDuration;
            _currentTotal = _amount;
            _currentUser = _amount;
        } else {
            _currentTotal = currentPeriodAmount + _amount;
            _currentUser = _userAmount.currentPeriodAmount + _amount;
        }

        // check total limit
        if (_currentTotal > totalLimit) {
            revert ExceedTotalLimit();
        }

        // check user limit
        uint256 _userLimit = defaultUserLimit;
        if (_userAmount.limit != 0) {
            _userLimit = _userAmount.limit;
        }
        if (_currentUser > _userLimit) {
            revert ExceedUserLimit();
        }

        _userAmount.currentPeriodAmount = SafeCast.toUint128(_currentUser);

        currentPeriodAmount = _currentTotal;
        userAmount[_sender] = _userAmount;
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the total token amount limit.
    /// @param _newTotalLimit The new total limit.
    function updateTotalLimit(uint128 _newTotalLimit) external onlyOwner {
        if (_newTotalLimit == 0) {
            revert TotalLimitIsZero();
        }

        uint256 _oldTotalLimit = totalLimit;
        totalLimit = _newTotalLimit;

        emit UpdateTotalLimit(_oldTotalLimit, _newTotalLimit);
    }

    /// @notice Update the default token amount limit per user.
    /// @param _newDefaultUserLimit The new default limit per user.
    function updateDefaultUserLimit(uint256 _newDefaultUserLimit) external onlyOwner {
        if (_newDefaultUserLimit == 0) {
            revert DefaultUserLimitIsZero();
        }

        uint256 _oldDefaultUserLimit = defaultUserLimit;
        defaultUserLimit = _newDefaultUserLimit;

        emit UpdateDefaultUserLimit(_oldDefaultUserLimit, _newDefaultUserLimit);
    }

    /// @notice Update the custom limit for specific user.
    ///
    /// @dev Use `_newLimit=0` if owner wants to reset the custom limit.
    ///
    /// @param _account The address of the user.
    /// @param _newLimit The new custom limit for the user.
    function updateCustomUserLimit(address _account, uint128 _newLimit) external onlyOwner {
        uint256 _oldLimit = userAmount[_account].limit;
        userAmount[_account].limit = _newLimit;

        emit UpdateCustomUserLimit(_account, _oldLimit, _newLimit);
    }
}
