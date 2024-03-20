// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.24;

import {BatchHeaderV0Codec} from "../../../contracts/src/libraries/codec/BatchHeaderV0Codec.sol";
import {ChunkCodec} from "../../../contracts/src/libraries/codec/ChunkCodec.sol";

contract MockBridge {
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
  bytes32 public committedBatchHash;
  uint256 public l2BaseFee;

  function setL2BaseFee(uint256 _newL2BaseFee) external {
    l2BaseFee = _newL2BaseFee;
  }

  function importGenesisBatch(bytes calldata _batchHeader, bytes32 _stateRoot) external {}

  function commitBatch(
    uint8 /*version*/,
    bytes calldata _parentBatchHeader,
    bytes[] memory chunks,
    bytes calldata /*skippedL1MessageBitmap*/
  ) external {
    // check whether the batch is empty
    uint256 _chunksLength = chunks.length;
    require(_chunksLength > 0, "batch is empty");

    (, bytes32 _parentBatchHash) = _loadBatchHeader(_parentBatchHeader);

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
    uint256 batchPtr;
    assembly {
      let dataLen := mul(_chunksLength, 0x20)
      _dataHash := keccak256(sub(dataPtr, dataLen), dataLen)
      batchPtr := mload(0x40) // reset batchPtr
    }

    BatchHeaderV0Codec.storeVersion(batchPtr, 0);
    BatchHeaderV0Codec.storeBatchIndex(batchPtr, 1);
    BatchHeaderV0Codec.storeDataHash(batchPtr, _dataHash);
    BatchHeaderV0Codec.storeParentBatchHash(batchPtr, _parentBatchHash);
    bytes32 _batchHash = BatchHeaderV0Codec.computeBatchHash(batchPtr, 89);
    committedBatchHash = _batchHash;
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

    bytes32 _batchHash = committedBatchHash;
    emit FinalizeBatch(batchIndex, _batchHash, postStateRoot, withdrawRoot);
  }

  function _loadBatchHeader(bytes calldata _batchHeader) internal pure returns (uint256 memPtr, bytes32 _batchHash) {
    uint256 _length;
    (memPtr, _length) = BatchHeaderV0Codec.loadAndValidate(_batchHeader);
    _batchHash = BatchHeaderV0Codec.computeBatchHash(memPtr, _length);
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
