// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IL2GasPriceOracle {
    /// @notice Estimate fee for cross chain message call.
    /// @param _gasLimit Gas limit required to complete the message relay on L2.
    function estimateCrossDomainMessageFee(uint256 _gasLimit) external view returns (uint256);
}
