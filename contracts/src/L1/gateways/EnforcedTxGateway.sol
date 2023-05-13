// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {ECDSAUpgradeable} from "@openzeppelin/contracts-upgradeable/utils/cryptography/ECDSAUpgradeable.sol";
import {ReentrancyGuardUpgradeable} from "@openzeppelin/contracts-upgradeable/security/ReentrancyGuardUpgradeable.sol";
import {PausableUpgradeable} from "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";

import {IL1MessageQueue} from "../rollup/IL1MessageQueue.sol";

contract EnforcedTxGateway is OwnableUpgradeable, ReentrancyGuardUpgradeable, PausableUpgradeable {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates fee vault contract.
    /// @param _oldFeeVault The address of old fee vault contract.
    /// @param _newFeeVault The address of new fee vault contract.
    event UpdateFeeVault(address _oldFeeVault, address _newFeeVault);

    /*************
     * Variables *
     *************/

    /// @notice The address of L1MessageQueue contract.
    address public messageQueue;

    /// @notice The address of fee vault contract.
    address public feeVault;

    /***************
     * Constructor *
     ***************/

    function initialize(address _queue, address _feeVault) external initializer {
        messageQueue = _queue;
        feeVault = _feeVault;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Add an enforced transaction to L2.
    /// @dev The caller should be EOA only.
    /// @param _target The address of target contract to call in L2.
    /// @param _value The value passed
    /// @param _gasLimit The maximum gas should be used for this transaction in L2.
    /// @param _data The calldata passed to target contract.
    function sendTransaction(
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes calldata _data
    ) external payable whenNotPaused nonReentrant {
        require(msg.sender == tx.origin, "Only EOA senders are allowed to send enforced transaction");

        _sendTransaction(msg.sender, _target, _value, _gasLimit, _data);
    }

    /// @notice Add an enforced transaction to L2.
    /// @dev The `_sender` should be EOA and match with the signature.
    /// @param _sender The address of sender who will initiate this transaction in L2.
    /// @param _target The address of target contract to call in L2.
    /// @param _value The value passed
    /// @param _gasLimit The maximum gas should be used for this transaction in L2.
    /// @param _data The calldata passed to target contract.
    /// @param v The `v` value of signature.
    /// @param r The `r` value of signature.
    /// @param s The `s` value of signature.
    function sendTransaction(
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes calldata _data,
        uint8 v,
        bytes32 r,
        bytes32 s
    ) external payable whenNotPaused nonReentrant {
        address _messageQueue = messageQueue;
        uint256 _queueIndex = IL1MessageQueue(_messageQueue).nextCrossDomainMessageIndex();
        bytes32 _hash = IL1MessageQueue(messageQueue).computeTransactionHash(
            _sender,
            _queueIndex,
            _value,
            _target,
            _gasLimit,
            _data
        );

        address _signer = ECDSAUpgradeable.recover(_hash, v, r, s);
        // no need to check `_signer != address(0)`, since it is checked in `recover`.
        require(_signer == _sender, "Incorrect signature");

        _sendTransaction(_sender, _target, _value, _gasLimit, _data);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the address of fee vault.
    /// @param _newFeeVault The address to update.
    function updateFeeVault(address _newFeeVault) external onlyOwner {
        address _oldFeeVault = feeVault;
        feeVault = _newFeeVault;

        emit UpdateFeeVault(_oldFeeVault, _newFeeVault);
    }

    /// @notice Pause or unpause this contract.
    /// @param _status Pause this contract if it is true, otherwise unpause this contract.
    function setPaused(bool _status) external onlyOwner {
        if (_status) {
            _pause();
        } else {
            _unpause();
        }
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to charge fee and add enforced transaction.
    /// @param _sender The address of sender who will initiate this transaction in L2.
    /// @param _target The address of target contract to call in L2.
    /// @param _value The value passed
    /// @param _gasLimit The maximum gas should be used for this transaction in L2.
    /// @param _data The calldata passed to target contract.
    function _sendTransaction(
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes calldata _data
    ) internal nonReentrant {
        address _messageQueue = messageQueue;

        // charge fee
        uint256 _fee = IL1MessageQueue(_messageQueue).estimateCrossDomainMessageFee(_gasLimit);
        require(msg.value >= _fee, "Insufficient value for fee");
        if (_fee > 0) {
            (bool _success, ) = feeVault.call{value: _fee}("");
            require(_success, "Failed to deduct the fee");
        }

        // append transaction
        IL1MessageQueue(messageQueue).appendEnforcedTransaction(_sender, _target, _value, _gasLimit, _data);

        // refund fee to msg.sender
        unchecked {
            uint256 _refund = msg.value - _fee;
            if (_refund > 0) {
                (bool _success, ) = msg.sender.call{value: _refund}("");
                require(_success, "Failed to refund the fee");
            }
        }
    }
}
