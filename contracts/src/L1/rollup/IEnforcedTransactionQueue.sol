// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IEnforcedTransactionQueue {
  /**********
   * Events *
   **********/

  /// @notice Emitted when an enforced transation is appended.
  /// @param transactionHash The hash of the appended transaction.
  /// @param rawTx The RLP encoding of transaction should be submitted to L2.
  event EnqueueTransaction(bytes32 indexed transactionHash, bytes rawTx);

  /// @notice Emitted when some transactions are included in L2.
  /// @param fromIndex The start index of `transactionQueue`, inclusive.
  /// @param toIndex The end index of `transactionQueue`, not inclusive.
  event IncludeTransaction(uint256 fromIndex, uint256 toIndex);

  /// @notice Emitted when some transactions are included in L2 by owner.
  /// @param fromIndex The start index of `transactionQueue`, inclusive.
  /// @param toIndex The end index of `transactionQueue`, not inclusive.
  event ForceIncludeTransaction(uint256 fromIndex, uint256 toIndex);

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Return whether the transaction is expired.
  /// @param l2Timestamp The L2 block timestamp of the transaction.
  /// @param index The index of the transaction in `transactionQueue`.
  /// @param transactionHash The hash of the transaction.
  function isTransactionExpired(
    uint256 l2Timestamp,
    uint256 index,
    bytes32 transactionHash
  ) external view returns (bool);

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Enqueue an enforced transation.
  /// @param rawTx The RLP encoding of the enforced transation.
  function enqueueTransaction(bytes calldata rawTx) external payable;

  /// @notice Include enforced transaction in L2 by ZK Rollup contract.
  /// @param nextIndex The next unincluded transaction index.
  function includeTransaction(uint256 nextIndex) external;
}
