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

    /// @notice The token amount used in current period.
    TokenAmount public currentPeriod;

    /// @notice The default token amount limit per user.
    uint256 public defaultUserLimit;

    /// @notice Mapping from user address to the amounts used in current period and custom token amount limit.
    mapping(address => TokenAmount) public userCurrentPeriod;

    /***************
     * Constructor *
     ***************/

    constructor(
        uint256 _periodDuration,
        address _spender,
        uint104 _totalLimit,
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

        currentPeriod.limit = _totalLimit;
        defaultUserLimit = _defaultUserLimit;
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @notice Return the token limit for specific user.
    /// @param _account The address of the user to query.
    function getUserCustomLimit(address _account) external view returns (uint256) {
        uint256 _userLimit = defaultUserLimit;
        TokenAmount memory _userCurrentPeriod = userCurrentPeriod[_account];
        if (_userCurrentPeriod.limit != 0) {
            _userLimit = _userCurrentPeriod.limit;
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

        uint256 _currentPeriodStart = (block.timestamp / periodDuration) * periodDuration;

        // check total limit
        uint256 _currentTotal;
        TokenAmount memory _currentPeriod = currentPeriod;

        if (_currentPeriod.lastUpdateTs < _currentPeriodStart) {
            _currentTotal = _amount;
        } else {
            _currentTotal = _currentPeriod.amount + _amount;
        }
        if (_currentTotal > _currentPeriod.limit) {
            revert ExceedTotalLimit();
        }

        // check user limit
        uint256 _currentUser;
        TokenAmount memory _userCurrentPeriod = userCurrentPeriod[_sender];
        if (_userCurrentPeriod.lastUpdateTs < _currentPeriodStart) {
            _currentUser = _amount;
        } else {
            _currentUser = _userCurrentPeriod.amount + _amount;
        }

        uint256 _userLimit = defaultUserLimit;
        if (_userCurrentPeriod.limit != 0) {
            _userLimit = _userCurrentPeriod.limit;
        }
        if (_currentUser > _userLimit) {
            revert ExceedUserLimit();
        }

        _currentPeriod.lastUpdateTs = uint48(block.timestamp);
        _currentPeriod.amount = SafeCast.toUint104(_currentTotal);

        _userCurrentPeriod.lastUpdateTs = uint48(block.timestamp);
        _userCurrentPeriod.amount = SafeCast.toUint104(_currentUser);

        currentPeriod = _currentPeriod;
        userCurrentPeriod[_sender] = _userCurrentPeriod;
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the total token amount limit.
    /// @param _newTotalLimit The new total limit.
    function updateTotalLimit(uint104 _newTotalLimit) external onlyOwner {
        if (_newTotalLimit == 0) {
            revert TotalLimitIsZero();
        }

        uint256 _oldTotalLimit = currentPeriod.limit;
        currentPeriod.limit = _newTotalLimit;

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
    function updateCustomUserLimit(address _account, uint104 _newLimit) external onlyOwner {
        uint256 _oldLimit = userCurrentPeriod[_account].limit;
        userCurrentPeriod[_account].limit = _newLimit;

        emit UpdateCustomUserLimit(_account, _oldLimit, _newLimit);
    }
}
