// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {IL1ETHGateway} from "./IL1ETHGateway.sol";
import {IL1ERC20Gateway} from "./IL1ERC20Gateway.sol";

interface IL1GatewayRouter is IL1ETHGateway, IL1ERC20Gateway {
    /**********
     * Events *
     **********/

    /// @notice Emitted when the address of ETH Gateway is updated.
    /// @param ethGateway The address of new ETH Gateway.
    event SetETHGateway(address indexed ethGateway);

    /// @notice Emitted when the address of default ERC20 Gateway is updated.
    /// @param defaultERC20Gateway The address of new default ERC20 Gateway.
    event SetDefaultERC20Gateway(address indexed defaultERC20Gateway);

    /// @notice Emitted when the `gateway` for `token` is updated.
    /// @param token The address of token updated.
    /// @param gateway The corresponding address of gateway updated.
    event SetERC20Gateway(address indexed token, address indexed gateway);
}
