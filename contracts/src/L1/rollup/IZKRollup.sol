// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IZKRollup {
  /**********
   * Events *
   **********/

  /// @notice Emitted when a new batch is commited.
  /// @param _batchHash The hash of the batch
  /// @param _batchIndex The index of the batch
  /// @param _parentHash The hash of parent batch
  event CommitBatch(bytes32 indexed _batchId, bytes32 _batchHash, uint256 _batchIndex, bytes32 _parentHash);

  /// @notice Emitted when a batch is reverted.
  /// @param _batchId The identification of the batch.
  event RevertBatch(bytes32 indexed _batchId);

  /// @notice Emitted when a batch is finalized.
  /// @param _batchHash The hash of the batch
  /// @param _batchIndex The index of the batch
  /// @param _parentHash The hash of parent batch
  event FinalizeBatch(bytes32 indexed _batchId, bytes32 _batchHash, uint256 _batchIndex, bytes32 _parentHash);

  /***********
   * Structs *
   ***********/

  /// @dev The transanction struct
  struct Layer2Transaction {
    uint64 nonce;
    address target;
    uint64 gas;
    uint256 gasPrice;
    uint256 value;
    bytes data;
    // signature
    uint256 r;
    uint256 s;
    uint64 v;
  }

  /// @dev The block header struct
  struct Layer2BlockHeader {
    bytes32 blockHash;
    bytes32 parentHash;
    uint256 baseFee;
    bytes32 stateRoot;
    uint64 blockHeight;
    uint64 gasUsed;
    uint64 timestamp;
    bytes extraData;
    Layer2Transaction[] txs;
    bytes32 messageRoot;
  }

  /// @dev The batch struct, the batch hash is always the last block hash of `blocks`.
  struct Layer2Batch {
    uint64 batchIndex;
    // The hash of the last block in the parent batch
    bytes32 parentHash;
    Layer2BlockHeader[] blocks;
  }

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Return whether the block is finalized by block hash.
  /// @param blockHash The hash of the block to query.
  function isBlockFinalized(bytes32 blockHash) external view returns (bool);

  /// @notice Return whether the block is finalized by block height.
  /// @param blockHeight The height of the block to query.
  function isBlockFinalized(uint256 blockHeight) external view returns (bool);

  /// @notice Return the layer 2 block gas limit.
  /// @param _blockNumber The block number to query
  function layer2GasLimit(uint256 _blockNumber) external view returns (uint256);

  /// @notice Return the merkle root of L2 message tree.
  /// @param blockHash The hash of the block to query.
  function getL2MessageRoot(bytes32 blockHash) external view returns (bytes32);

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice commit a batch in layer 1
  /// @dev store in a more compacted form later.
  /// @param _batch The layer2 batch to commit.
  function commitBatch(Layer2Batch memory _batch) external;

  /// @notice revert a pending batch.
  /// @dev one can only revert unfinalized batches.
  /// @param _batchId The identification of the batch.
  function revertBatch(bytes32 _batchId) external;

  /// @notice finalize commited batch in layer 1
  /// @dev will add more parameters if needed.
  /// @param _batchId The identification of the commited batch.
  /// @param _proof The corresponding proof of the commited batch.
  /// @param _instances Instance used to verify, generated from batch.
  function finalizeBatchWithProof(
    bytes32 _batchId,
    uint256[] memory _proof,
    uint256[] memory _instances
  ) external;
}
