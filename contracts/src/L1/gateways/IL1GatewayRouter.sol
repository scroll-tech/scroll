// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

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

    /*************************
     * Public View Functions *
     *************************/

    /// @notice Return the corresponding gateway address for given token address.
    /// @param _token The address of token to query.
    function getERC20Gateway(address _token) external view returns (address);

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Request ERC20 token transfer from users to gateways.
    /// @param sender The address of sender to request fund.
    /// @param token The address of token to request.
    /// @param amount The amount of token to request.
    function requestERC20(
        address sender,
        address token,
        uint256 amount
    ) external returns (uint256);

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the address of ETH gateway contract.
    /// @dev This function should only be called by contract owner.
    /// @param _ethGateway The address to update.
    function setETHGateway(address _ethGateway) external;

    /// @notice Update the address of default ERC20 gateway contract.
    /// @dev This function should only be called by contract owner.
    /// @param _defaultERC20Gateway The address to update.
    function setDefaultERC20Gateway(address _defaultERC20Gateway) external;

    /// @notice Update the mapping from token address to gateway address.
    /// @dev This function should only be called by contract owner.
    /// @param _tokens The list of addresses of tokens to update.
    /// @param _gateways The list of addresses of gateways to update.
    function setERC20Gateway(address[] memory _tokens, address[] memory _gateways) external;
}
