// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {IL1MessageQueue} from "./IL1MessageQueue.sol";

interface IL1MessageQueueWithGasPriceOracle is IL1MessageQueue {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates whitelist contract.
    /// @param _oldWhitelist The address of old whitelist contract.
    /// @param _newWhitelist The address of new whitelist contract.
    event UpdateWhitelist(address indexed _oldWhitelist, address indexed _newWhitelist);

    /// @notice Emitted when current l2 base fee is updated.
    /// @param oldL2BaseFee The original l2 base fee before update.
    /// @param newL2BaseFee The current l2 base fee updated.
    event UpdateL2BaseFee(uint256 oldL2BaseFee, uint256 newL2BaseFee);

    /**********
     * Errors *
     **********/

    /// @dev Thrown when the caller is not whitelisted.
    error ErrorNotWhitelistedSender();
}
