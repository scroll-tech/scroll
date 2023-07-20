// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

interface IScrollStandardERC20Factory {
    /// @notice Emitted when a l2 token is deployed.
    /// @param _l1Token The address of the l1 token.
    /// @param _l2Token The address of the l2 token.
    event DeployToken(address indexed _l1Token, address indexed _l2Token);

    /// @notice Compute the corresponding l2 token address given l1 token address.
    /// @param _gateway The address of gateway contract.
    /// @param _l1Token The address of l1 token.
    function computeL2TokenAddress(address _gateway, address _l1Token) external view returns (address);

    /// @notice Deploy the corresponding l2 token address given l1 token address.
    /// @param _gateway The address of gateway contract.
    /// @param _l1Token The address of l1 token.
    function deployL2Token(address _gateway, address _l1Token) external returns (address);
}
