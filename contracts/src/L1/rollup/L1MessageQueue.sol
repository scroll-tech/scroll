// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import {IL2GasPriceOracle} from "./IL2GasPriceOracle.sol";
import {IL1MessageQueue} from "./IL1MessageQueue.sol";

import {AddressAliasHelper} from "../../libraries/common/AddressAliasHelper.sol";

/// @title L1MessageQueue
/// @notice This contract will hold all L1 to L2 messages.
/// Each appended message is assigned with a unique and increasing `uint256` index denoting the message nonce.
contract L1MessageQueue is OwnableUpgradeable, IL1MessageQueue {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates gas oracle contract.
    /// @param _oldGasOracle The address of old gas oracle contract.
    /// @param _newGasOracle The address of new gas oracle contract.
    event UpdateGasOracle(address _oldGasOracle, address _newGasOracle);

    /*************
     * Variables *
     *************/

    /// @notice The address of L1ScrollMessenger contract.
    address public messenger;

    /// @notice The address of GasOracle contract.
    address public gasOracle;

    /// @notice The list of queued cross domain messages.
    bytes32[] public messageQueue;

    /***************
     * Constructor *
     ***************/

    function initialize(address _messenger, address _gasOracle) external initializer {
        OwnableUpgradeable.__Ownable_init();

        messenger = _messenger;
        gasOracle = _gasOracle;
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL1MessageQueue
    function nextCrossDomainMessageIndex() external view returns (uint256) {
        return messageQueue.length;
    }

    /// @inheritdoc IL1MessageQueue
    function getCrossDomainMessage(uint256 _queueIndex) external view returns (bytes32) {
        return messageQueue[_queueIndex];
    }

    /// @inheritdoc IL1MessageQueue
    function estimateCrossDomainMessageFee(uint256 _gasLimit) external view override returns (uint256) {
        address _oracle = gasOracle;
        if (_oracle == address(0)) return 0;
        return IL2GasPriceOracle(_oracle).estimateCrossDomainMessageFee(_gasLimit);
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL1MessageQueue
    function appendCrossDomainMessage(
        address _target,
        uint256 _gasLimit,
        bytes calldata _data
    ) external override {
        require(msg.sender == messenger, "Only callable by the L1ScrollMessenger");

        // do address alias to avoid replay attack in L2.
        address _sender = AddressAliasHelper.applyL1ToL2Alias(msg.sender);

        // @todo Change it to rlp encoding later.
        bytes32 _hash = keccak256(abi.encode(_sender, _target, 0, _gasLimit, _data));

        uint256 _queueIndex = messageQueue.length;
        emit QueueTransaction(_sender, _target, 0, _queueIndex, _gasLimit, _data);

        messageQueue.push(_hash);
    }

    /// @inheritdoc IL1MessageQueue
    function appendEnforcedTransaction(
        address,
        address,
        uint256,
        uint256,
        bytes calldata
    ) external override {
        // @todo
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the address of gas oracle.
    /// @dev This function can only called by contract owner.
    /// @param _newGasOracle The address to update.
    function updateGasOracle(address _newGasOracle) external onlyOwner {
        address _oldGasOracle = gasOracle;
        gasOracle = _newGasOracle;

        emit UpdateGasOracle(_oldGasOracle, _newGasOracle);
    }
}
