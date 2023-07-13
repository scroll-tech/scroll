// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

interface IScrollGateway {
    /// @notice The address of corresponding L1/L2 Gateway contract.
    function counterpart() external view returns (address);

    /// @notice The address of L1GatewayRouter/L2GatewayRouter contract.
    function router() external view returns (address);

    /// @notice The address of corresponding L1ScrollMessenger/L2ScrollMessenger contract.
    function messenger() external view returns (address);
}
