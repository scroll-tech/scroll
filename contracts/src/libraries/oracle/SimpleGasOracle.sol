// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import "./IGasOracle.sol";

/// @title Simple Gas Oracle
contract SimpleGasOracle is OwnableUpgradeable, IGasOracle {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner update default FeeConfig.
    /// @param _baseFees The amount base fee to pay.
    /// @param _feesPerByte The amount fee to pay per message byte.
    event UpdateDefaultFeeConfig(uint256 _baseFees, uint256 _feesPerByte);

    /// @notice Emitted when owner update custom FeeConfig.
    /// @param _sender The address of custom message sender.
    /// @param _baseFees The amount base fee to pay.
    /// @param _feesPerByte The amount fee to pay per message byte.
    event UpdateCustomFeeConfig(address indexed _sender, uint256 _baseFees, uint256 _feesPerByte);

    /*************
     * Variables *
     *************/

    struct FeeConfig {
        uint128 baseFees;
        uint128 feesPerByte;
    }

    /// @notice The default cross chain message FeeConfig.
    FeeConfig public defaultFeeConfig;

    /// @notice Mapping from sender address to custom FeeConfig.
    mapping(address => FeeConfig) public customFeeConfig;

    /// @notice Whether the sender should user custom FeeConfig.
    mapping(address => bool) public hasCustomConfig;

    /***************
     * Constructor *
     ***************/

    function initialize() external initializer {
        OwnableUpgradeable.__Ownable_init();
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IGasOracle
    function estimateMessageFee(
        address _sender,
        address,
        bytes memory _message,
        uint256
    ) external view override returns (uint256) {
        FeeConfig memory _feeConfig;
        if (hasCustomConfig[_sender]) {
            _feeConfig = customFeeConfig[_sender];
        } else {
            _feeConfig = defaultFeeConfig;
        }

        unchecked {
            return _feeConfig.baseFees + uint256(_feeConfig.feesPerByte) * _message.length;
        }
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update default fee config.
    /// @param _baseFees The amount of baseFees to update.
    /// @param _feesPerByte The amount of fees per byte to update.
    function updateDefaultFeeConfig(uint128 _baseFees, uint128 _feesPerByte) external onlyOwner {
        defaultFeeConfig = FeeConfig(_baseFees, _feesPerByte);

        emit UpdateDefaultFeeConfig(_baseFees, _feesPerByte);
    }

    /// @notice Update custom fee config for sender.
    /// @param _sender The address of sender to update custom FeeConfig.
    /// @param _baseFees The amount of baseFees to update.
    /// @param _feesPerByte The amount of fees per byte to update.
    function updateCustomFeeConfig(
        address _sender,
        uint128 _baseFees,
        uint128 _feesPerByte
    ) external onlyOwner {
        customFeeConfig[_sender] = FeeConfig(_baseFees, _feesPerByte);
        hasCustomConfig[_sender] = true;

        emit UpdateCustomFeeConfig(_sender, _baseFees, _feesPerByte);
    }
}
