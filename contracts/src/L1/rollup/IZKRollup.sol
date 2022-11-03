// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IZKRollup {
  /**************************************** Events ****************************************/

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

  /// @dev The transanction struct
  struct Layer2Transaction {
    address caller;
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
  }

  /// @dev The batch struct, the batch hash is always the last block hash of `blocks`.
  struct Layer2Batch {
    uint64 batchIndex;
    // The hash of the last block in the parent batch
    bytes32 parentHash;
    Layer2BlockHeader[] blocks;
  }

  /**************************************** View Functions ****************************************/

  /// @notice Return the message hash by index.
  /// @param _index The index to query.
  function getMessageHashByIndex(uint256 _index) external view returns (bytes32);

  /// @notice Return the index of the first queue element not yet executed.
  function getNextQueueIndex() external view returns (uint256);

  /// @notice Return the layer 2 block gas limit.
  /// @param _blockNumber The block number to query
  function layer2GasLimit(uint256 _blockNumber) external view returns (uint256);

  /// @notice Verify a state proof for message relay.
  /// @dev add more fields.
  function verifyMessageStateProof(uint256 _batchIndex, uint256 _blockHeight) external view returns (bool);

  /**************************************** Mutated Functions ****************************************/

  /// @notice Append a cross chain message to message queue.
  /// @dev This function should only be called by L1ScrollMessenger for safety.
  /// @param _sender The address of message sender in layer 1.
  /// @param _target The address of message recipient in layer 2.
  /// @param _value The amount of ether sent to recipient in layer 2.
  /// @param _fee The amount of ether paid to relayer in layer 2.
  /// @param _deadline The deadline of the message.
  /// @param _message The content of the message.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function appendMessage(
    address _sender,
    address _target,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    bytes memory _message,
    uint256 _gasLimit
  ) external returns (uint256);

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
