// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {IL2ETHGateway} from "./IL2ETHGateway.sol";
import {IL2ERC20Gateway} from "./IL2ERC20Gateway.sol";

interface IL2GatewayRouter is IL2ETHGateway, IL2ERC20Gateway {
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
