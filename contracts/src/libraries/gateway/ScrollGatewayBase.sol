// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {ReentrancyGuardUpgradeable} from "@openzeppelin/contracts-upgradeable/security/ReentrancyGuardUpgradeable.sol";

import {IScrollGateway} from "./IScrollGateway.sol";
import {IScrollMessenger} from "../IScrollMessenger.sol";
import {IScrollGatewayCallback} from "../callbacks/IScrollGatewayCallback.sol";
import {ScrollConstants} from "../constants/ScrollConstants.sol";
import {ITokenRateLimiter} from "../../rate-limiter/ITokenRateLimiter.sol";

/// @title ScrollGatewayBase
/// @notice The `ScrollGatewayBase` is a base contract for gateway contracts used in both in L1 and L2.
abstract contract ScrollGatewayBase is ReentrancyGuardUpgradeable, OwnableUpgradeable, IScrollGateway {
    /*************
     * Constants *
     *************/

    /// @inheritdoc IScrollGateway
    address public immutable override counterpart;

    /// @inheritdoc IScrollGateway
    address public immutable override router;

    /// @inheritdoc IScrollGateway
    address public immutable override messenger;

    /*************
     * Variables *
     *************/

    /// @dev The storage slot used as counterpart gateway contract, which is deprecated now.
    address private __counterpart;

    /// @dev The storage slot used as gateway router contract, which is deprecated now.
    address private __router;

    /// @dev The storage slot used as scroll messenger contract, which is deprecated now.
    address private __messenger;

    /// @dev The storage slot used as token rate limiter contract, which is deprecated now.
    address private __rateLimiter;

    /// @dev The storage slots for future usage.
    uint256[46] private __gap;

    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyCallByCounterpart() {
        // check caller is messenger
        if (_msgSender() != messenger) {
            revert ErrorCallerIsNotMessenger();
        }

        // check cross domain caller is counterpart gateway
        if (counterpart != IScrollMessenger(messenger).xDomainMessageSender()) {
            revert ErrorCallerIsNotCounterpartGateway();
        }
        _;
    }

    modifier onlyInDropContext() {
        // check caller is messenger
        if (_msgSender() != messenger) {
            revert ErrorCallerIsNotMessenger();
        }

        // check we are dropping message in ScrollMessenger.
        if (ScrollConstants.DROP_XDOMAIN_MESSAGE_SENDER != IScrollMessenger(messenger).xDomainMessageSender()) {
            revert ErrorNotInDropMessageContext();
        }
        _;
    }

    /***************
     * Constructor *
     ***************/

    constructor(
        address _counterpart,
        address _router,
        address _messenger
    ) {
        if (_counterpart == address(0) || _messenger == address(0)) {
            revert ErrorZeroAddress();
        }

        counterpart = _counterpart;
        router = _router;
        messenger = _messenger;
    }

    function _initialize(
        address,
        address,
        address
    ) internal {
        ReentrancyGuardUpgradeable.__ReentrancyGuard_init();
        OwnableUpgradeable.__Ownable_init();
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to forward calldata to target contract.
    /// @param _to The address of contract to call.
    /// @param _data The calldata passed to the contract.
    function _doCallback(address _to, bytes memory _data) internal {
        if (_data.length > 0 && _to.code.length > 0) {
            IScrollGatewayCallback(_to).onScrollGatewayCallback(_data);
        }
    }
}
