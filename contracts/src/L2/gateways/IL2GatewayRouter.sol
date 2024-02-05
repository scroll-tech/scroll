// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {IL2ETHGateway} from "./IL2ETHGateway.sol";
import {IL2ERC20Gateway} from "./IL2ERC20Gateway.sol";

interface IL2GatewayRouter is IL2ETHGateway, IL2ERC20Gateway {
    /**********
     * Events *
     **********/

    /// @notice Emitted when the address of default ERC20 Gateway is updated.
    /// @param oldDefaultERC20Gateway The address of the old default ERC20 Gateway.
    /// @param newDefaultERC20Gateway The address of the new default ERC20 Gateway.
    event SetDefaultERC20Gateway(address indexed oldDefaultERC20Gateway, address indexed newDefaultERC20Gateway);

    /// @notice Emitted when the `gateway` for `token` is updated.
    /// @param token The address of token updated.
    /// @param oldGateway The corresponding address of the old gateway.
    /// @param newGateway The corresponding address of the new gateway.
    event SetERC20Gateway(address indexed token, address indexed oldGateway, address indexed newGateway);

    /**********
     * Errors *
     **********/

    /// @dev Thrown when the given address is `address(0)`.
    error ErrorZeroAddress();
}
