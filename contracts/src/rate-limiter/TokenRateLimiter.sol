// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

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

    /// @dev The storage slots for future usage.
    uint256[49] private __gap;

    /***************
     * Constructor *
     ***************/

    constructor(uint256 _periodDuration) {
        if (_periodDuration == 0) {
            revert PeriodIsZero();
        }

        _grantRole(DEFAULT_ADMIN_ROLE, _msgSender());

        periodDuration = _periodDuration;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc ITokenRateLimiter
    function addUsedAmount(address _token, uint256 _amount) external override onlyRole(TOKEN_SPENDER_ROLE) {
        if (_amount == 0) return;

        uint256 _currentPeriodStart = (block.timestamp / periodDuration) * periodDuration;

        // check total limit, `0` means no limit at all.
        uint256 _currentTotalAmount;
        TokenAmount memory _currentPeriod = currentPeriod[_token];
        if (uint256(_currentPeriod.lastUpdateTs) < _currentPeriodStart) {
            _currentTotalAmount = _amount;
        } else {
            _currentTotalAmount = _currentPeriod.amount + _amount;
        }
        if (_currentPeriod.limit != 0 && _currentTotalAmount > _currentPeriod.limit) {
            revert ExceedTotalLimit(_token);
        }

        _currentPeriod.lastUpdateTs = uint48(block.timestamp);
        _currentPeriod.amount = SafeCast.toUint104(_currentTotalAmount);

        currentPeriod[_token] = _currentPeriod;
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
}
