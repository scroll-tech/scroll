// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {PausableUpgradeable} from "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";
import {ReentrancyGuardUpgradeable} from "@openzeppelin/contracts-upgradeable/security/ReentrancyGuardUpgradeable.sol";

import {ScrollConstants} from "./constants/ScrollConstants.sol";
import {IETHRateLimiter} from "../rate-limiter/IETHRateLimiter.sol";
import {IScrollMessenger} from "./IScrollMessenger.sol";

// solhint-disable var-name-mixedcase

abstract contract ScrollMessengerBase is
    OwnableUpgradeable,
    PausableUpgradeable,
    ReentrancyGuardUpgradeable,
    IScrollMessenger
{
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates fee vault contract.
    /// @param _oldFeeVault The address of old fee vault contract.
    /// @param _newFeeVault The address of new fee vault contract.
    event UpdateFeeVault(address _oldFeeVault, address _newFeeVault);

    /// @notice Emitted when owner updates rate limiter contract.
    /// @param _oldRateLimiter The address of old rate limiter contract.
    /// @param _newRateLimiter The address of new rate limiter contract.
    event UpdateRateLimiter(address indexed _oldRateLimiter, address indexed _newRateLimiter);

    /*************
     * Variables *
     *************/

    /// @notice See {IScrollMessenger-xDomainMessageSender}
    address public override xDomainMessageSender;

    /// @notice The address of counterpart ScrollMessenger contract in L1/L2.
    address public counterpart;

    /// @notice The address of fee vault, collecting cross domain messaging fee.
    address public feeVault;

    /// @notice The address of ETH rate limiter contract.
    address public rateLimiter;

    /// @dev The storage slots for future usage.
    uint256[46] private __gap;

    /**********************
     * Function Modifiers *
     **********************/

    modifier notInExecution() {
        require(
            xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER,
            "Message is already in execution"
        );
        _;
    }

    /***************
     * Constructor *
     ***************/

    function __ScrollMessengerBase_init(address _counterpart, address _feeVault) internal onlyInitializing {
        OwnableUpgradeable.__Ownable_init();
        PausableUpgradeable.__Pausable_init();
        ReentrancyGuardUpgradeable.__ReentrancyGuard_init();

        // initialize to a nonzero value
        xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

        counterpart = _counterpart;
        if (_feeVault != address(0)) {
            feeVault = _feeVault;
        }
    }

    // make sure only owner can send ether to messenger to avoid possible user fund loss.
    receive() external payable onlyOwner {}

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update fee vault contract.
    /// @dev This function can only called by contract owner.
    /// @param _newFeeVault The address of new fee vault contract.
    function updateFeeVault(address _newFeeVault) external onlyOwner {
        address _oldFeeVault = feeVault;

        feeVault = _newFeeVault;
        emit UpdateFeeVault(_oldFeeVault, _newFeeVault);
    }

    /// @notice Update rate limiter contract.
    /// @dev This function can only called by contract owner.
    /// @param _newRateLimiter The address of new rate limiter contract.
    function updateRateLimiter(address _newRateLimiter) external onlyOwner {
        address _oldRateLimiter = rateLimiter;

        rateLimiter = _newRateLimiter;
        emit UpdateRateLimiter(_oldRateLimiter, _newRateLimiter);
    }

    /// @notice Pause the contract
    /// @dev This function can only called by contract owner.
    /// @param _status The pause status to update.
    function setPause(bool _status) external onlyOwner {
        if (_status) {
            _pause();
        } else {
            _unpause();
        }
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to generate the correct cross domain calldata for a message.
    /// @param _sender Message sender address.
    /// @param _target Target contract address.
    /// @param _value The amount of ETH pass to the target.
    /// @param _messageNonce Nonce for the provided message.
    /// @param _message Message to send to the target.
    /// @return ABI encoded cross domain calldata.
    function _encodeXDomainCalldata(
        address _sender,
        address _target,
        uint256 _value,
        uint256 _messageNonce,
        bytes memory _message
    ) internal pure returns (bytes memory) {
        return
            abi.encodeWithSignature(
                "relayMessage(address,address,uint256,uint256,bytes)",
                _sender,
                _target,
                _value,
                _messageNonce,
                _message
            );
    }

    /// @dev Internal function to increase ETH usage for the given `_sender`.
    /// @param _amount The amount of ETH used.
    function _addUsedAmount(uint256 _amount) internal {
        if (_amount == 0) return;

        address _rateLimiter = rateLimiter;
        if (_rateLimiter != address(0)) {
            IETHRateLimiter(_rateLimiter).addUsedAmount(_amount);
        }
    }

    /// @dev Internal function to check whether the `_target` address is blacklisted.
    /// @param _target The address of target address to check.
    function _validateTargetAddress(address _target) internal view {
        // @note check more `_target` address to avoid attack in the future when we add more external contracts.

        address _rateLimiter = rateLimiter;
        if (_rateLimiter != address(0)) {
            require(_target != _rateLimiter, "Forbid to call rate limiter");
        }
        require(_target != address(this), "Forbid to call self");
    }
}
