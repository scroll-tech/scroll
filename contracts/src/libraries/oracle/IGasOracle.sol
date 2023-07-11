// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IGasOracle {
    /// @notice Estimate fee for cross chain message call.
    /// @param _sender The address of sender who invoke the call.
    /// @param _to The target address to receive the call.
    /// @param _message The message will be passed to the target address.
    function estimateMessageFee(
        address _sender,
        address _to,
        bytes memory _message,
        uint256 _gasLimit
    ) external view returns (uint256);
}
