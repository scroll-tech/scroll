// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import {IL1MessageQueue} from "./IL1MessageQueue.sol";
import {IScrollChain} from "./IScrollChain.sol";
import {BatchHeaderV0Codec} from "../../libraries/codec/BatchHeaderV0Codec.sol";
import {ChunkCodec} from "../../libraries/codec/ChunkCodec.sol";
import {IRollupVerifier} from "../../libraries/verifier/IRollupVerifier.sol";

// solhint-disable no-inline-assembly
// solhint-disable reason-string

/// @title ScrollChain
/// @notice This contract maintains data for the Scroll rollup.
contract ScrollChain is OwnableUpgradeable, IScrollChain {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates the status of sequencer.
    /// @param account The address of account updated.
    /// @param status The status of the account updated.
    event UpdateSequencer(address indexed account, bool status);

    /// @notice Emitted when owner updates the status of prover.
    /// @param account The address of account updated.
    /// @param status The status of the account updated.
    event UpdateProver(address indexed account, bool status);

    /// @notice Emitted when the address of rollup verifier is updated.
    /// @param oldVerifier The address of old rollup verifier.
    /// @param newVerifier The address of new rollup verifier.
    event UpdateVerifier(address oldVerifier, address newVerifier);

    /// @notice Emitted when the value of `maxNumL2TxInChunk` is updated.
    /// @param oldMaxNumL2TxInChunk The old value of `maxNumL2TxInChunk`.
    /// @param newMaxNumL2TxInChunk The new value of `maxNumL2TxInChunk`.
    event UpdateMaxNumL2TxInChunk(uint256 oldMaxNumL2TxInChunk, uint256 newMaxNumL2TxInChunk);

    /*************
     * Constants *
     *************/

    /// @notice The chain id of the corresponding layer 2 chain.
    uint64 public immutable layer2ChainId;

    /*************
     * Variables *
     *************/

    /// @notice The maximum number of transactions allowed in each chunk.
    uint256 public maxNumL2TxInChunk;

    /// @notice The address of L1MessageQueue.
    address public messageQueue;

    /// @notice The address of RollupVerifier.
    address public verifier;

    /// @notice Whether an account is a sequencer.
    mapping(address => bool) public isSequencer;

    /// @notice Whether an account is a prover.
    mapping(address => bool) public isProver;

    /// @notice The latest finalized batch index.
    uint256 public lastFinalizedBatchIndex;

    /// @inheritdoc IScrollChain
    mapping(uint256 => bytes32) public override committedBatches;

    /// @inheritdoc IScrollChain
    mapping(uint256 => bytes32) public override finalizedStateRoots;

    /// @inheritdoc IScrollChain
    mapping(uint256 => bytes32) public override withdrawRoots;

    /**********************
     * Function Modifiers *
     **********************/

    modifier OnlySequencer() {
        // @note In the decentralized mode, it should be only called by a list of validator.
        require(isSequencer[msg.sender], "caller not sequencer");
        _;
    }

    modifier OnlyProver() {
        require(isProver[msg.sender], "caller not prover");
        _;
    }

    /***************
     * Constructor *
     ***************/

    constructor(uint64 _chainId) {
        layer2ChainId = _chainId;
    }

    function initialize(
        address _messageQueue,
        address _verifier,
        uint256 _maxNumL2TxInChunk
    ) public initializer {
        OwnableUpgradeable.__Ownable_init();

        messageQueue = _messageQueue;
        verifier = _verifier;
        maxNumL2TxInChunk = _maxNumL2TxInChunk;

        emit UpdateVerifier(address(0), _verifier);
        emit UpdateMaxNumL2TxInChunk(0, _maxNumL2TxInChunk);
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IScrollChain
    function isBatchFinalized(uint256 _batchIndex) external view override returns (bool) {
        return _batchIndex <= lastFinalizedBatchIndex;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Import layer 2 genesis block
    function importGenesisBatch(bytes calldata _batchHeader, bytes32 _stateRoot) external {
        // check genesis batch header length
        require(_stateRoot != bytes32(0), "zero state root");

        // check whether the genesis batch is imported
        require(finalizedStateRoots[0] == bytes32(0), "Genesis batch imported");

        (uint256 memPtr, bytes32 _batchHash) = _loadBatchHeader(_batchHeader);

        // check all fields except `dataHash` and `lastBlockHash` are zero
        unchecked {
            uint256 sum = BatchHeaderV0Codec.version(memPtr) +
                BatchHeaderV0Codec.batchIndex(memPtr) +
                BatchHeaderV0Codec.l1MessagePopped(memPtr) +
                BatchHeaderV0Codec.totalL1MessagePopped(memPtr);
            require(sum == 0, "not all fields are zero");
        }
        require(BatchHeaderV0Codec.dataHash(memPtr) != bytes32(0), "zero data hash");
        require(BatchHeaderV0Codec.parentBatchHash(memPtr) == bytes32(0), "nonzero parent batch hash");

        committedBatches[0] = _batchHash;
        finalizedStateRoots[0] = _stateRoot;

        emit CommitBatch(_batchHash);
        emit FinalizeBatch(_batchHash, _stateRoot, bytes32(0));
    }

    /// @inheritdoc IScrollChain
    function commitBatch(
        uint8 _version,
        bytes calldata _parentBatchHeader,
        bytes[] memory _chunks,
        bytes calldata _skippedL1MessageBitmap
    ) external override OnlySequencer {
        require(_version == 0, "invalid version");

        // check whether the batch is empty
        uint256 _chunksLength = _chunks.length;
        require(_chunksLength > 0, "batch is empty");

        // The overall memory layout in this function is organized as follows
        // +---------------------+-------------------+------------------+
        // | parent batch header | chunk data hashes | new batch header |
        // +---------------------+-------------------+------------------+
        // ^                     ^                   ^
        // batchPtr              dataPtr             newBatchPtr (re-use var batchPtr)
        //
        // 1. We copy the parent batch header from calldata to memory starting at batchPtr
        // 2. We store `_chunksLength` number of Keccak hashes starting at `dataPtr`. Each Keccak
        //    hash corresponds to the data hash of a chunk. So we reserve the memory region from
        //    `dataPtr` to `dataPtr + _chunkLength * 32` for the chunk data hashes.
        // 3. The memory starting at `newBatchPtr` is used to store the new batch header and compute
        //    the batch hash.

        // the variable `batchPtr` will be reused later for the current batch
        (uint256 batchPtr, bytes32 _parentBatchHash) = _loadBatchHeader(_parentBatchHeader);

        uint256 _batchIndex = BatchHeaderV0Codec.batchIndex(batchPtr);
        uint256 _totalL1MessagesPoppedOverall = BatchHeaderV0Codec.totalL1MessagePopped(batchPtr);
        require(committedBatches[_batchIndex] == _parentBatchHash, "incorrect parent batch hash");
        require(committedBatches[_batchIndex + 1] == 0, "batch already committed");

        // load `dataPtr` and reserve the memory region for chunk data hashes
        uint256 dataPtr;
        assembly {
            dataPtr := mload(0x40)
            mstore(0x40, add(dataPtr, mul(_chunksLength, 32)))
        }

        // compute the data hash for each chunk
        uint256 _totalL1MessagesPoppedInBatch;
        for (uint256 i = 0; i < _chunksLength; i++) {
            uint256 _totalNumL1MessagesInChunk = _commitChunk(
                dataPtr,
                _chunks[i],
                _totalL1MessagesPoppedInBatch,
                _totalL1MessagesPoppedOverall,
                _skippedL1MessageBitmap
            );

            unchecked {
                _totalL1MessagesPoppedInBatch += _totalNumL1MessagesInChunk;
                _totalL1MessagesPoppedOverall += _totalNumL1MessagesInChunk;
                dataPtr += 32;
            }
        }

        // check the length of bitmap
        unchecked {
            require(
                ((_totalL1MessagesPoppedInBatch + 255) / 256) * 32 == _skippedL1MessageBitmap.length,
                "wrong bitmap length"
            );
        }

        // compute the data hash for current batch
        bytes32 _dataHash;
        assembly {
            let dataLen := mul(_chunksLength, 0x20)
            _dataHash := keccak256(sub(dataPtr, dataLen), dataLen)

            batchPtr := mload(0x40) // reset batchPtr
            _batchIndex := add(_batchIndex, 1) // increase batch index
        }

        // store entries, the order matters
        BatchHeaderV0Codec.storeVersion(batchPtr, _version);
        BatchHeaderV0Codec.storeBatchIndex(batchPtr, _batchIndex);
        BatchHeaderV0Codec.storeL1MessagePopped(batchPtr, _totalL1MessagesPoppedInBatch);
        BatchHeaderV0Codec.storeTotalL1MessagePopped(batchPtr, _totalL1MessagesPoppedOverall);
        BatchHeaderV0Codec.storeDataHash(batchPtr, _dataHash);
        BatchHeaderV0Codec.storeParentBatchHash(batchPtr, _parentBatchHash);
        BatchHeaderV0Codec.storeSkippedBitmap(batchPtr, _skippedL1MessageBitmap);

        // compute batch hash
        bytes32 _batchHash = BatchHeaderV0Codec.computeBatchHash(batchPtr, 89 + _skippedL1MessageBitmap.length);

        committedBatches[_batchIndex] = _batchHash;
        emit CommitBatch(_batchHash);
    }

    /// @inheritdoc IScrollChain
    /// @dev If the owner want to revert a sequence of batches by sending multiple transactions,
    ///      make sure to revert recent batches first.
    function revertBatch(bytes calldata _batchHeader, uint256 _count) external onlyOwner {
        require(_count > 0, "count must be nonzero");

        (uint256 memPtr, bytes32 _batchHash) = _loadBatchHeader(_batchHeader);

        // check batch hash
        uint256 _batchIndex = BatchHeaderV0Codec.batchIndex(memPtr);
        require(committedBatches[_batchIndex] == _batchHash, "incorrect batch hash");
        // make sure no gap is left when reverting from the ending to the beginning.
        require(committedBatches[_batchIndex + _count] == bytes32(0), "reverting must start from the ending");

        // check finalization
        require(_batchIndex > lastFinalizedBatchIndex, "can only revert unfinalized batch");

        while (_count > 0) {
            committedBatches[_batchIndex] = bytes32(0);
            unchecked {
                _batchIndex += 1;
                _count -= 1;
            }

            emit RevertBatch(_batchHash);

            _batchHash = committedBatches[_batchIndex];
            if (_batchHash == bytes32(0)) break;
        }
    }

    /// @inheritdoc IScrollChain
    function finalizeBatchWithProof(
        bytes calldata _batchHeader,
        bytes32 _prevStateRoot,
        bytes32 _postStateRoot,
        bytes32 _withdrawRoot,
        bytes calldata _aggrProof
    ) external override OnlyProver {
        require(_prevStateRoot != bytes32(0), "previous state root is zero");
        require(_postStateRoot != bytes32(0), "new state root is zero");

        // compute batch hash and verify
        (uint256 memPtr, bytes32 _batchHash) = _loadBatchHeader(_batchHeader);

        bytes32 _dataHash = BatchHeaderV0Codec.dataHash(memPtr);
        uint256 _batchIndex = BatchHeaderV0Codec.batchIndex(memPtr);
        require(committedBatches[_batchIndex] == _batchHash, "incorrect batch hash");

        // verify previous state root.
        require(finalizedStateRoots[_batchIndex - 1] == _prevStateRoot, "incorrect previous state root");

        // avoid duplicated verification
        require(finalizedStateRoots[_batchIndex] == bytes32(0), "batch already verified");

        // compute public input hash
        bytes32 _publicInputHash = keccak256(
            abi.encodePacked(layer2ChainId, _prevStateRoot, _postStateRoot, _withdrawRoot, _dataHash)
        );

        // verify batch
        IRollupVerifier(verifier).verifyAggregateProof(_batchIndex, _aggrProof, _publicInputHash);

        // check and update lastFinalizedBatchIndex
        unchecked {
            require(lastFinalizedBatchIndex + 1 == _batchIndex, "incorrect batch index");
            lastFinalizedBatchIndex = _batchIndex;
        }

        // record state root and withdraw root
        finalizedStateRoots[_batchIndex] = _postStateRoot;
        withdrawRoots[_batchIndex] = _withdrawRoot;

        // Pop finalized and non-skipped message from L1MessageQueue.
        uint256 _l1MessagePopped = BatchHeaderV0Codec.l1MessagePopped(memPtr);
        if (_l1MessagePopped > 0) {
            IL1MessageQueue _queue = IL1MessageQueue(messageQueue);

            unchecked {
                uint256 _startIndex = BatchHeaderV0Codec.totalL1MessagePopped(memPtr) - _l1MessagePopped;

                for (uint256 i = 0; i < _l1MessagePopped; i += 256) {
                    uint256 _count = 256;
                    if (_l1MessagePopped - i < _count) {
                        _count = _l1MessagePopped - i;
                    }
                    uint256 _skippedBitmap = BatchHeaderV0Codec.skippedBitmap(memPtr, i / 256);

                    _queue.popCrossDomainMessage(_startIndex, _count, _skippedBitmap);

                    _startIndex += 256;
                }
            }
        }

        emit FinalizeBatch(_batchHash, _postStateRoot, _withdrawRoot);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the status of sequencer.
    /// @dev This function can only called by contract owner.
    /// @param _account The address of account to update.
    /// @param _status The status of the account to update.
    function updateSequencer(address _account, bool _status) external onlyOwner {
        isSequencer[_account] = _status;

        emit UpdateSequencer(_account, _status);
    }

    /// @notice Update the status of prover.
    /// @dev This function can only called by contract owner.
    /// @param _account The address of account to update.
    /// @param _status The status of the account to update.
    function updateProver(address _account, bool _status) external onlyOwner {
        isProver[_account] = _status;

        emit UpdateProver(_account, _status);
    }

    /// @notice Update the address verifier contract.
    /// @param _newVerifier The address of new verifier contract.
    function updateVerifier(address _newVerifier) external onlyOwner {
        address _oldVerifier = verifier;
        verifier = _newVerifier;

        emit UpdateVerifier(_oldVerifier, _newVerifier);
    }

    /// @notice Update the value of `maxNumL2TxInChunk`.
    /// @param _maxNumL2TxInChunk The new value of `maxNumL2TxInChunk`.
    function updateMaxNumL2TxInChunk(uint256 _maxNumL2TxInChunk) external onlyOwner {
        uint256 _oldMaxNumL2TxInChunk = maxNumL2TxInChunk;
        maxNumL2TxInChunk = _maxNumL2TxInChunk;

        emit UpdateMaxNumL2TxInChunk(_oldMaxNumL2TxInChunk, _maxNumL2TxInChunk);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to load batch header from calldata to memory.
    /// @param _batchHeader The batch header in calldata.
    /// @return memPtr The start memory offset of loaded batch header.
    /// @return _batchHash The hash of the loaded batch header.
    function _loadBatchHeader(bytes calldata _batchHeader) internal pure returns (uint256 memPtr, bytes32 _batchHash) {
        // load to memory
        uint256 _length;
        (memPtr, _length) = BatchHeaderV0Codec.loadAndValidate(_batchHeader);

        // compute batch hash
        _batchHash = BatchHeaderV0Codec.computeBatchHash(memPtr, _length);
    }

    /// @dev Internal function to commit a chunk.
    /// @param memPtr The start memory offset to store list of `dataHash`.
    /// @param _chunk The encoded chunk to commit.
    /// @param _totalL1MessagesPoppedInBatch The total number of L1 messages popped in current batch.
    /// @param _totalL1MessagesPoppedOverall The total number of L1 messages popped in all batches including current batch.
    /// @param _skippedL1MessageBitmap The bitmap indicates whether each L1 message is skipped or not.
    /// @return _totalNumL1MessagesInChunk The total number of L1 message popped in current chunk
    function _commitChunk(
        uint256 memPtr,
        bytes memory _chunk,
        uint256 _totalL1MessagesPoppedInBatch,
        uint256 _totalL1MessagesPoppedOverall,
        bytes calldata _skippedL1MessageBitmap
    ) internal view returns (uint256 _totalNumL1MessagesInChunk) {
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
            // concatenate l1 message hashes
            uint256 _numL1MessagesInBlock = ChunkCodec.numL1Messages(blockPtr);
            dataPtr = _loadL1MessageHashes(
                dataPtr,
                _numL1MessagesInBlock,
                _totalL1MessagesPoppedInBatch,
                _totalL1MessagesPoppedOverall,
                _skippedL1MessageBitmap
            );

            // concatenate l2 transaction hashes
            uint256 _numTransactionsInBlock = ChunkCodec.numTransactions(blockPtr);
            for (uint256 j = _numL1MessagesInBlock; j < _numTransactionsInBlock; j++) {
                bytes32 txHash;
                (txHash, l2TxPtr) = ChunkCodec.loadL2TxHash(l2TxPtr);
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
                blockPtr += ChunkCodec.BLOCK_CONTEXT_LENGTH;
            }
        }

        // check the number of L2 transactions in the chunk
        require(
            _totalTransactionsInChunk - _totalNumL1MessagesInChunk <= maxNumL2TxInChunk,
            "too many L2 txs in one chunk"
        );

        // check chunk has correct length
        require(l2TxPtr - chunkPtr == _chunk.length, "incomplete l2 transaction data");

        // compute data hash and store to memory
        assembly {
            let dataHash := keccak256(startDataPtr, sub(dataPtr, startDataPtr))
            mstore(memPtr, dataHash)
        }

        return _totalNumL1MessagesInChunk;
    }

    /// @dev Internal function to load L1 message hashes from the message queue.
    /// @param _ptr The memory offset to store the transaction hash.
    /// @param _numL1Messages The number of L1 messages to load.
    /// @param _totalL1MessagesPoppedInBatch The total number of L1 messages popped in current batch.
    /// @param _totalL1MessagesPoppedOverall The total number of L1 messages popped in all batches including current batch.
    /// @param _skippedL1MessageBitmap The bitmap indicates whether each L1 message is skipped or not.
    /// @return uint256 The new memory offset after loading.
    function _loadL1MessageHashes(
        uint256 _ptr,
        uint256 _numL1Messages,
        uint256 _totalL1MessagesPoppedInBatch,
        uint256 _totalL1MessagesPoppedOverall,
        bytes calldata _skippedL1MessageBitmap
    ) internal view returns (uint256) {
        if (_numL1Messages == 0) return _ptr;
        IL1MessageQueue _messageQueue = IL1MessageQueue(messageQueue);

        unchecked {
            uint256 _bitmap;
            for (uint256 i = 0; i < _numL1Messages; i++) {
                uint256 quo = _totalL1MessagesPoppedInBatch >> 8;
                uint256 rem = _totalL1MessagesPoppedInBatch & 0xff;

                // load bitmap every 256 bits
                if (i == 0 || rem == 0) {
                    assembly {
                        _bitmap := calldataload(add(_skippedL1MessageBitmap.offset, mul(0x20, quo)))
                    }
                }
                if (((_bitmap >> rem) & 1) == 0) {
                    // message not skipped
                    bytes32 _hash = _messageQueue.getCrossDomainMessage(_totalL1MessagesPoppedOverall);
                    assembly {
                        mstore(_ptr, _hash)
                        _ptr := add(_ptr, 0x20)
                    }
                }

                _totalL1MessagesPoppedInBatch += 1;
                _totalL1MessagesPoppedOverall += 1;
            }
        }

        return _ptr;
    }
}
