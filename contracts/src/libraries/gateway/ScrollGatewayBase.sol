// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {ReentrancyGuardUpgradeable} from "@openzeppelin/contracts-upgradeable/security/ReentrancyGuardUpgradeable.sol";

import {IScrollGateway} from "./IScrollGateway.sol";
import {IScrollMessenger} from "../IScrollMessenger.sol";
import {IScrollGatewayCallback} from "../callbacks/IScrollGatewayCallback.sol";
import {ScrollConstants} from "../constants/ScrollConstants.sol";
import {ITokenRateLimiter} from "../../rate-limiter/ITokenRateLimiter.sol";

abstract contract ScrollGatewayBase is ReentrancyGuardUpgradeable, OwnableUpgradeable, IScrollGateway {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates rate limiter contract.
    /// @param _oldRateLimiter The address of old rate limiter contract.
    /// @param _newRateLimiter The address of new rate limiter contract.
    event UpdateRateLimiter(address indexed _oldRateLimiter, address indexed _newRateLimiter);

    /*************
     * Variables *
     *************/

    /// @inheritdoc IScrollGateway
    address public override counterpart;

    /// @inheritdoc IScrollGateway
    address public override router;

    /// @inheritdoc IScrollGateway
    address public override messenger;

    /// @notice The address of token rate limiter contract.
    address public rateLimiter;

    /// @dev The storage slots for future usage.
    uint256[46] private __gap;

    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyCallByCounterpart() {
        address _messenger = messenger; // gas saving
        require(msg.sender == _messenger, "only messenger can call");
        require(counterpart == IScrollMessenger(_messenger).xDomainMessageSender(), "only call by counterpart");
        _;
    }

    modifier onlyInDropContext() {
        address _messenger = messenger; // gas saving
        require(msg.sender == _messenger, "only messenger can call");
        require(
            ScrollConstants.DROP_XDOMAIN_MESSAGE_SENDER == IScrollMessenger(_messenger).xDomainMessageSender(),
            "only called in drop context"
        );
        _;
    }

    /***************
     * Constructor *
     ***************/

    function _initialize(
        address _counterpart,
        address _router,
        address _messenger
    ) internal {
        require(_counterpart != address(0), "zero counterpart address");
        require(_messenger != address(0), "zero messenger address");

        ReentrancyGuardUpgradeable.__ReentrancyGuard_init();
        OwnableUpgradeable.__Ownable_init();

        counterpart = _counterpart;
        messenger = _messenger;

        // @note: the address of router could be zero, if this contract is GatewayRouter.
        if (_router != address(0)) {
            router = _router;
        }
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update rate limiter contract.
    /// @dev This function can only called by contract owner.
    /// @param _newRateLimiter The address of new rate limiter contract.
    function updateRateLimiter(address _newRateLimiter) external onlyOwner {
        address _oldRateLimiter = rateLimiter;

        rateLimiter = _newRateLimiter;
        emit UpdateRateLimiter(_oldRateLimiter, _newRateLimiter);
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

    /// @dev Internal function to increase token usage for the given `_sender`.
    /// @param _token The address of token.
    /// @param _amount The amount of token used.
    function _addUsedAmount(address _token, uint256 _amount) internal {
        if (_amount == 0) return;

        address _rateLimiter = rateLimiter;
        if (_rateLimiter != address(0)) {
            ITokenRateLimiter(_rateLimiter).addUsedAmount(_token, _amount);
        }
    }
}
