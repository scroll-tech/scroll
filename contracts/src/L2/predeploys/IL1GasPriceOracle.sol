// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

interface IL1GasPriceOracle {
    /**********
     * Events *
     **********/

    /// @notice Emitted when current fee overhead is updated.
    /// @param overhead The current fee overhead updated.
    event OverheadUpdated(uint256 overhead);

    /// @notice Emitted when current fee scalar is updated.
    /// @param scalar The current fee scalar updated.
    event ScalarUpdated(uint256 scalar);

    /// @notice Emitted when current commit fee scalar is updated.
    /// @param scalar The current commit fee scalar updated.
    event CommitScalarUpdated(uint256 scalar);

    /// @notice Emitted when current blob fee scalar is updated.
    /// @param scalar The current blob fee scalar updated.
    event BlobScalarUpdated(uint256 scalar);

    /// @notice Emitted when current l1 base fee is updated.
    /// @param l1BaseFee The current l1 base fee updated.
    event L1BaseFeeUpdated(uint256 l1BaseFee);

    /// @notice Emitted when current l1 blob base fee is updated.
    /// @param l1BlobBaseFee The current l1 blob base fee updated.
    event L1BlobBaseFeeUpdated(uint256 l1BlobBaseFee);

    /*************************
     * Public View Functions *
     *************************/

    /// @notice Return the current l1 fee overhead.
    function overhead() external view returns (uint256);

    /// @notice Return the current l1 fee scalar before Curie fork.
    function scalar() external view returns (uint256);

    /// @notice Return the current l1 commit fee scalar.
    function commitScalar() external view returns (uint256);

    /// @notice Return the current l1 blob fee scalar.
    function blobScalar() external view returns (uint256);

    /// @notice Return the latest known l1 base fee.
    function l1BaseFee() external view returns (uint256);

    /// @notice Return the latest known l1 blob base fee.
    function l1BlobBaseFee() external view returns (uint256);

    /// @notice Computes the L1 portion of the fee based on the size of the rlp encoded input
    ///         transaction, the current L1 base fee, and the various dynamic parameters.
    /// @param data Signed fully RLP-encoded transaction to get the L1 fee for.
    /// @return L1 fee that should be paid for the tx
    function getL1Fee(bytes memory data) external view returns (uint256);

    /// @notice Computes the amount of L1 gas used for a transaction. Adds the overhead which
    ///         represents the per-transaction gas overhead of posting the transaction and state
    ///         roots to L1. Adds 74 bytes of padding to account for the fact that the input does
    ///         not have a signature.
    /// @param data Signed fully RLP-encoded transaction to get the L1 gas for.
    /// @return Amount of L1 gas used to publish the transaction.
    function getL1GasUsed(bytes memory data) external view returns (uint256);

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Allows whitelisted caller to modify the l1 base fee.
    /// @param _l1BaseFee New l1 base fee.
    function setL1BaseFee(uint256 _l1BaseFee) external;

    /// @notice Allows whitelisted caller to modify the l1 base fee.
    /// @param _l1BaseFee New l1 base fee.
    /// @param _l1BlobBaseFee New l1 blob base fee.
    function setL1BaseFeeAndBlobBaseFee(uint256 _l1BaseFee, uint256 _l1BlobBaseFee) external;
}
