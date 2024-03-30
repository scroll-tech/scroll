// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

import {IL2ETHGateway} from "./IL2ETHGateway.sol";
import {IL2ERC20Gateway} from "./IL2ERC20Gateway.sol";

interface IL2GatewayRouter is IL2ETHGateway, IL2ERC20Gateway {
    /**********
     * Events *
     **********/

    /// @notice Emitted when the address of ETH Gateway is updated.
    /// @param oldETHGateway The address of the old ETH Gateway.
    /// @param newEthGateway The address of the new ETH Gateway.
    event SetETHGateway(address indexed oldETHGateway, address indexed newEthGateway);

    /// @notice Emitted when the address of default ERC20 Gateway is updated.
    /// @param oldDefaultERC20Gateway The address of the old default ERC20 Gateway.
    /// @param newDefaultERC20Gateway The address of the new default ERC20 Gateway.
    event SetDefaultERC20Gateway(address indexed oldDefaultERC20Gateway, address indexed newDefaultERC20Gateway);

    /// @notice Emitted when the `gateway` for `token` is updated.
    /// @param token The address of token updated.
    /// @param oldGateway The corresponding address of the old gateway.
    /// @param newGateway The corresponding address of the new gateway.
    event SetERC20Gateway(address indexed token, address indexed oldGateway, address indexed newGateway);

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the address of ETH gateway contract.
    /// @dev This function should only be called by contract owner.
    /// @param _newEthGateway The address to update.
    function setETHGateway(address _newEthGateway) external;

    /// @notice Update the address of default ERC20 gateway contract.
    /// @dev This function should only be called by contract owner.
    /// @param _newDefaultERC20Gateway The address to update.
    function setDefaultERC20Gateway(address _newDefaultERC20Gateway) external;

    /// @notice Update the mapping from token address to gateway address.
    /// @dev This function should only be called by contract owner.
    /// @param _tokens The list of addresses of tokens to update.
    /// @param _gateways The list of addresses of gateways to update.
    function setERC20Gateway(address[] calldata _tokens, address[] calldata _gateways) external;
}
