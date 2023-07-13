// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

interface IWhitelist {
    /// @notice Check whether the sender is allowed to do something.
    /// @param _sender The address of sender.
    function isSenderAllowed(address _sender) external view returns (bool);
}
