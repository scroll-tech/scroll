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
    uint256 fee,
    uint256 deadline,
    bytes message,
    uint256 messageNonce,
    uint256 gasLimit
  );

  event MessageDropped(bytes32 indexed msgHash);

  event RelayedMessage(bytes32 indexed msgHash);

  event FailedRelayedMessage(bytes32 indexed msgHash);

  /******************************
   * Events from L1MessageQueue *
   ******************************/

  /// @notice Emitted when a L1 to L2 message is appended.
  /// @param msgHash The hash of the appended message.
  event AppendMessage(bytes32 indexed msgHash);

  /************************
   * Events from ZKRollup *
   ************************/

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

  struct L2MessageProof {
    bytes32 blockHash;
    bytes32[] messageRootProof;
  }

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

  struct Layer2BatchStored {
    bytes32 batchHash;
    bytes32 parentHash;
    uint64 batchIndex;
    bool verified;
  }

  /*************
   * Variables *
   *************/

  /// @notice Message nonce, used to avoid relay attack.
  uint256 public messageNonce;

  /// @notice Mapping from batch id to batch struct.
  mapping(bytes32 => Layer2BatchStored) public batches;

  /************************************
   * Functions from L1ScrollMessenger *
   ************************************/

  function sendMessage(
    address _to,
    uint256 _fee,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable {
    // solhint-disable-next-line not-rely-on-time
    uint256 _deadline = block.timestamp + 1 days;
    uint256 _value;
    unchecked {
      _value = msg.value - _fee;
    }
    uint256 _nonce = messageNonce;
    bytes32 _msghash = keccak256(abi.encodePacked(msg.sender, _to, _value, _fee, _deadline, _nonce, _message));
    emit AppendMessage(_msghash);
    emit SentMessage(_to, msg.sender, _value, _fee, _deadline, _message, _nonce, _gasLimit);
    messageNonce += 1;
  }

  function relayMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message,
    L2MessageProof memory
  ) external {
    bytes32 _msghash = keccak256(abi.encodePacked(_from, _to, _value, _fee, _deadline, _nonce, _message));
    emit RelayedMessage(_msghash);
  }

  /***************************
   * Functions from ZKRollup *
   ***************************/

  function commitBatch(Layer2Batch memory _batch) external {
    bytes32 _batchHash = _batch.blocks[_batch.blocks.length - 1].blockHash;
    bytes32 _batchId = _computeBatchId(_batchHash, _batch.parentHash, _batch.batchIndex);

    Layer2BatchStored storage _batchStored = batches[_batchId];
    _batchStored.batchHash = _batchHash;
    _batchStored.parentHash = _batch.parentHash;
    _batchStored.batchIndex = _batch.batchIndex;

    emit CommitBatch(_batchId, _batchHash, _batch.batchIndex, _batch.parentHash);
  }
  
  function revertBatch(bytes32 _batchId) external {
    emit RevertBatch(_batchId);
  }

  function finalizeBatchWithProof(
    bytes32 _batchId,
    uint256[] memory,
    uint256[] memory
  ) external {
    Layer2BatchStored storage _batch = batches[_batchId];
    uint256 _batchIndex = _batch.batchIndex;

    emit FinalizeBatch(_batchId, _batch.batchHash, _batchIndex, _batch.parentHash);
  }

  /// @dev Internal function to compute a unique batch id for mapping.
  /// @param _batchHash The hash of the batch.
  /// @param _parentHash The hash of the batch.
  /// @param _batchIndex The index of the batch.
  /// @return Return the computed batch id.
  function _computeBatchId(
    bytes32 _batchHash,
    bytes32 _parentHash,
    uint256 _batchIndex
  ) internal pure returns (bytes32) {
    return keccak256(abi.encode(_batchHash, _parentHash, _batchIndex));
  }
}