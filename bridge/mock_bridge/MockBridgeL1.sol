// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

contract MockBridgeL1 {
  /*********************************
   * Events from L1ScrollMessenger *
   *********************************/

  event SentMessage(
    address indexed target,
    address sender,
    uint256 value,
    uint256 messageNonce,
    bytes message
  );

  event RelayedMessage(bytes32 indexed messageHash);

  /************************
   * Events from ZKRollup *
   ************************/

  /// @notice Emitted when a new batch is commited.
  /// @param batchHash The hash of the batch
  event CommitBatch(bytes32 indexed batchHash);

  /// @notice Emitted when a batch is reverted.
  /// @param batchHash The hash of the batch
  event RevertBatch(bytes32 indexed batchHash);

  /// @notice Emitted when a batch is finalized.
  /// @param batchHash The hash of the batch
  event FinalizeBatch(bytes32 indexed batchHash);

  /***********
   * Structs *
   ***********/

  struct BlockContext {
    // The hash of this block.
    bytes32 blockHash;
    // The parent hash of this block.
    bytes32 parentHash;
    // The height of this block.
    uint64 blockNumber;
    // The timestamp of this block.
    uint64 timestamp;
    // The base fee of this block.
    // Currently, it is not used, because we disable EIP-1559.
    // We keep it for future proof.
    uint256 baseFee;
    // The gas limit of this block.
    uint64 gasLimit;
    // The number of transactions in this block, both L1 & L2 txs.
    uint16 numTransactions;
    // The number of l1 messages in this block.
    uint16 numL1Messages;
  }

  struct Batch {
    // The list of blocks in this batch
    BlockContext[] blocks; // MAX_NUM_BLOCKS = 100, about 5 min
    // The state root of previous batch.
    // The first batch will use 0x0 for prevStateRoot
    bytes32 prevStateRoot;
    // The state root of the last block in this batch.
    bytes32 newStateRoot;
    // The withdraw trie root of the last block in this batch.
    bytes32 withdrawTrieRoot;
    // The index of the batch.
    uint64 batchIndex;
    // Concatenated raw data of RLP encoded L2 txs
    bytes l2Transactions;
  }

  struct L2MessageProof {
    bytes32 batchHash;
    bytes merkleProof;
  }

  /*************
   * Variables *
   *************/

  /// @notice Message nonce, used to avoid relay attack.
  uint256 public messageNonce;

  /************************************
   * Functions from L1ScrollMessenger *
   ************************************/

  function sendMessage(
    address target,
    uint256 value,
    bytes calldata message,
    uint256 gasLimit
  ) external payable {
    emit SentMessage(msg.sender, target, value, messageNonce, message);
    messageNonce += 1;
  }

  function relayMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _nonce,
    bytes memory _message,
    L2MessageProof memory
  ) external {
    bytes32 _msghash = keccak256(abi.encodePacked(_from, _to, _value, _nonce, _message));
    emit RelayedMessage(_msghash);
  }

  /***************************
   * Functions from ZKRollup *
   ***************************/

  function commitBatch(Batch memory _batch) external {
    _commitBatch(_batch);
  }

  function commitBatches(Batch[] memory _batches) external {
    for (uint256 i = 0; i < _batches.length; i++) {
      _commitBatch(_batches[i]);
    }
  }
  
  function revertBatch(bytes32 _batchHash) external {
    emit RevertBatch(_batchHash);
  }

  function finalizeBatchWithProof(
    bytes32 _batchHash,
    uint256[] memory _proof,
    uint256[] memory _instances
  ) external {
    emit FinalizeBatch(_batchHash);
  }

  function _commitBatch(Batch memory _batch) internal {
    bytes32 _batchHash = _batch.blocks[_batch.blocks.length - 1].blockHash;
    emit CommitBatch(_batchHash);
  }
}
