// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import {IL1MessageQueue} from "./IL1MessageQueue.sol";
import {IScrollChain} from "./IScrollChain.sol";
import {BatchHeaderV0Codec} from "../../libraries/codec/BatchHeaderV0Codec.sol";
import {ChunkCodec} from "../../libraries/codec/ChunkCodec.sol";
import {IRollupVerifier} from "../../libraries/verifier/IRollupVerifier.sol";

// solhint-disable reason-string

/// @title ScrollChain
/// @notice This contract maintains essential data for scroll rollup.
contract ScrollChain is OwnableUpgradeable, IScrollChain {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates the status of sequencer.
    /// @param account The address of account updated.
    /// @param status The status of the account updated.
    event UpdateSequencer(address indexed account, bool status);

    /// @notice Emitted when the address of rollup verifier is updated.
    /// @param oldVerifier The address of old rollup verifier.
    /// @param newVerifier The address of new rollup verifier.
    event UpdateVerifier(address oldVerifier, address newVerifier);

    /*************
     * Constants *
     *************/

    /// @notice The chain id of the corresponding layer 2 chain.
    uint256 public immutable layer2ChainId;

    /// @notice The maximum number of transactions allowed in each chunk.
    uint256 public immutable maxNumL2TxInChunk;

    /*************
     * Variables *
     *************/

    /// @notice The address of L1MessageQueue.
    address public messageQueue;

    /// @notice The address of RollupVerifier.
    address public verifier;

    /// @notice Whether an account is a sequencer.
    mapping(address => bool) public isSequencer;

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

    /***************
     * Constructor *
     ***************/

    constructor(uint256 _chainId, uint256 _maxNumL2TxInChunk) {
        layer2ChainId = _chainId;
        maxNumL2TxInChunk = _maxNumL2TxInChunk;
    }

    function initialize(address _messageQueue, address _verifier) public initializer {
        OwnableUpgradeable.__Ownable_init();

        messageQueue = _messageQueue;
        verifier = _verifier;

        emit UpdateVerifier(address(0), _verifier);
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
    /// @dev Although `_withdrawRoot` is always zero, we add this parameterfor the convenience of unit testing.
    function importGenesisBatch(
        bytes calldata _batchHeader,
        bytes32 _stateRoot,
        bytes32 _withdrawRoot
    ) external {
        // check genesis batch header length
        require(_stateRoot != bytes32(0), "zero state root");

        // check whether the genesis batch is imported
        require(finalizedStateRoots[0] == bytes32(0), "Genesis batch imported");

        (uint256 memPtr, bytes32 _batchHash) = _loadBatchHeader(_batchHeader);

        // check all fields except `dataHash` and `lastBlockHash` are zero
        unchecked {
            uint256 sums = BatchHeaderV0Codec.version(memPtr) +
                BatchHeaderV0Codec.batchIndex(memPtr) +
                BatchHeaderV0Codec.l1MessagePopped(memPtr) +
                BatchHeaderV0Codec.totalL1MessagePopped(memPtr);
            require(sums == 0, "not all fields are zero");
        }
        require(BatchHeaderV0Codec.dataHash(memPtr) != bytes32(0), "zero data hash");
        require(BatchHeaderV0Codec.parentBatchHash(memPtr) == bytes32(0), "nonzero parent batch hash");

        committedBatches[0] = _batchHash;
        finalizedStateRoots[0] = _stateRoot;
        withdrawRoots[0] = _withdrawRoot;

        emit CommitBatch(_batchHash);

        emit FinalizeBatch(_batchHash, _stateRoot, _withdrawRoot);
    }

    /// @inheritdoc IScrollChain
    function commitBatch(
        uint8 _version,
        bytes calldata _parentBatchHeader,
        bytes[] memory _chunks,
        bytes calldata _skippedL1MessageBitmap
    ) external override OnlySequencer {
        // check whether the batch is empty
        uint256 _chunksLength = _chunks.length;
        require(_chunksLength > 0, "batch is empty");

        // The variable `memPtr` will be reused for other purposes later.
        (uint256 memPtr, bytes32 _parentBatchHash) = _loadBatchHeader(_parentBatchHeader);

        uint256 _batchIndex = BatchHeaderV0Codec.batchIndex(memPtr);
        uint256 _totalL1MessagesPoppedOverall = BatchHeaderV0Codec.totalL1MessagePopped(memPtr);
        require(committedBatches[_batchIndex] == _parentBatchHash, "incorrect parent batch bash");

        // compute data hash for each chunk
        // We will store `_chunksLength` number of keccak hash digests starting at `memPtr`,
        // each of which is the data hash of the corresponding chunk. So we reserve the memory
        // region from `memPtr` to `memPtr + _chunkLength * 32` for chunk data hashes.
        assembly {
            memPtr := mload(0x40)
            mstore(0x40, add(memPtr, mul(_chunksLength, 32)))
        }

        uint256 _totalL1MessagesPoppedInBatch;
        for (uint256 i = 0; i < _chunksLength; i++) {
            uint256 _totalNumL1MessagesInChunk = _commitChunk(
                memPtr,
                _chunks[i],
                _totalL1MessagesPoppedInBatch,
                _totalL1MessagesPoppedOverall,
                _skippedL1MessageBitmap
            );

            // load `numL1Messages` from memory
            unchecked {
                _totalL1MessagesPoppedInBatch += _totalNumL1MessagesInChunk;
                _totalL1MessagesPoppedOverall += _totalNumL1MessagesInChunk;
                memPtr += 32;
            }
        }

        unchecked {
            require(
                ((_totalL1MessagesPoppedInBatch + 255) / 256) * 32 == _skippedL1MessageBitmap.length,
                "wrong bitmap length"
            );
        }

        // compute current batch hash
        bytes32 _dataHash;
        assembly {
            memPtr := sub(memPtr, mul(_chunksLength, 0x20))
            _dataHash := keccak256(memPtr, mul(_chunksLength, 0x20))
            memPtr := mload(0x20)
            _batchIndex := add(_batchIndex, 1)
        }

        // store entries
        BatchHeaderV0Codec.storeVersion(memPtr, _version);
        BatchHeaderV0Codec.storeBatchIndex(memPtr, _batchIndex);
        BatchHeaderV0Codec.storeL1MessagePopped(memPtr, _totalL1MessagesPoppedInBatch);
        BatchHeaderV0Codec.storeTotalL1MessagePopped(memPtr, _totalL1MessagesPoppedOverall);
        BatchHeaderV0Codec.storeDataHash(memPtr, _dataHash);
        BatchHeaderV0Codec.storeParentBatchHash(memPtr, _parentBatchHash);
        BatchHeaderV0Codec.storeBitMap(memPtr, _skippedL1MessageBitmap);

        // compute batch hash
        bytes32 _batchHash = BatchHeaderV0Codec.computeBatchHash(memPtr, 89 + _skippedL1MessageBitmap.length);

        committedBatches[_batchIndex] = _batchHash;
        emit CommitBatch(_batchHash);
    }

    /// @inheritdoc IScrollChain
    function revertBatch(bytes calldata _batchHeader) external OnlySequencer {
        (uint256 memPtr, bytes32 _batchHash) = _loadBatchHeader(_batchHeader);

        // check batch hash
        uint256 _batchIndex = BatchHeaderV0Codec.batchIndex(memPtr);
        require(committedBatches[_batchIndex] == _batchHash, "incorrect batch bash");

        // check finalization
        require(_batchIndex > lastFinalizedBatchIndex, "can only revert unfinalized batch");

        committedBatches[_batchIndex] = bytes32(0);

        emit RevertBatch(_batchHash);
    }

    /// @inheritdoc IScrollChain
    function finalizeBatchWithProof(
        bytes calldata _batchHeader,
        bytes32 _prevStateRoot,
        bytes32 _newStateRoot,
        bytes32 _withdrawRoot,
        bytes calldata _aggrProof
    ) external override OnlySequencer {
        // compute batch hash and verify
        (uint256 memPtr, bytes32 _batchHash) = _loadBatchHeader(_batchHeader);

        bytes32 _dataHash = BatchHeaderV0Codec.dataHash(memPtr);
        uint256 _batchIndex = BatchHeaderV0Codec.batchIndex(memPtr);
        require(committedBatches[_batchIndex] == _batchHash, "incorrect batch bash");

        // verify previous state root.
        require(finalizedStateRoots[_batchIndex - 1] == _prevStateRoot, "incorrect previous state root");

        // avoid duplicated verification
        require(finalizedStateRoots[_batchIndex] == bytes32(0), "batch already verified");

        // compute public input hash
        bytes32 _publicInputHash = keccak256(abi.encode(_prevStateRoot, _newStateRoot, _withdrawRoot, _dataHash));

        // verify batch
        IRollupVerifier(verifier).verifyAggregateProof(_aggrProof, _publicInputHash);

        // check and update lastFinalizedBatchIndex
        unchecked {
            uint256 _lastFinalizedBatchIndex = lastFinalizedBatchIndex;
            require(_lastFinalizedBatchIndex + 1 == _batchIndex, "incorrect batch index");
            lastFinalizedBatchIndex = _batchIndex;
        }

        // record state root and withdraw root
        finalizedStateRoots[_batchIndex] = _newStateRoot;
        withdrawRoots[_batchIndex] = _withdrawRoot;

        // @todo pop finalized and non-skipped message from L1MessageQueue.

        emit FinalizeBatch(_batchHash, _newStateRoot, _withdrawRoot);
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

    /// @notice Update the address verifier contract.
    /// @param _newVerifier The address of new verifier contract.
    function updateVerifier(address _newVerifier) external onlyOwner {
        address _oldVerifier = verifier;
        verifier = _newVerifier;

        emit UpdateVerifier(_oldVerifier, _newVerifier);
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
        uint256 dataPtr;
        assembly {
            dataPtr := mload(0x40)
            chunkPtr := add(_chunk, 0x20) // skip chunkLength
        }

        uint256 _numBlocks = ChunkCodec.validateChunkLength(chunkPtr, _chunk.length);

        // concatenate block contexts
        for (uint256 i = 0; i < _numBlocks; i++) {
            dataPtr = ChunkCodec.copyBlockContext(chunkPtr, dataPtr, i);
        }

        // concatenate tx hashes
        uint256 l2TxPtr = ChunkCodec.l2TxPtr(chunkPtr, _numBlocks);
        unchecked {
            chunkPtr += 1; // skip numBlocks
        }

        // avoid stack too deep on forge coverage
        uint256 _totalTransactionsInChunk;
        while (_numBlocks > 0) {
            // concatenate l1 messages
            uint256 _numL1MessagesInBlock = ChunkCodec.numL1Messages(chunkPtr);
            dataPtr = _loadL1Messages(
                dataPtr,
                _numL1MessagesInBlock,
                _totalL1MessagesPoppedInBatch,
                _totalL1MessagesPoppedOverall,
                _skippedL1MessageBitmap
            );

            // concatenate l2 transactions
            uint256 _numTransactionsInBlock = ChunkCodec.numTransactions(chunkPtr);
            for (uint256 j = _numL1MessagesInBlock; j < _numTransactionsInBlock; j++) {
                bytes32 txHash;
                (txHash, l2TxPtr) = ChunkCodec.loadL2TxHash(l2TxPtr);
                assembly {
                    mstore(dataPtr, txHash)
                    dataPtr := add(dataPtr, 0x20)
                }
            }

            unchecked {
                _totalTransactionsInChunk += _numTransactionsInBlock;
                _totalNumL1MessagesInChunk += _numL1MessagesInBlock;
                _totalL1MessagesPoppedInBatch += _numL1MessagesInBlock;
                _totalL1MessagesPoppedOverall += _numL1MessagesInBlock;

                _numBlocks -= 1;
                chunkPtr += ChunkCodec.BLOCK_CONTEXT_LENGTH;
            }
        }
        require(
            _totalTransactionsInChunk - _totalNumL1MessagesInChunk <= maxNumL2TxInChunk,
            "too many L2 txs in one chunk"
        );

        // check chunk has correct length
        assembly {
            chunkPtr := add(_chunk, 0x20)
        }
        require(l2TxPtr - chunkPtr == _chunk.length, "incomplete l2 transaction data");

        // compute data hash and store to memory
        assembly {
            let startPtr := mload(0x40)
            let dataHash := keccak256(startPtr, sub(dataPtr, startPtr))

            mstore(memPtr, dataHash)
        }

        return _totalNumL1MessagesInChunk;
    }

    /// @dev Internal function to load L1 messages from message queue.
    /// @param _ptr The memory offset to store the transaction hash.
    /// @param _numL1Messages The number of L1 messages to load.
    /// @param _totalL1MessagesPoppedInBatch The total number of L1 messages popped in current batch.
    /// @param _totalL1MessagesPoppedOverall The total number of L1 messages popped in all batches including current batch.
    /// @param _skippedL1MessageBitmap The bitmap indicates whether each L1 message is skipped or not.
    /// @return uint256 The new memory offset after loading.
    function _loadL1Messages(
        uint256 _ptr,
        uint256 _numL1Messages,
        uint256 _totalL1MessagesPoppedInBatch,
        uint256 _totalL1MessagesPoppedOverall,
        bytes calldata _skippedL1MessageBitmap
    ) internal view returns (uint256) {
        if (_numL1Messages == 0) return _ptr;
        IL1MessageQueue _messageQueue = IL1MessageQueue(messageQueue);

        for (uint256 j = 0; j < _numL1Messages; j++) {
            uint256 _skipped;
            assembly {
                // compute the position in bitmap
                let index := add(_totalL1MessagesPoppedInBatch, j)
                let r := and(index, 0xff)
                index := shr(8, index)

                // load the corresponding bit
                _skipped := calldataload(add(_skippedL1MessageBitmap.offset, add(0x20, mul(index, 0x20))))
                _skipped := and(1, shr(r, _skipped))
            }

            if (_skipped == 0) {
                // message not skipped
                bytes32 _hash = _messageQueue.getCrossDomainMessage(_totalL1MessagesPoppedOverall + j);
                assembly {
                    mstore(_ptr, _hash)
                    _ptr := add(_ptr, 0x20)
                }
            }
        }
        return _ptr;
    }
}
