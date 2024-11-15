// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {BatchHeaderV0Codec} from "../../../scroll-contracts/src/libraries/codec/BatchHeaderV0Codec.sol";
import {BatchHeaderV1Codec} from "../../../scroll-contracts/src/libraries/codec/BatchHeaderV1Codec.sol";
import {BatchHeaderV3Codec} from "../../../scroll-contracts/src/libraries/codec/BatchHeaderV3Codec.sol";
import {ChunkCodecV0} from "../../../scroll-contracts/src/libraries/codec/ChunkCodecV0.sol";
import {ChunkCodecV1} from "../../../scroll-contracts/src/libraries/codec/ChunkCodecV1.sol";

contract MockBridge {
    /**********
     * Errors *
     **********/

    /// @dev Thrown when committing a committed batch.
    error ErrorBatchIsAlreadyCommitted();

    /// @dev Thrown when finalizing a verified batch.
    error ErrorBatchIsAlreadyVerified();

    /// @dev Thrown when committing empty batch (batch without chunks)
    error ErrorBatchIsEmpty();

    /// @dev Thrown when call precompile failed.
    error ErrorCallPointEvaluationPrecompileFailed();

    /// @dev Thrown when the transaction has multiple blobs.
    error ErrorFoundMultipleBlobs();

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

    /// @dev Thrown when the batch version is incorrect.
    error ErrorIncorrectBatchVersion();

    /// @dev Thrown when no blob found in the transaction.
    error ErrorNoBlobFound();

    /// @dev Thrown when the number of transactions is less than number of L1 message in one block.
    error ErrorNumTxsLessThanNumL1Msgs();

    /// @dev Thrown when the given state root is zero.
    error ErrorStateRootIsZero();

    /// @dev Thrown when a chunk contains too many transactions.
    error ErrorTooManyTxsInOneChunk();

    /// @dev Thrown when the precompile output is incorrect.
    error ErrorUnexpectedPointEvaluationPrecompileOutput();

    event CommitBatch(uint256 indexed batchIndex, bytes32 indexed batchHash);
    event FinalizeBatch(uint256 indexed batchIndex, bytes32 indexed batchHash, bytes32 stateRoot, bytes32 withdrawRoot);

    /*************
     * Constants *
     *************/

    /// @dev Address of the point evaluation precompile used for EIP-4844 blob verification.
    address internal constant POINT_EVALUATION_PRECOMPILE_ADDR = address(0x0A);

    /// @dev BLS Modulus value defined in EIP-4844 and the magic value returned from a successful call to the
    /// point evaluation precompile
    uint256 internal constant BLS_MODULUS =
        52435875175126190479447740508185965837690552500527637822603658699938581184513;

    /// @notice The chain id of the corresponding layer 2 chain.
    uint64 public immutable layer2ChainId;

    /*************
     * Variables *
     *************/

    /// @notice The maximum number of transactions allowed in each chunk.
    uint256 public maxNumTxInChunk;

    uint256 public l1BaseFee;
    uint256 public l1BlobBaseFee;
    uint256 public l2BaseFee;
    uint256 public lastFinalizedBatchIndex;

    mapping(uint256 => bytes32) public committedBatches;

    mapping(uint256 => bytes32) public finalizedStateRoots;

    mapping(uint256 => bytes32) public withdrawRoots;

    function setL1BaseFeeAndBlobBaseFee(uint256 _l1BaseFee, uint256 _l1BlobBaseFee) external {
        l1BaseFee = _l1BaseFee;
        l1BlobBaseFee = _l1BlobBaseFee;
    }

    function setL2BaseFee(uint256 _l2BaseFee) external {
        l2BaseFee = _l2BaseFee;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Import layer 2 genesis block
    /// @param _batchHeader The header of the genesis batch.
    /// @param _stateRoot The state root of the genesis block.
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
        (bytes32 _parentBatchHash, uint256 _batchIndex, uint256 _totalL1MessagesPoppedOverall) = _beforeCommitBatch(
            _parentBatchHeader,
            _chunks
        );

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
        } else if (_version <= 2) {
            // versions 1 and 2 both use ChunkCodecV1 and BatchHeaderV1Codec,
            // but they use different blob encoding and different verifiers.
            (_dataHash, _totalL1MessagesPoppedInBatch) = _commitChunksV1(
                _totalL1MessagesPoppedOverall,
                _chunks
            );
            assembly {
                batchPtr := mload(0x40)
                _totalL1MessagesPoppedOverall := add(_totalL1MessagesPoppedOverall, _totalL1MessagesPoppedInBatch)
            }

            // store entries, the order matters
            // Some are using `BatchHeaderV0Codec`, see comments of `BatchHeaderV1Codec`.
            BatchHeaderV0Codec.storeVersion(batchPtr, _version);
            BatchHeaderV0Codec.storeBatchIndex(batchPtr, _batchIndex);
            BatchHeaderV0Codec.storeL1MessagePopped(batchPtr, _totalL1MessagesPoppedInBatch);
            BatchHeaderV0Codec.storeTotalL1MessagePopped(batchPtr, _totalL1MessagesPoppedOverall);
            BatchHeaderV0Codec.storeDataHash(batchPtr, _dataHash);
            BatchHeaderV1Codec.storeBlobVersionedHash(batchPtr, _getBlobVersionedHash());
            BatchHeaderV1Codec.storeParentBatchHash(batchPtr, _parentBatchHash);
            // compute batch hash, V1 and V2 has same code as V0
            _batchHash = BatchHeaderV0Codec.computeBatchHash(
                batchPtr,
                BatchHeaderV1Codec.BATCH_HEADER_FIXED_LENGTH
            );
        } else {
            revert ErrorIncorrectBatchVersion();
        }

        _afterCommitBatch(_batchIndex, _batchHash);
    }

    /// @dev This function will revert unless all V0/V1/V2 batches are finalized. This is because we start to
    /// pop L1 messages in `commitBatchWithBlobProof` but not in `commitBatch`. We also introduce `finalizedQueueIndex`
    /// in `L1MessageQueue`. If one of V0/V1/V2 batches not finalized, `L1MessageQueue.pendingQueueIndex` will not
    /// match `parentBatchHeader.totalL1MessagePopped` and thus revert.
    function commitBatchWithBlobProof(
        uint8 _version,
        bytes calldata _parentBatchHeader,
        bytes[] memory _chunks,
        bytes calldata,
        bytes calldata _blobDataProof
    ) external {
        if (_version <= 2) {
            revert ErrorIncorrectBatchVersion();
        }

        // allocate memory of batch header and store entries if necessary, the order matters
        // @note why store entries if necessary, to avoid stack overflow problem.
        // The codes for `version`, `batchIndex`, `l1MessagePopped`, `totalL1MessagePopped` and `dataHash`
        // are the same as `BatchHeaderV0Codec`.
        // The codes for `blobVersionedHash`, and `parentBatchHash` are the same as `BatchHeaderV1Codec`.
        uint256 batchPtr;
        assembly {
            batchPtr := mload(0x40)
            // This is `BatchHeaderV3Codec.BATCH_HEADER_FIXED_LENGTH`, use `193` here to reduce code
            // complexity. Be careful that the length may changed in future versions.
            mstore(0x40, add(batchPtr, 193))
        }
        BatchHeaderV0Codec.storeVersion(batchPtr, _version);

        (bytes32 _parentBatchHash, uint256 _batchIndex, uint256 _totalL1MessagesPoppedOverall) = _beforeCommitBatch(
            _parentBatchHeader,
            _chunks
        );
        BatchHeaderV0Codec.storeBatchIndex(batchPtr, _batchIndex);

        // versions 2 and 3 both use ChunkCodecV1
        (bytes32 _dataHash, uint256 _totalL1MessagesPoppedInBatch) = _commitChunksV1(
            _totalL1MessagesPoppedOverall,
            _chunks
        );
        unchecked {
            _totalL1MessagesPoppedOverall += _totalL1MessagesPoppedInBatch;
        }

        BatchHeaderV0Codec.storeL1MessagePopped(batchPtr, _totalL1MessagesPoppedInBatch);
        BatchHeaderV0Codec.storeTotalL1MessagePopped(batchPtr, _totalL1MessagesPoppedOverall);
        BatchHeaderV0Codec.storeDataHash(batchPtr, _dataHash);

        // verify blob versioned hash
        bytes32 _blobVersionedHash = _getBlobVersionedHash();
        _checkBlobVersionedHash(_blobVersionedHash, _blobDataProof);
        BatchHeaderV1Codec.storeBlobVersionedHash(batchPtr, _blobVersionedHash);
        BatchHeaderV1Codec.storeParentBatchHash(batchPtr, _parentBatchHash);

        uint256 lastBlockTimestamp;
        {
            bytes memory lastChunk = _chunks[_chunks.length - 1];
            lastBlockTimestamp = ChunkCodecV1.getLastBlockTimestamp(lastChunk);
        }
        BatchHeaderV3Codec.storeLastBlockTimestamp(batchPtr, lastBlockTimestamp);
        BatchHeaderV3Codec.storeBlobDataProof(batchPtr, _blobDataProof);

        // compute batch hash, V3 has same code as V0
        bytes32 _batchHash = BatchHeaderV0Codec.computeBatchHash(
            batchPtr,
            BatchHeaderV3Codec.BATCH_HEADER_FIXED_LENGTH
        );

        _afterCommitBatch(_batchIndex, _batchHash);
    }

    /// @dev We keep this function to upgrade to 4844 more smoothly.
    function finalizeBatchWithProof(
        bytes calldata _batchHeader,
        bytes32, /*_prevStateRoot*/
        bytes32 _postStateRoot,
        bytes32 _withdrawRoot,
        bytes calldata
    ) external {
        (uint256 batchPtr, bytes32 _batchHash, uint256 _batchIndex) = _beforeFinalizeBatch(
            _batchHeader,
            _postStateRoot
        );

        // compute public input hash
        bytes32 _publicInputHash;
        {
            bytes32 _dataHash = BatchHeaderV0Codec.getDataHash(batchPtr);
            bytes32 _prevStateRoot = finalizedStateRoots[_batchIndex - 1];
            _publicInputHash = keccak256(
                abi.encodePacked(layer2ChainId, _prevStateRoot, _postStateRoot, _withdrawRoot, _dataHash)
            );
        }

        // Pop finalized and non-skipped message from L1MessageQueue.
        uint256 _totalL1MessagesPoppedOverall = BatchHeaderV0Codec.getTotalL1MessagePopped(batchPtr);
        _afterFinalizeBatch(_totalL1MessagesPoppedOverall, _batchIndex, _batchHash, _postStateRoot, _withdrawRoot);
    }

    /// @dev Memory layout of `_blobDataProof`:
    /// ```text
    /// | z       | y       | kzg_commitment | kzg_proof |
    /// |---------|---------|----------------|-----------|
    /// | bytes32 | bytes32 | bytes48        | bytes48   |
    /// ```
    function finalizeBatchWithProof4844(
        bytes calldata _batchHeader,
        bytes32,
        bytes32 _postStateRoot,
        bytes32 _withdrawRoot,
        bytes calldata _blobDataProof,
        bytes calldata
    ) external {
        (uint256 batchPtr, bytes32 _batchHash, uint256 _batchIndex) = _beforeFinalizeBatch(
            _batchHeader,
            _postStateRoot
        );

        // compute public input hash
        bytes32 _publicInputHash;
        {
            bytes32 _dataHash = BatchHeaderV0Codec.getDataHash(batchPtr);
            bytes32 _blobVersionedHash = BatchHeaderV1Codec.getBlobVersionedHash(batchPtr);
            bytes32 _prevStateRoot = finalizedStateRoots[_batchIndex - 1];
            // verify blob versioned hash
            _checkBlobVersionedHash(_blobVersionedHash, _blobDataProof);
            _publicInputHash = keccak256(
                abi.encodePacked(
                    layer2ChainId,
                    _prevStateRoot,
                    _postStateRoot,
                    _withdrawRoot,
                    _dataHash,
                    _blobDataProof[0:64],
                    _blobVersionedHash
                )
            );
        }

        // Pop finalized and non-skipped message from L1MessageQueue.
        uint256 _totalL1MessagesPoppedOverall = BatchHeaderV0Codec.getTotalL1MessagePopped(batchPtr);
        _afterFinalizeBatch(_totalL1MessagesPoppedOverall, _batchIndex, _batchHash, _postStateRoot, _withdrawRoot);
    }

    function finalizeBundleWithProof(
        bytes calldata _batchHeader,
        bytes32 _postStateRoot,
        bytes32 _withdrawRoot,
        bytes calldata
    ) external {
        if (_postStateRoot == bytes32(0)) revert ErrorStateRootIsZero();

        // retrieve finalized state root and batch hash from storage
        uint256 _finalizedBatchIndex = lastFinalizedBatchIndex;

        // compute pending batch hash and verify
        (, bytes32 _batchHash, uint256 _batchIndex, ) = _loadBatchHeader(_batchHeader);
        if (_batchIndex <= _finalizedBatchIndex) revert ErrorBatchIsAlreadyVerified();

        // store in state
        // @note we do not store intermediate finalized roots
        lastFinalizedBatchIndex = _batchIndex;
        finalizedStateRoots[_batchIndex] = _postStateRoot;
        withdrawRoots[_batchIndex] = _withdrawRoot;

        emit FinalizeBatch(_batchIndex, _batchHash, _postStateRoot, _withdrawRoot);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to do common checks before actual batch committing.
    /// @param _parentBatchHeader The parent batch header in calldata.
    /// @param _chunks The list of chunks in memory.
    /// @return _parentBatchHash The batch hash of parent batch header.
    /// @return _batchIndex The index of current batch.
    /// @return _totalL1MessagesPoppedOverall The total number of L1 messages popped before current batch.
    function _beforeCommitBatch(bytes calldata _parentBatchHeader, bytes[] memory _chunks)
        private
        view
        returns (
            bytes32 _parentBatchHash,
            uint256 _batchIndex,
            uint256 _totalL1MessagesPoppedOverall
        )
    {
        // check whether the batch is empty
        if (_chunks.length == 0) revert ErrorBatchIsEmpty();
        (, _parentBatchHash, _batchIndex, _totalL1MessagesPoppedOverall) = _loadBatchHeader(_parentBatchHeader);
        unchecked {
            _batchIndex += 1;
        }
        if (committedBatches[_batchIndex] != 0) revert ErrorBatchIsAlreadyCommitted();
    }

    /// @dev Internal function to do common checks after actual batch committing.
    /// @param _batchIndex The index of current batch.
    /// @param _batchHash The hash of current batch.
    function _afterCommitBatch(uint256 _batchIndex, bytes32 _batchHash) private {
        committedBatches[_batchIndex] = _batchHash;
        emit CommitBatch(_batchIndex, _batchHash);
    }

    /// @dev Internal function to do common checks before actual batch finalization.
    /// @param _batchHeader The current batch header in calldata.
    /// @param _postStateRoot The state root after current batch.
    /// @return batchPtr The start memory offset of current batch in memory.
    /// @return _batchHash The hash of current batch.
    /// @return _batchIndex The index of current batch.
    function _beforeFinalizeBatch(bytes calldata _batchHeader, bytes32 _postStateRoot)
        internal
        view
        returns (
            uint256 batchPtr,
            bytes32 _batchHash,
            uint256 _batchIndex
        )
    {
        if (_postStateRoot == bytes32(0)) revert ErrorStateRootIsZero();

        // compute batch hash and verify
        (batchPtr, _batchHash, _batchIndex, ) = _loadBatchHeader(_batchHeader);

        // avoid duplicated verification
        if (finalizedStateRoots[_batchIndex] != bytes32(0)) revert ErrorBatchIsAlreadyVerified();
    }

    /// @dev Internal function to do common checks after actual batch finalization.
    /// @param
    /// @param _batchIndex The index of current batch.
    /// @param _batchHash The hash of current batch.
    /// @param _postStateRoot The state root after current batch.
    /// @param _withdrawRoot The withdraw trie root after current batch.
    function _afterFinalizeBatch(
        uint256,
        uint256 _batchIndex,
        bytes32 _batchHash,
        bytes32 _postStateRoot,
        bytes32 _withdrawRoot
    ) internal {
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

    /// @dev Internal function to check blob versioned hash.
    /// @param _blobVersionedHash The blob versioned hash to check.
    /// @param _blobDataProof The blob data proof used to verify the blob versioned hash.
    function _checkBlobVersionedHash(bytes32 _blobVersionedHash, bytes calldata _blobDataProof) internal view {
        // Calls the point evaluation precompile and verifies the output
        (bool success, bytes memory data) = POINT_EVALUATION_PRECOMPILE_ADDR.staticcall(
            abi.encodePacked(_blobVersionedHash, _blobDataProof)
        );
        // We verify that the point evaluation precompile call was successful by testing the latter 32 bytes of the
        // response is equal to BLS_MODULUS as defined in https://eips.ethereum.org/EIPS/eip-4844#point-evaluation-precompile
        if (!success) revert ErrorCallPointEvaluationPrecompileFailed();
        (, uint256 result) = abi.decode(data, (uint256, uint256));
        if (result != BLS_MODULUS) revert ErrorUnexpectedPointEvaluationPrecompileOutput();
    }

    /// @dev Internal function to get the blob versioned hash.
    /// @return _blobVersionedHash The retrieved blob versioned hash.
    function _getBlobVersionedHash() internal virtual returns (bytes32 _blobVersionedHash) {
        bytes32 _secondBlob;
        // Get blob's versioned hash
        assembly {
            _blobVersionedHash := blobhash(0)
            _secondBlob := blobhash(1)
        }
        if (_blobVersionedHash == bytes32(0)) revert ErrorNoBlobFound();
        if (_secondBlob != bytes32(0)) revert ErrorFoundMultipleBlobs();
    }

    /// @dev Internal function to commit chunks with version 0
    /// @param _totalL1MessagesPoppedOverall The number of L1 messages popped before the list of chunks.
    /// @param _chunks The list of chunks to commit.
    /// @return _batchDataHash The computed data hash for the list of chunks.
    /// @return _totalL1MessagesPoppedInBatch The total number of L1 messages popped in this batch, including skipped one.
    function _commitChunksV0(
        uint256 _totalL1MessagesPoppedOverall,
        bytes[] memory _chunks
    ) internal view returns (bytes32 _batchDataHash, uint256 _totalL1MessagesPoppedInBatch) {
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
    /// @return _batchDataHash The computed data hash for the list of chunks.
    /// @return _totalL1MessagesPoppedInBatch The total number of L1 messages popped in this batch, including skipped one.
    function _commitChunksV1(
        uint256 _totalL1MessagesPoppedOverall,
        bytes[] memory _chunks
    ) internal view returns (bytes32 _batchDataHash, uint256 _totalL1MessagesPoppedInBatch) {
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

        uint256 _length;
        if (version == 0) {
            (batchPtr, _length) = BatchHeaderV0Codec.loadAndValidate(_batchHeader);
        } else if (version <= 2) {
            (batchPtr, _length) = BatchHeaderV1Codec.loadAndValidate(_batchHeader);
        } else if (version >= 3) {
            (batchPtr, _length) = BatchHeaderV3Codec.loadAndValidate(_batchHeader);
        }

        // the code for compute batch hash is the same for V0, V1, V2, V3
        // also the `_batchIndex` and `_totalL1MessagesPoppedOverall`.
        _batchHash = BatchHeaderV0Codec.computeBatchHash(batchPtr, _length);
        _batchIndex = BatchHeaderV0Codec.getBatchIndex(batchPtr);
        _totalL1MessagesPoppedOverall = BatchHeaderV0Codec.getTotalL1MessagePopped(batchPtr);

        // only check when genesis is imported
        if (committedBatches[_batchIndex] != _batchHash && finalizedStateRoots[0] != bytes32(0)) {
            revert ErrorIncorrectBatchHash();
        }
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
    ) internal view returns (bytes32 _dataHash, uint256 _totalNumL1MessagesInChunk) {
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

        // It is used to compute the actual number of transactions in chunk.
        uint256 txHashStartDataPtr = dataPtr;
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

        // check the actual number of transactions in the chunk
        if ((dataPtr - txHashStartDataPtr) / 32 > maxNumTxInChunk) revert ErrorTooManyTxsInOneChunk();

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
    ) internal view returns (bytes32 _dataHash, uint256 _totalNumL1MessagesInChunk) {
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
                _totalTransactionsInChunk += (dataPtr - startPtr) / 32; // number of non-skipped l1 messages
                _totalTransactionsInChunk += _numTransactionsInBlock - _numL1MessagesInBlock; // number of l2 txs
                _totalL1MessagesPoppedInBatch += _numL1MessagesInBlock;
                _totalL1MessagesPoppedOverall += _numL1MessagesInBlock;

                _numBlocks -= 1;
                chunkPtr += ChunkCodecV1.BLOCK_CONTEXT_LENGTH;
            }
        }

        // check the actual number of transactions in the chunk
        if (_totalTransactionsInChunk > maxNumTxInChunk) {
            revert ErrorTooManyTxsInOneChunk();
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
