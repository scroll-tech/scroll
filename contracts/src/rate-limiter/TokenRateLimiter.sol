// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {AccessControlEnumerable} from "@openzeppelin/contracts/access/AccessControlEnumerable.sol";
import {SafeCast} from "@openzeppelin/contracts/utils/math/SafeCast.sol";

import {ITokenRateLimiter} from "./ITokenRateLimiter.sol";

// solhint-disable func-name-mixedcase
// solhint-disable not-rely-on-time

contract TokenRateLimiter is AccessControlEnumerable, ITokenRateLimiter {
    /***********
     * Structs *
     ***********/

    struct TokenAmount {
        // The timestamp when the amount is updated.
        uint48 lastUpdateTs;
        // The token limit.
        uint104 limit;
        // The amount of token in current period.
        uint104 amount;
    }

    /*************
     * Constants *
     *************/

    /// @notice The role for token spender.
    bytes32 public constant TOKEN_SPENDER_ROLE = keccak256("TOKEN_SPENDER_ROLE");

    /// @notice The period length in seconds.
    /// @dev The time frame for the `k`-th period is `[periodDuration * k, periodDuration * (k + 1))`.
    uint256 public immutable periodDuration;

    /*************
     * Variables *
     *************/

    /// @notice Mapping from token address to the total amounts used in current period and total token amount limit.
    mapping(address => TokenAmount) public currentPeriod;

    /// @notice Mapping from token address to the default token amount limit per user.
    mapping(address => uint256) public defaultUserLimit;

    /// @notice Mapping from token address to user address to the amounts used in current period and custom token amount limit.
    mapping(address => mapping(address => TokenAmount)) public userCurrentPeriod;

    /// @dev The storage slots for future usage.
    uint256[45] private __gap;

    /***************
     * Constructor *
     ***************/

    constructor(uint256 _periodDuration) {
        if (_periodDuration == 0) {
            revert PeriodIsZero();
        }

        _setupRole(DEFAULT_ADMIN_ROLE, msg.sender);

        periodDuration = _periodDuration;
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @notice Return the token limit for specific user.
    /// @param _token The address of the token to query.
    /// @param _account The address of the user to query.
    function getUserCustomLimit(address _token, address _account) external view returns (uint256) {
        uint256 _userLimit = defaultUserLimit[_token];
        TokenAmount memory _userCurrentPeriod = userCurrentPeriod[_token][_account];
        if (_userCurrentPeriod.limit != 0) {
            _userLimit = _userCurrentPeriod.limit;
        }
        return _userLimit;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc ITokenRateLimiter
    function addUsedAmount(
        address _token,
        address _sender,
        uint256 _amount
    ) external override onlyRole(TOKEN_SPENDER_ROLE) {
        if (_amount == 0) return;

        uint256 _currentPeriodStart = (block.timestamp / periodDuration) * periodDuration;

        // check total limit, `0` means no limit at all.
        uint256 _currentTotalAmount;
        TokenAmount memory _currentPeriod = currentPeriod[_token];
        if (_currentPeriod.lastUpdateTs < _currentPeriodStart) {
            _currentTotalAmount = _amount;
        } else {
            _currentTotalAmount = _currentPeriod.amount + _amount;
        }
        if (_currentPeriod.limit != 0 && _currentTotalAmount > _currentPeriod.limit) {
            revert ExceedTotalLimit(_token);
        }

        // check user limit, `0` means no limit at all.
        uint256 _currentUserAmount;
        TokenAmount memory _userCurrentPeriod = userCurrentPeriod[_token][_sender];
        if (_userCurrentPeriod.lastUpdateTs < _currentPeriodStart) {
            _currentUserAmount = _amount;
        } else {
            _currentUserAmount = _userCurrentPeriod.amount + _amount;
        }

        uint256 _userLimit = defaultUserLimit[_token];
        if (_userCurrentPeriod.limit != 0) {
            _userLimit = _userCurrentPeriod.limit;
        }
        if (_userLimit != 0 && _currentUserAmount > _userLimit) {
            revert ExceedUserLimit(_token);
        }

        _currentPeriod.lastUpdateTs = uint48(block.timestamp);
        _currentPeriod.amount = SafeCast.toUint104(_currentTotalAmount);

        _userCurrentPeriod.lastUpdateTs = uint48(block.timestamp);
        _userCurrentPeriod.amount = SafeCast.toUint104(_currentUserAmount);

        currentPeriod[_token] = _currentPeriod;
        userCurrentPeriod[_token][_sender] = _userCurrentPeriod;
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the total token amount limit.
    /// @param _newTotalLimit The new total limit.
    function updateTotalLimit(address _token, uint104 _newTotalLimit) external onlyRole(DEFAULT_ADMIN_ROLE) {
        if (_newTotalLimit == 0) {
            revert TotalLimitIsZero(_token);
        }

        uint256 _oldTotalLimit = currentPeriod[_token].limit;
        currentPeriod[_token].limit = _newTotalLimit;

        emit UpdateTotalLimit(_token, _oldTotalLimit, _newTotalLimit);
    }

    /// @notice Update the default token amount limit per user.
    /// @param _newDefaultUserLimit The new default limit per user.
    function updateDefaultUserLimit(address _token, uint256 _newDefaultUserLimit)
        external
        onlyRole(DEFAULT_ADMIN_ROLE)
    {
        if (_newDefaultUserLimit == 0) {
            revert DefaultUserLimitIsZero(_token);
        }

        uint256 _oldDefaultUserLimit = defaultUserLimit[_token];
        defaultUserLimit[_token] = _newDefaultUserLimit;

        emit UpdateDefaultUserLimit(_token, _oldDefaultUserLimit, _newDefaultUserLimit);
    }

    /// @notice Update the custom limit for specific user.
    ///
    /// @dev Use `_newLimit=0` if owner wants to reset the custom limit.
    ///
    /// @param _account The address of the user.
    /// @param _newLimit The new custom limit for the user.
    function updateCustomUserLimit(
        address _token,
        address _account,
        uint104 _newLimit
    ) external onlyRole(DEFAULT_ADMIN_ROLE) {
        uint256 _oldLimit = userCurrentPeriod[_token][_account].limit;
        userCurrentPeriod[_token][_account].limit = _newLimit;

        emit UpdateCustomUserLimit(_token, _account, _oldLimit, _newLimit);
    }
}
