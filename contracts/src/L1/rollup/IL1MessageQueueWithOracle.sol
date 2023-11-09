// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {IL1MessageQueue} from "./IL1MessageQueue.sol";

interface IL1MessageQueueWithOracle is IL1MessageQueue {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates whitelist contract.
    /// @param _oldWhitelist The address of old whitelist contract.
    /// @param _newWhitelist The address of new whitelist contract.
    event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

    /// @notice Emitted when current l2 base fee is updated.
    /// @param oldL2BaseFee The original l2 base fee before update.
    /// @param newL2BaseFee The current l2 base fee updated.
    event L2BaseFeeUpdated(uint256 oldL2BaseFee, uint256 newL2BaseFee);

    /// @notice Emitted when intrinsic params are updated.
    /// @param txGas The intrinsic gas for transaction.
    /// @param txGasContractCreation The intrinsic gas for contract creation.
    /// @param zeroGas The intrinsic gas for each zero byte.
    /// @param nonZeroGas The intrinsic gas for each nonzero byte.
    event IntrinsicParamsUpdated(uint256 txGas, uint256 txGasContractCreation, uint256 zeroGas, uint256 nonZeroGas);

    /**********
     * Errors *
     **********/

    /// @dev Thrown when the caller is not whitelisted.
    error ErrorNotWhitelistedSender();

    /// @dev Thrown when the given `txGas` is zero.
    error ErrorTxGasIsZero();

    /// @dev Thrown when the given `zeroGas` is zero.
    error ErrorZeroGasIsZero();

    /// @dev Thrown when the given `nonZeroGas` is zero.
    error ErrorNonZeroGasIsZero();

    /// @dev Thrown when the given `txGasContractCreation` is smaller than `txGas`.
    error ErrorTxGasContractCreationLessThanTxGas();
}
