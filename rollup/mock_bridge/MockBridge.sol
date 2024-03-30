// SPDX-License-Identifier: UNLICENSED
pragma solidity =0.8.24;

import {BatchHeaderV0Codec} from "../../../contracts/src/libraries/codec/BatchHeaderV0Codec.sol";
import {BatchHeaderV1Codec} from "../../../contracts/src/libraries/codec/BatchHeaderV1Codec.sol";
import {ChunkCodecV0} from "../../../contracts/src/libraries/codec/ChunkCodecV0.sol";
import {ChunkCodecV1} from "../../../contracts/src/libraries/codec/ChunkCodecV1.sol";

contract MockBridge {
    /// @dev Thrown when committing a committed batch.
    error ErrorBatchIsAlreadyCommitted();

    /// @dev Thrown when finalizing a verified batch.
    error ErrorBatchIsAlreadyVerified();

    /// @dev Thrown when committing empty batch (batch without chunks)
    error ErrorBatchIsEmpty();

    /// @dev Thrown when call precompile failed.
    error ErrorCallPointEvaluationPrecompileFailed();

    /// @dev Thrown when the transaction has multiple blobs.
    error ErrorFoundMultipleBlob();

    /// @dev Thrown when some fields are not zero in genesis batch.
    error ErrorGenesisBatchHasNonZeroField();

    /// @dev Thrown when importing genesis batch twice.
    error ErrorGenesisBatchImported();

    /// @dev Thrown when data hash in genesis batch is zero.
    error ErrorGenesisDataHashIsZero();

    /// @dev Thrown when the parent batch hash in genesis batch is zero.
    error ErrorGenesisParentBatchHashIsNonZero();

    /// @dev Thrown when the l2 transaction is incomplete.
    error ErrorIncompleteL2TransactionData();

    /// @dev Thrown when the batch hash is incorrect.
    error ErrorIncorrectBatchHash();

    /// @dev Thrown when the batch index is incorrect.
    error ErrorIncorrectBatchIndex();

    /// @dev Thrown when the previous state root doesn't match stored one.
    error ErrorIncorrectPreviousStateRoot();

    /// @dev Thrown when the batch header version is invalid.
    error ErrorInvalidBatchHeaderVersion();

    /// @dev Thrown when no blob found in the transaction.
    error ErrorNoBlobFound();

    /// @dev Thrown when the number of transactions is less than number of L1 message in one block.
    error ErrorNumTxsLessThanNumL1Msgs();

    /// @dev Thrown when the given previous state is zero.
    error ErrorPreviousStateRootIsZero();

    /// @dev Thrown when the given state root is zero.
    error ErrorStateRootIsZero();

    /// @dev Thrown when a chunk contains too many transactions.
    error ErrorTooManyTxsInOneChunk();

    /// @dev Thrown when the precompile output is incorrect.
    error ErrorUnexpectedPointEvaluationPrecompileOutput();

    event CommitBatch(uint256 indexed batchIndex, bytes32 indexed batchHash);
    event FinalizeBatch(uint256 indexed batchIndex, bytes32 indexed batchHash, bytes32 stateRoot, bytes32 withdrawRoot);

    struct L2MessageProof {
        uint256 batchIndex;
        bytes merkleProof;
    }

    /// @dev Address of the point evaluation precompile used for EIP-4844 blob verification.
    address constant POINT_EVALUATION_PRECOMPILE_ADDR = address(0x0A);

    /// @dev BLS Modulus value defined in EIP-4844 and the magic value returned from a successful call to the
    /// point evaluation precompile
    uint256 constant BLS_MODULUS = 52435875175126190479447740508185965837690552500527637822603658699938581184513;

    uint256 public l2BaseFee;
    uint256 public lastFinalizedBatchIndex;
    mapping(uint256 => bytes32) public committedBatches;
    mapping(uint256 => bytes32) public finalizedStateRoots;
    mapping(uint256 => bytes32) public withdrawRoots;

    function setL2BaseFee(uint256 _newL2BaseFee) external {
      l2BaseFee = _newL2BaseFee;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Import layer 2 genesis block
    function importGenesisBatch(bytes calldata _batchHeader, bytes32 _stateRoot) external {
        // check genesis batch header length
        if (_stateRoot == bytes32(0)) revert ErrorStateRootIsZero();

        // check whether the genesis batch is imported
        if (finalizedStateRoots[0] != bytes32(0)) revert ErrorGenesisBatchImported();

        (uint256 memPtr, bytes32 _batchHash, , ) = _loadBatchHeader(_batchHeader);

        // check all fields except `dataHash` and `lastBlockHash` are zero
        unchecked {
            uint256 sum = BatchHeaderV0Codec.getVersion(memPtr) +
                BatchHeaderV0Codec.getBatchIndex(memPtr) +
                BatchHeaderV0Codec.getL1MessagePopped(memPtr) +
                BatchHeaderV0Codec.getTotalL1MessagePopped(memPtr);
            if (sum != 0) revert ErrorGenesisBatchHasNonZeroField();
        }
        if (BatchHeaderV0Codec.getDataHash(memPtr) == bytes32(0)) revert ErrorGenesisDataHashIsZero();
        if (BatchHeaderV0Codec.getParentBatchHash(memPtr) != bytes32(0)) revert ErrorGenesisParentBatchHashIsNonZero();

        committedBatches[0] = _batchHash;
        finalizedStateRoots[0] = _stateRoot;

        emit CommitBatch(0, _batchHash);
        emit FinalizeBatch(0, _batchHash, _stateRoot, bytes32(0));
    }

    function commitBatch(
        uint8 _version,
        bytes calldata _parentBatchHeader,
        bytes[] memory _chunks,
        bytes calldata
    ) external {
        // check whether the batch is empty
        if (_chunks.length == 0) revert ErrorBatchIsEmpty();

        (, bytes32 _parentBatchHash, uint256 _batchIndex, uint256 _totalL1MessagesPoppedOverall) = _loadBatchHeader(
            _parentBatchHeader
        );
        unchecked {
            _batchIndex += 1;
        }
        if (committedBatches[_batchIndex] != 0) revert ErrorBatchIsAlreadyCommitted();

        bytes32 _batchHash;
        uint256 batchPtr;
        bytes32 _dataHash;
        uint256 _totalL1MessagesPoppedInBatch;
        if (_version == 0) {
            (_dataHash, _totalL1MessagesPoppedInBatch) = _commitChunksV0(
                _totalL1MessagesPoppedOverall,
                _chunks
            );
            assembly {
                batchPtr := mload(0x40)
                _totalL1MessagesPoppedOverall := add(_totalL1MessagesPoppedOverall, _totalL1MessagesPoppedInBatch)
            }
            // store entries, the order matters
            BatchHeaderV0Codec.storeVersion(batchPtr, 0);
            BatchHeaderV0Codec.storeBatchIndex(batchPtr, _batchIndex);
            BatchHeaderV0Codec.storeL1MessagePopped(batchPtr, _totalL1MessagesPoppedInBatch);
            BatchHeaderV0Codec.storeTotalL1MessagePopped(batchPtr, _totalL1MessagesPoppedOverall);
            BatchHeaderV0Codec.storeDataHash(batchPtr, _dataHash);
            BatchHeaderV0Codec.storeParentBatchHash(batchPtr, _parentBatchHash);
            // compute batch hash
            _batchHash = BatchHeaderV0Codec.computeBatchHash(
                batchPtr,
                BatchHeaderV0Codec.BATCH_HEADER_FIXED_LENGTH
            );
        } else if (_version == 1) {
            bytes32 blobVersionedHash;
            (blobVersionedHash, _dataHash, _totalL1MessagesPoppedInBatch) = _commitChunksV1(
                _totalL1MessagesPoppedOverall,
                _chunks
            );
            assembly {
                batchPtr := mload(0x40)
                _totalL1MessagesPoppedOverall := add(_totalL1MessagesPoppedOverall, _totalL1MessagesPoppedInBatch)
            }
            // store entries, the order matters
            BatchHeaderV1Codec.storeVersion(batchPtr, 1);
            BatchHeaderV1Codec.storeBatchIndex(batchPtr, _batchIndex);
            BatchHeaderV1Codec.storeL1MessagePopped(batchPtr, _totalL1MessagesPoppedInBatch);
            BatchHeaderV1Codec.storeTotalL1MessagePopped(batchPtr, _totalL1MessagesPoppedOverall);
            BatchHeaderV1Codec.storeDataHash(batchPtr, _dataHash);
            BatchHeaderV1Codec.storeBlobVersionedHash(batchPtr, blobVersionedHash);
            BatchHeaderV1Codec.storeParentBatchHash(batchPtr, _parentBatchHash);
            // compute batch hash
            _batchHash = BatchHeaderV1Codec.computeBatchHash(
                batchPtr,
                BatchHeaderV1Codec.BATCH_HEADER_FIXED_LENGTH
            );
        } else {
            revert ErrorInvalidBatchHeaderVersion();
        }

        committedBatches[_batchIndex] = _batchHash;
        emit CommitBatch(_batchIndex, _batchHash);
    }

    /// @dev We keep this function to upgrade to 4844 more smoothly.
    function finalizeBatchWithProof(
        bytes calldata _batchHeader,
        bytes32 _prevStateRoot,
        bytes32 _postStateRoot,
        bytes32 _withdrawRoot,
        bytes calldata
    ) external {
        if (_prevStateRoot == bytes32(0)) revert ErrorPreviousStateRootIsZero();
        if (_postStateRoot == bytes32(0)) revert ErrorStateRootIsZero();

        // compute batch hash and verify
        (, bytes32 _batchHash, uint256 _batchIndex, ) = _loadBatchHeader(_batchHeader);

        // verify previous state root.
        if (finalizedStateRoots[_batchIndex - 1] != _prevStateRoot) revert ErrorIncorrectPreviousStateRoot();

        // avoid duplicated verification
        if (finalizedStateRoots[_batchIndex] != bytes32(0)) revert ErrorBatchIsAlreadyVerified();

        // check and update lastFinalizedBatchIndex
        unchecked {
            if (lastFinalizedBatchIndex + 1 != _batchIndex) revert ErrorIncorrectBatchIndex();
            lastFinalizedBatchIndex = _batchIndex;
        }

        // record state root and withdraw root
        finalizedStateRoots[_batchIndex] = _postStateRoot;
        withdrawRoots[_batchIndex] = _withdrawRoot;

        emit FinalizeBatch(_batchIndex, _batchHash, _postStateRoot, _withdrawRoot);
    }

    /// @dev Memory layout of `_blobDataProof`:
    /// ```text
    /// | z       | y       | kzg_commitment | kzg_proof |
    /// |---------|---------|----------------|-----------|
    /// | bytes32 | bytes32 | bytes48        | bytes48   |
    /// ```
    function finalizeBatchWithProof4844(
        bytes calldata _batchHeader,
        bytes32 _prevStateRoot,
        bytes32 _postStateRoot,
        bytes32 _withdrawRoot,
        bytes calldata _blobDataProof,
        bytes calldata
    ) external {
        if (_prevStateRoot == bytes32(0)) revert ErrorPreviousStateRootIsZero();
        if (_postStateRoot == bytes32(0)) revert ErrorStateRootIsZero();

        // compute batch hash and verify
        (uint256 memPtr, bytes32 _batchHash, uint256 _batchIndex, ) = _loadBatchHeader(_batchHeader);
        bytes32 _blobVersionedHash = BatchHeaderV1Codec.getBlobVersionedHash(memPtr);

        // Calls the point evaluation precompile and verifies the output
        {
            (bool success, bytes memory data) = POINT_EVALUATION_PRECOMPILE_ADDR.staticcall(
                abi.encodePacked(_blobVersionedHash, _blobDataProof)
            );
            // We verify that the point evaluation precompile call was successful by testing the latter 32 bytes of the
            // response is equal to BLS_MODULUS as defined in https://eips.ethereum.org/EIPS/eip-4844#point-evaluation-precompile
            if (!success) revert ErrorCallPointEvaluationPrecompileFailed();
            (, uint256 result) = abi.decode(data, (uint256, uint256));
            if (result != BLS_MODULUS) revert ErrorUnexpectedPointEvaluationPrecompileOutput();
        }

        // verify previous state root.
        if (finalizedStateRoots[_batchIndex - 1] != _prevStateRoot) revert ErrorIncorrectPreviousStateRoot();

        // avoid duplicated verification
        if (finalizedStateRoots[_batchIndex] != bytes32(0)) revert ErrorBatchIsAlreadyVerified();

        // check and update lastFinalizedBatchIndex
        unchecked {
            if (lastFinalizedBatchIndex + 1 != _batchIndex) revert ErrorIncorrectBatchIndex();
            lastFinalizedBatchIndex = _batchIndex;
        }

        // record state root and withdraw root
        finalizedStateRoots[_batchIndex] = _postStateRoot;
        withdrawRoots[_batchIndex] = _withdrawRoot;

        emit FinalizeBatch(_batchIndex, _batchHash, _postStateRoot, _withdrawRoot);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to commit chunks with version 0
    /// @param _totalL1MessagesPoppedOverall The number of L1 messages popped before the list of chunks.
    /// @param _chunks The list of chunks to commit.
    /// @return _batchDataHash The computed data hash for the list of chunks.
    /// @return _totalL1MessagesPoppedInBatch The total number of L1 messages poped in this batch, including skipped one.
    function _commitChunksV0(
        uint256 _totalL1MessagesPoppedOverall,
        bytes[] memory _chunks
    ) internal pure returns (bytes32 _batchDataHash, uint256 _totalL1MessagesPoppedInBatch) {
        uint256 _chunksLength = _chunks.length;

        // load `batchDataHashPtr` and reserve the memory region for chunk data hashes
        uint256 batchDataHashPtr;
        assembly {
            batchDataHashPtr := mload(0x40)
            mstore(0x40, add(batchDataHashPtr, mul(_chunksLength, 32)))
        }

        // compute the data hash for each chunk
        for (uint256 i = 0; i < _chunksLength; i++) {
            uint256 _totalNumL1MessagesInChunk;
            bytes32 _chunkDataHash;
            (_chunkDataHash, _totalNumL1MessagesInChunk) = _commitChunkV0(
                _chunks[i],
                _totalL1MessagesPoppedInBatch,
                _totalL1MessagesPoppedOverall
            );
            unchecked {
                _totalL1MessagesPoppedInBatch += _totalNumL1MessagesInChunk;
                _totalL1MessagesPoppedOverall += _totalNumL1MessagesInChunk;
            }
            assembly {
                mstore(batchDataHashPtr, _chunkDataHash)
                batchDataHashPtr := add(batchDataHashPtr, 0x20)
            }
        }

        assembly {
            let dataLen := mul(_chunksLength, 0x20)
            _batchDataHash := keccak256(sub(batchDataHashPtr, dataLen), dataLen)
        }
    }

    /// @dev Internal function to commit chunks with version 1
    /// @param _totalL1MessagesPoppedOverall The number of L1 messages popped before the list of chunks.
    /// @param _chunks The list of chunks to commit.
    /// @return _blobVersionedHash The blob versioned hash for the blob carried in this transaction.
    /// @return _batchDataHash The computed data hash for the list of chunks.
    /// @return _totalL1MessagesPoppedInBatch The total number of L1 messages poped in this batch, including skipped one.
    function _commitChunksV1(
        uint256 _totalL1MessagesPoppedOverall,
        bytes[] memory _chunks
    )
        internal
        view
        returns (
            bytes32 _blobVersionedHash,
            bytes32 _batchDataHash,
            uint256 _totalL1MessagesPoppedInBatch
        )
    {
        {
            bytes32 _secondBlob;
            // Get blob's versioned hash
            assembly {
                _blobVersionedHash := blobhash(0)
                _secondBlob := blobhash(1)
            }
            if (_blobVersionedHash == bytes32(0)) revert ErrorNoBlobFound();
            if (_secondBlob != bytes32(0)) revert ErrorFoundMultipleBlob();
        }

        uint256 _chunksLength = _chunks.length;

        // load `batchDataHashPtr` and reserve the memory region for chunk data hashes
        uint256 batchDataHashPtr;
        assembly {
            batchDataHashPtr := mload(0x40)
            mstore(0x40, add(batchDataHashPtr, mul(_chunksLength, 32)))
        }

        // compute the data hash for each chunk
        for (uint256 i = 0; i < _chunksLength; i++) {
            uint256 _totalNumL1MessagesInChunk;
            bytes32 _chunkDataHash;
            (_chunkDataHash, _totalNumL1MessagesInChunk) = _commitChunkV1(
                _chunks[i],
                _totalL1MessagesPoppedInBatch,
                _totalL1MessagesPoppedOverall
            );
            unchecked {
                _totalL1MessagesPoppedInBatch += _totalNumL1MessagesInChunk;
                _totalL1MessagesPoppedOverall += _totalNumL1MessagesInChunk;
            }
            assembly {
                mstore(batchDataHashPtr, _chunkDataHash)
                batchDataHashPtr := add(batchDataHashPtr, 0x20)
            }
        }

        // compute the data hash for current batch
        assembly {
            let dataLen := mul(_chunksLength, 0x20)
            _batchDataHash := keccak256(sub(batchDataHashPtr, dataLen), dataLen)
        }
    }

    /// @dev Internal function to load batch header from calldata to memory.
    /// @param _batchHeader The batch header in calldata.
    /// @return batchPtr The start memory offset of loaded batch header.
    /// @return _batchHash The hash of the loaded batch header.
    /// @return _batchIndex The index of this batch.
    /// @param _totalL1MessagesPoppedOverall The number of L1 messages popped after this batch.
    function _loadBatchHeader(bytes calldata _batchHeader)
        internal
        view
        returns (
            uint256 batchPtr,
            bytes32 _batchHash,
            uint256 _batchIndex,
            uint256 _totalL1MessagesPoppedOverall
        )
    {
        // load version from batch header, it is always the first byte.
        uint256 version;
        assembly {
            version := shr(248, calldataload(_batchHeader.offset))
        }

        // version should be always 0 or 1 in current code
        uint256 _length;
        if (version == 0) {
            (batchPtr, _length) = BatchHeaderV0Codec.loadAndValidate(_batchHeader);
            _batchHash = BatchHeaderV0Codec.computeBatchHash(batchPtr, _length);
            _batchIndex = BatchHeaderV0Codec.getBatchIndex(batchPtr);
        } else if (version == 1) {
            (batchPtr, _length) = BatchHeaderV1Codec.loadAndValidate(_batchHeader);
            _batchHash = BatchHeaderV1Codec.computeBatchHash(batchPtr, _length);
            _batchIndex = BatchHeaderV1Codec.getBatchIndex(batchPtr);
        } else {
            revert ErrorInvalidBatchHeaderVersion();
        }
        // only check when genesis is imported
        if (committedBatches[_batchIndex] != _batchHash && finalizedStateRoots[0] != bytes32(0)) {
            revert ErrorIncorrectBatchHash();
        }
        _totalL1MessagesPoppedOverall = BatchHeaderV0Codec.getTotalL1MessagePopped(batchPtr);
    }

    /// @dev Internal function to commit a chunk with version 0.
    /// @param _chunk The encoded chunk to commit.
    /// @param _totalL1MessagesPoppedInBatch The total number of L1 messages popped in the current batch before this chunk.
    /// @param _totalL1MessagesPoppedOverall The total number of L1 messages popped in all batches including the current batch, before this chunk.
    /// @return _dataHash The computed data hash for this chunk.
    /// @return _totalNumL1MessagesInChunk The total number of L1 message popped in current chunk
    function _commitChunkV0(
        bytes memory _chunk,
        uint256 _totalL1MessagesPoppedInBatch,
        uint256 _totalL1MessagesPoppedOverall
    ) internal pure returns (bytes32 _dataHash, uint256 _totalNumL1MessagesInChunk) {
        uint256 chunkPtr;
        uint256 startDataPtr;
        uint256 dataPtr;

        assembly {
            dataPtr := mload(0x40)
            startDataPtr := dataPtr
            chunkPtr := add(_chunk, 0x20) // skip chunkLength
        }

        uint256 _numBlocks = ChunkCodecV0.validateChunkLength(chunkPtr, _chunk.length);

        // concatenate block contexts, use scope to avoid stack too deep
        {
            uint256 _totalTransactionsInChunk;
            for (uint256 i = 0; i < _numBlocks; i++) {
                dataPtr = ChunkCodecV0.copyBlockContext(chunkPtr, dataPtr, i);
                uint256 blockPtr = chunkPtr + 1 + i * ChunkCodecV0.BLOCK_CONTEXT_LENGTH;
                uint256 _numTransactionsInBlock = ChunkCodecV0.getNumTransactions(blockPtr);
                unchecked {
                    _totalTransactionsInChunk += _numTransactionsInBlock;
                }
            }
            assembly {
                mstore(0x40, add(dataPtr, mul(_totalTransactionsInChunk, 0x20))) // reserve memory for tx hashes
            }
        }

        // concatenate tx hashes
        uint256 l2TxPtr = ChunkCodecV0.getL2TxPtr(chunkPtr, _numBlocks);
        chunkPtr += 1;
        while (_numBlocks > 0) {
            // concatenate l1 message hashes
            uint256 _numL1MessagesInBlock = ChunkCodecV0.getNumL1Messages(chunkPtr);

            // concatenate l2 transaction hashes
            uint256 _numTransactionsInBlock = ChunkCodecV0.getNumTransactions(chunkPtr);
            if (_numTransactionsInBlock < _numL1MessagesInBlock) revert ErrorNumTxsLessThanNumL1Msgs();
            for (uint256 j = _numL1MessagesInBlock; j < _numTransactionsInBlock; j++) {
                bytes32 txHash;
                (txHash, l2TxPtr) = ChunkCodecV0.loadL2TxHash(l2TxPtr);
                assembly {
                    mstore(dataPtr, txHash)
                    dataPtr := add(dataPtr, 0x20)
                }
            }

            unchecked {
                _totalNumL1MessagesInChunk += _numL1MessagesInBlock;
                _totalL1MessagesPoppedInBatch += _numL1MessagesInBlock;
                _totalL1MessagesPoppedOverall += _numL1MessagesInBlock;

                _numBlocks -= 1;
                chunkPtr += ChunkCodecV0.BLOCK_CONTEXT_LENGTH;
            }
        }

        assembly {
            chunkPtr := add(_chunk, 0x20)
        }
        // check chunk has correct length
        if (l2TxPtr - chunkPtr != _chunk.length) revert ErrorIncompleteL2TransactionData();

        // compute data hash and store to memory
        assembly {
            _dataHash := keccak256(startDataPtr, sub(dataPtr, startDataPtr))
        }
    }

    /// @dev Internal function to commit a chunk with version 1.
    /// @param _chunk The encoded chunk to commit.
    /// @param _totalL1MessagesPoppedInBatch The total number of L1 messages popped in current batch.
    /// @param _totalL1MessagesPoppedOverall The total number of L1 messages popped in all batches including current batch.
    /// @return _dataHash The computed data hash for this chunk.
    /// @return _totalNumL1MessagesInChunk The total number of L1 message popped in current chunk
    function _commitChunkV1(
        bytes memory _chunk,
        uint256 _totalL1MessagesPoppedInBatch,
        uint256 _totalL1MessagesPoppedOverall
    ) internal pure returns (bytes32 _dataHash, uint256 _totalNumL1MessagesInChunk) {
        uint256 chunkPtr;
        uint256 startDataPtr;
        uint256 dataPtr;

        assembly {
            dataPtr := mload(0x40)
            startDataPtr := dataPtr
            chunkPtr := add(_chunk, 0x20) // skip chunkLength
        }

        uint256 _numBlocks = ChunkCodecV1.validateChunkLength(chunkPtr, _chunk.length);
        // concatenate block contexts, use scope to avoid stack too deep
        for (uint256 i = 0; i < _numBlocks; i++) {
            dataPtr = ChunkCodecV1.copyBlockContext(chunkPtr, dataPtr, i);
            uint256 blockPtr = chunkPtr + 1 + i * ChunkCodecV1.BLOCK_CONTEXT_LENGTH;
            uint256 _numL1MessagesInBlock = ChunkCodecV1.getNumL1Messages(blockPtr);
            unchecked {
                _totalNumL1MessagesInChunk += _numL1MessagesInBlock;
            }
        }
        assembly {
            mstore(0x40, add(dataPtr, mul(_totalNumL1MessagesInChunk, 0x20))) // reserve memory for l1 message hashes
            chunkPtr := add(chunkPtr, 1)
        }

        // the number of actual transactions in one chunk: non-skipped l1 messages + l2 txs
        uint256 _totalTransactionsInChunk;
        // concatenate tx hashes
        while (_numBlocks > 0) {
            // concatenate l1 message hashes
            uint256 _numL1MessagesInBlock = ChunkCodecV1.getNumL1Messages(chunkPtr);
            uint256 startPtr = dataPtr;
            uint256 _numTransactionsInBlock = ChunkCodecV1.getNumTransactions(chunkPtr);
            if (_numTransactionsInBlock < _numL1MessagesInBlock) revert ErrorNumTxsLessThanNumL1Msgs();
            unchecked {
                _totalTransactionsInChunk += dataPtr - startPtr; // number of non-skipped l1 messages
                _totalTransactionsInChunk += _numTransactionsInBlock - _numL1MessagesInBlock; // number of l2 txs
                _totalL1MessagesPoppedInBatch += _numL1MessagesInBlock;
                _totalL1MessagesPoppedOverall += _numL1MessagesInBlock;

                _numBlocks -= 1;
                chunkPtr += ChunkCodecV1.BLOCK_CONTEXT_LENGTH;
            }
        }

        // compute data hash and store to memory
        assembly {
            _dataHash := keccak256(startDataPtr, sub(dataPtr, startDataPtr))
        }
    }

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

        (bool success,) = POINT_EVALUATION_PRECOMPILE_ADDR.staticcall(pointEvaluationCalldata);

        if (!success) {
            revert("Proof verification failed");
        }
    }
}
