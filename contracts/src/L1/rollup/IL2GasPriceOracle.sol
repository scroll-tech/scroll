// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

interface IL2GasPriceOracle {
    /// @notice Estimate fee for cross chain message call.
    /// @param _gasLimit Gas limit required to complete the message relay on L2.
    function estimateCrossDomainMessageFee(uint256 _gasLimit) external view returns (uint256);

    /// @notice Estimate intrinsic gas fee for cross chain message call.
    /// @param _message The message to be relayed on L2.
    function calculateIntrinsicGasFee(bytes memory _message) external view returns (uint256);
}
