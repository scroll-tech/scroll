// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.24;

import {BatchHeaderV0Codec} from "../../../contracts/src/libraries/codec/BatchHeaderV0Codec.sol";
import {ChunkCodec} from "../../../contracts/src/libraries/codec/ChunkCodec.sol";
import {IL1MessageQueue} from "../../../contracts/src/L1/rollup/IL1MessageQueue.sol";

contract MockBridge {
  /******************************
   * Events from L1MessageQueue *
   ******************************/

  /// @notice Emitted when a new L1 => L2 transaction is appended to the queue.
  /// @param sender The address of account who initiates the transaction.
  /// @param target The address of account who will recieve the transaction.
  /// @param value The value passed with the transaction.
  /// @param queueIndex The index of this transaction in the queue.
  /// @param gasLimit Gas limit required to complete the message relay on L2.
  /// @param data The calldata of the transaction.
  event QueueTransaction(
    address indexed sender,
    address indexed target,
    uint256 value,
    uint64 queueIndex,
    uint256 gasLimit,
    bytes data
  );

  /*********************************
   * Events from L1ScrollMessenger *
   *********************************/

  /// @notice Emitted when a cross domain message is sent.
  /// @param sender The address of the sender who initiates the message.
  /// @param target The address of target contract to call.
  /// @param value The amount of value passed to the target contract.
  /// @param messageNonce The nonce of the message.
  /// @param gasLimit The optional gas limit passed to L1 or L2.
  /// @param message The calldata passed to the target contract.
  event SentMessage(
    address indexed sender,
    address indexed target,
    uint256 value,
    uint256 messageNonce,
    uint256 gasLimit,
    bytes message
  );

  /// @notice Emitted when a cross domain message is relayed successfully.
  /// @param messageHash The hash of the message.
  event RelayedMessage(bytes32 indexed messageHash);

  /***************************
   * Events from ScrollChain *
   ***************************/

  /// @notice Emitted when a new batch is committed.
  /// @param batchHash The hash of the batch.
  event CommitBatch(uint256 indexed batchIndex, bytes32 indexed batchHash);

  /// @notice Emitted when a batch is finalized.
  /// @param batchHash The hash of the batch
  /// @param stateRoot The state root on layer 2 after this batch.
  /// @param withdrawRoot The merkle root on layer2 after this batch.
  event FinalizeBatch(uint256 indexed batchIndex, bytes32 indexed batchHash, bytes32 stateRoot, bytes32 withdrawRoot);

  /***********
   * Structs *
   ***********/

  struct L2MessageProof {
    // The index of the batch where the message belongs to.
    uint256 batchIndex;
    // Concatenation of merkle proof for withdraw merkle trie.
    bytes merkleProof;
  }

  /*************
   * Variables *
   *************/

  uint256 public messageNonce;
  mapping(uint256 => bytes32) public committedBatches;
  uint256 public l2BaseFee;

  /***********************************
   * Functions from L2GasPriceOracle *
   ***********************************/

  function setL2BaseFee(uint256 _newL2BaseFee) external {
    l2BaseFee = _newL2BaseFee;
  }

  /************************************
   * Functions from L1ScrollMessenger *
   ************************************/

  function sendMessage(
    address target,
    uint256 value,
    bytes calldata message,
    uint256 gasLimit
  ) external payable {
    bytes memory _xDomainCalldata = _encodeXDomainCalldata(msg.sender, target, value, messageNonce, message);
    {
      address _sender = applyL1ToL2Alias(address(this));
      emit QueueTransaction(_sender, target, 0, uint64(messageNonce), gasLimit, _xDomainCalldata);
    }

    emit SentMessage(msg.sender, target, value, messageNonce, gasLimit, message);
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
    bytes memory _xDomainCalldata = _encodeXDomainCalldata(_from, _to, _value, _nonce, _message);
    bytes32 _xDomainCalldataHash = keccak256(_xDomainCalldata);
    emit RelayedMessage(_xDomainCalldataHash);
  }

  /******************************
   * Functions from ScrollChain *
   ******************************/

  /// @notice Import layer 2 genesis block
  function importGenesisBatch(bytes calldata _batchHeader, bytes32 _stateRoot) external {
  }

  function commitBatch(
    uint8 /*version*/,
    bytes calldata _parentBatchHeader,
    bytes[] memory chunks,
    bytes calldata /*skippedL1MessageBitmap*/
  ) external {
    // check whether the batch is empty
    uint256 _chunksLength = chunks.length;
    require(_chunksLength > 0, "batch is empty");

    // decode batch index
    uint256 headerLength = _parentBatchHeader.length;
    uint256 parentBatchPtr;
    uint256 parentBatchIndex;
    assembly {
      parentBatchPtr := mload(0x40)
      calldatacopy(parentBatchPtr, _parentBatchHeader.offset, headerLength)
      mstore(0x40, add(parentBatchPtr, headerLength))
      parentBatchIndex := shr(192, mload(add(parentBatchPtr, 1)))
    }

    uint256 dataPtr;
    assembly {
      dataPtr := mload(0x40)
      mstore(0x40, add(dataPtr, mul(_chunksLength, 32)))
    }

    for (uint256 i = 0; i < _chunksLength; i++) {
      _commitChunk(dataPtr, chunks[i]);

      unchecked {
        dataPtr += 32;
      }
    }

    bytes32 _dataHash;
    assembly {
      let dataLen := mul(_chunksLength, 0x20)
      _dataHash := keccak256(sub(dataPtr, dataLen), dataLen)
    }

    bytes memory paddedData = new bytes(89);
    assembly {
      mstore(add(paddedData, 57), _dataHash)
    }

    uint256 batchPtr;
    assembly {
      batchPtr := add(paddedData, 32)
    }
    bytes32 _batchHash = BatchHeaderV0Codec.computeBatchHash(batchPtr, 89);
    committedBatches[0] = _batchHash;
    emit CommitBatch(parentBatchIndex + 1, _batchHash);
  }

  function finalizeBatchWithProof(
    bytes calldata batchHeader,
    bytes32 /*prevStateRoot*/,
    bytes32 postStateRoot,
    bytes32 withdrawRoot,
    bytes calldata /*aggrProof*/
  ) external {
    // decode batch index
    uint256 headerLength = batchHeader.length;
    uint256 batchPtr;
    uint256 batchIndex;
    assembly {
      batchPtr := mload(0x40)
      calldatacopy(batchPtr, batchHeader.offset, headerLength)
      mstore(0x40, add(batchPtr, headerLength))
      batchIndex := shr(192, mload(add(batchPtr, 1)))
    }

    bytes32 _batchHash = committedBatches[0];
    emit FinalizeBatch(batchIndex, _batchHash, postStateRoot, withdrawRoot);
  }

  /**********************
   * Internal Functions *
   **********************/

  /// @dev Internal function to generate the correct cross domain calldata for a message.
  /// @param _sender Message sender address.
  /// @param _target Target contract address.
  /// @param _value The amount of ETH pass to the target.
  /// @param _messageNonce Nonce for the provided message.
  /// @param _message Message to send to the target.
  /// @return ABI encoded cross domain calldata.
  function _encodeXDomainCalldata(
    address _sender,
    address _target,
    uint256 _value,
    uint256 _messageNonce,
    bytes memory _message
  ) internal pure returns (bytes memory) {
    return
      abi.encodeWithSignature(
        "relayMessage(address,address,uint256,uint256,bytes)",
        _sender,
        _target,
        _value,
        _messageNonce,
        _message
      );
  }

  /// @notice Utility function that converts the address in the L1 that submitted a tx to
  /// the inbox to the msg.sender viewed in the L2
  /// @param l1Address the address in the L1 that triggered the tx to L2
  /// @return l2Address L2 address as viewed in msg.sender
  function applyL1ToL2Alias(address l1Address) internal pure returns (address l2Address) {
    uint160 offset = uint160(0x1111000000000000000000000000000000001111);
    unchecked {
      l2Address = address(uint160(l1Address) + offset);
    }
  }

  function _commitChunk(
    uint256 memPtr,
    bytes memory _chunk
  ) internal pure {
    uint256 chunkPtr;
    uint256 startDataPtr;
    uint256 dataPtr;
    uint256 blockPtr;

    assembly {
      dataPtr := mload(0x40)
      startDataPtr := dataPtr
      chunkPtr := add(_chunk, 0x20) // skip chunkLength
      blockPtr := add(chunkPtr, 1) // skip numBlocks
    }

    uint256 _numBlocks = ChunkCodec.validateChunkLength(chunkPtr, _chunk.length);

    // concatenate block contexts
    uint256 _totalTransactionsInChunk;
    for (uint256 i = 0; i < _numBlocks; i++) {
      dataPtr = ChunkCodec.copyBlockContext(chunkPtr, dataPtr, i);
      uint256 _numTransactionsInBlock = ChunkCodec.numTransactions(blockPtr);
      unchecked {
        _totalTransactionsInChunk += _numTransactionsInBlock;
        blockPtr += ChunkCodec.BLOCK_CONTEXT_LENGTH;
      }
    }

    assembly {
      mstore(0x40, add(dataPtr, mul(_totalTransactionsInChunk, 0x20))) // reserve memory for tx hashes
      blockPtr := add(chunkPtr, 1) // reset block ptr
    }

    // concatenate tx hashes
    uint256 l2TxPtr = ChunkCodec.l2TxPtr(chunkPtr, _numBlocks);
    while (_numBlocks > 0) {
      // concatenate l2 transaction hashes
      uint256 _numTransactionsInBlock = ChunkCodec.numTransactions(blockPtr);
      for (uint256 j = 0; j < _numTransactionsInBlock; j++) {
        bytes32 txHash;
        (txHash, l2TxPtr) = ChunkCodec.loadL2TxHash(l2TxPtr);
        assembly {
          mstore(dataPtr, txHash)
          dataPtr := add(dataPtr, 0x20)
        }
      }

      unchecked {
        _numBlocks -= 1;
        blockPtr += ChunkCodec.BLOCK_CONTEXT_LENGTH;
      }
    }

    // check chunk has correct length
    require(l2TxPtr - chunkPtr == _chunk.length, "incomplete l2 transaction data");

    // compute data hash and store to memory
    assembly {
      let dataHash := keccak256(startDataPtr, sub(dataPtr, startDataPtr))
      mstore(memPtr, dataHash)
    }
  }

  address private constant POINT_EVALUATION_PRECOMPILE_ADDRESS = 0x000000000000000000000000000000000000000A;
  uint256 private constant BLS_MODULUS = 52435875175126190479447740508185965837690552500527637822603658699938581184513;

  function verifyProof(
    bytes32 claim,
    bytes memory commitment,
    bytes memory proof
  ) external view {
    require(commitment.length == 48, "Commitment must be 48 bytes");
    require(proof.length == 48, "Proof must be 48 bytes");

    bytes32 versionedHash = blobhash(0);

    // Compute random challenge point.
    uint256 point = uint256(keccak256(abi.encodePacked(versionedHash))) % BLS_MODULUS;

    bytes memory pointEvaluationCalldata = abi.encodePacked(
      versionedHash,
      point,
      claim,
      commitment,
      proof
    );

    (bool success,) = POINT_EVALUATION_PRECOMPILE_ADDRESS.staticcall(pointEvaluationCalldata);

    if (!success) {
      revert("Proof verification failed");
    }
  }
}
