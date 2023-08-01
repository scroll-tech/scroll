// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {AccessControlEnumerable} from "@openzeppelin/contracts/access/AccessControlEnumerable.sol";

import {ITokenRateLimiter} from "./ITokenRateLimiter.sol";

// solhint-disable func-name-mixedcase
// solhint-disable not-rely-on-time

contract TokenRateLimiter is AccessControlEnumerable, ITokenRateLimiter {
    /***********
     * Structs *
     ***********/

    struct TokenAmount {
        // The token limit.
        uint128 limit;
        // The amount of token in current period.
        uint128 currentPeriodAmount;
    }

    /*************
     * Constants *
     *************/

    /// @notice The role for token spender.
    bytes32 public constant TOKEN_SPENDER_ROLE = keccak256("TOKEN_SPENDER_ROLE");

    /// @notice The period length in seconds.
    uint256 public immutable periodDuration;

    /*************
     * Variables *
     *************/

    /// @notice Mapping from token address to the time at which the current period ends at.
    mapping(address => uint256) public currentPeriodEnd;

    /// @notice Mapping from token address to the total amounts used in current period and total token amount limit.
    mapping(address => TokenAmount) public totalAmount;

    /// @notice Mapping from token address to the default token amount limit per user.
    mapping(address => uint256) public defaultUserLimit;

    /// @notice Mapping from token address to user address to the amounts used in current period and custom token amount limit.
    mapping(address => mapping(address => TokenAmount)) public userAmount;

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
        TokenAmount memory _userAmount = userAmount[_token][_account];
        if (_userAmount.limit != 0) {
            _userLimit = _userAmount.limit;
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

        uint256 _currentTotal;
        uint256 _currentUser;

        TokenAmount memory _totalAmount = totalAmount[_token];
        TokenAmount memory _userAmount = userAmount[_token][_sender];
        // @note The value of `currentPeriodEnd[_token]` is also initialized on the first call to this token.
        if (currentPeriodEnd[_token] < block.timestamp) {
            currentPeriodEnd[_token] = block.timestamp + periodDuration;
            _currentTotal = _amount;
            _currentUser = _amount;
        } else {
            _currentTotal = _totalAmount.currentPeriodAmount + _amount;
            _currentUser = _userAmount.currentPeriodAmount + _amount;
        }

        // check total limit, `0` means no limit at all.
        if (_totalAmount.limit != 0 && _currentTotal > _totalAmount.limit) {
            revert ExceedTotalLimit(_token);
        }

        // check user limit, `0` means no limit at all.
        uint256 _userLimit = defaultUserLimit[_token];
        if (_userAmount.limit != 0) {
            _userLimit = _userAmount.limit;
        }
        if (_userLimit != 0 && _currentUser > _userLimit) {
            revert ExceedUserLimit(_token);
        }

        totalAmount[_token] = _totalAmount;
        userAmount[_token][_sender] = _userAmount;
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the total token amount limit.
    /// @param _newTotalLimit The new total limit.
    function updateTotalLimit(address _token, uint128 _newTotalLimit) external onlyRole(DEFAULT_ADMIN_ROLE) {
        if (_newTotalLimit == 0) {
            revert TotalLimitIsZero(_token);
        }

        uint256 _oldTotalLimit = totalAmount[_token].limit;
        totalAmount[_token].limit = _newTotalLimit;

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
        uint128 _newLimit
    ) external onlyRole(DEFAULT_ADMIN_ROLE) {
        uint256 _oldLimit = userAmount[_token][_account].limit;
        userAmount[_token][_account].limit = _newLimit;

        emit UpdateCustomUserLimit(_token, _account, _oldLimit, _newLimit);
    }
}
