// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

interface IL2GasPriceOracle {
    /// @notice Return the latest known l2 base fee.
    function l2BaseFee() external view returns (uint256);

    /// @notice Return the address of whitelist contract.
    function whitelist() external view returns (address);

    /// @notice Estimate fee for cross chain message call.
    /// @param _gasLimit Gas limit required to complete the message relay on L2.
    function estimateCrossDomainMessageFee(uint256 _gasLimit) external view returns (uint256);

    /// @notice Estimate intrinsic gas fee for cross chain message call.
    /// @param _message The message to be relayed on L2.
    function calculateIntrinsicGasFee(bytes memory _message) external view returns (uint256);
}
