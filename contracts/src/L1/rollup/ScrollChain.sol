// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {PausableUpgradeable} from "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";

import {IL1MessageQueue} from "./IL1MessageQueue.sol";
import {IScrollChain} from "./IScrollChain.sol";
import {BatchHeaderV0Codec} from "../../libraries/codec/BatchHeaderV0Codec.sol";
import {ChunkCodec} from "../../libraries/codec/ChunkCodec.sol";
import {IRollupVerifier} from "../../libraries/verifier/IRollupVerifier.sol";
import {IZkTrieVerifier} from "../../libraries/verifier/IZkTrieVerifier.sol";

// solhint-disable no-inline-assembly
// solhint-disable reason-string

/// @title ScrollChain
/// @notice This contract maintains data for the Scroll rollup.
contract ScrollChain is OwnableUpgradeable, PausableUpgradeable, IScrollChain {
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

    /// @notice Emitted when the value of `maxNumTxInChunk` is updated.
    /// @param oldMaxNumTxInChunk The old value of `maxNumTxInChunk`.
    /// @param newMaxNumTxInChunk The new value of `maxNumTxInChunk`.
    event UpdateMaxNumTxInChunk(uint256 oldMaxNumTxInChunk, uint256 newMaxNumTxInChunk);

    /// @notice Emitted when the value of `zkTrieVerifier` is updated.
    /// @param oldZkTrieVerifier The old value of `zkTrieVerifier`.
    /// @param newZkTrieVerifier The new value of `zkTrieVerifier`.
    event UpdateZkTrieVerifier(address indexed oldZkTrieVerifier, address indexed newZkTrieVerifier);

    /// @notice Emitted when the value of `maxFinalizationDelay` is updated.
    /// @param oldMaxFinalizationDelay The old value of `maxFinalizationDelay`.
    /// @param newMaxFinalizationDelay The new value of `maxFinalizationDelay`.
    event UpdateMaxFinalizationDelay(uint256 oldMaxFinalizationDelay, uint256 newMaxFinalizationDelay);

    /*************
     * Constants *
     *************/

    /// @notice The chain id of the corresponding layer 2 chain.
    uint64 public immutable layer2ChainId;

    /// @notice The address of L1MessageQueue contract.
    address public immutable messageQueue;

    /// @notice The address of RollupVerifier.
    address public immutable verifier;

    /***********
     * Structs *
     ***********/

    /// @param lastIndex The index of latest finalized batch
    /// @param timestamp The block timestamp of last finalization
    /// @param mode The current status, 1 means enforced mode, 0 means not.
    struct FinalizationState {
        uint128 lastIndex;
        uint64 timestamp;
        uint8 mode;
    }

    /*************
     * Variables *
     *************/

    /// @notice The maximum number of transactions allowed in each chunk.
    uint256 public maxNumTxInChunk;

    /// @dev The storage slot used as L1MessageQueue contract, which is deprecated now.
    address private __messageQueue;

    /// @dev The storage slot used as RollupVerifier contract, which is deprecated now.
    address private __verifier;

    /// @notice Whether an account is a sequencer.
    mapping(address => bool) public isSequencer;

    /// @notice Whether an account is a prover.
    mapping(address => bool) public isProver;

    /// @dev The storage slot used as lastFinalizedBatchIndex, which is deprecated now.
    uint256 private __lastFinalizedBatchIndex;

    /// @inheritdoc IScrollChain
    mapping(uint256 => bytes32) public override committedBatches;

    /// @inheritdoc IScrollChain
    mapping(uint256 => bytes32) public override finalizedStateRoots;

    /// @inheritdoc IScrollChain
    mapping(uint256 => bytes32) public override withdrawRoots;

    FinalizationState internal finalizationState;

    /// @notice The maximum finalization delay in seconds before entering the enforced mode.
    uint256 public maxFinalizationDelay;

    /// @notice The address of zk trie verifier.
    address public zkTrieVerifier;

    /**********************
     * Function Modifiers *
     **********************/

    modifier OnlySequencer() {
        // @note In the decentralized mode, it should be only called by a list of validator.
        require(isSequencer[_msgSender()], "caller not sequencer");
        _;
    }

    modifier OnlyProver() {
        require(isProver[_msgSender()], "caller not prover");
        _;
    }

    /***************
     * Constructor *
     ***************/

    /// @notice Constructor for `ScrollChain` implementation contract.
    ///
    /// @param _chainId The chain id of L2.
    /// @param _messageQueue The address of `L1MessageQueue` contract.
    /// @param _verifier The address of zkevm verifier contract.
    constructor(
        uint64 _chainId,
        address _messageQueue,
        address _verifier
    ) {
        if (_messageQueue == address(0) || _verifier == address(0)) {
            revert ErrorZeroAddress();
        }

        _disableInitializers();

        layer2ChainId = _chainId;
        messageQueue = _messageQueue;
        verifier = _verifier;
    }

    /// @notice Initialize the storage of ScrollChain.
    ///
    /// @dev The parameters `_messageQueue` are no longer used.
    ///
    /// @param _messageQueue The address of `L1MessageQueue` contract.
    /// @param _verifier The address of zkevm verifier contract.
    /// @param _maxNumTxInChunk The maximum number of transactions allowed in each chunk.
    function initialize(
        address _messageQueue,
        address _verifier,
        uint256 _maxNumTxInChunk
    ) public initializer {
        OwnableUpgradeable.__Ownable_init();

        maxNumTxInChunk = _maxNumTxInChunk;
        __verifier = _verifier;
        __messageQueue = _messageQueue;

        emit UpdateMaxNumTxInChunk(0, _maxNumTxInChunk);
    }

    function initializeV2(address _zkTrieVerifier) external reinitializer(2) {
        finalizationState = FinalizationState(uint128(__lastFinalizedBatchIndex), uint64(block.timestamp), 0);

        _updateZkTrieVerifier(_zkTrieVerifier);
        _updateMaxFinalizationDelay(1 weeks);
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IScrollChain
    function lastFinalizedBatchIndex() public view override returns (uint256) {
        return finalizationState.lastIndex;
    }

    /// @inheritdoc IScrollChain
    function isBatchFinalized(uint256 _batchIndex) external view override returns (bool) {
        return _batchIndex <= lastFinalizedBatchIndex();
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

        (uint256 memPtr, bytes32 _batchHash) = _loadBatchHeaderCalldata(_batchHeader);

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

        emit CommitBatch(0, _batchHash);
        emit FinalizeBatch(0, _batchHash, _stateRoot, bytes32(0));
    }

    /// @inheritdoc IScrollChain
    function commitBatch(
        uint8 _version,
        bytes calldata _parentBatchHeader,
        bytes[] memory _chunks,
        bytes calldata _skippedL1MessageBitmap
    ) external override OnlySequencer whenNotPaused {
        // if we are in enforced mode, exit from it.
        if (finalizationState.mode == 1) finalizationState.mode = 0;

        require(_version == 0, "invalid version");

        _commitBatch(_parentBatchHeader, _chunks, _skippedL1MessageBitmap);
    }

    /// @inheritdoc IScrollChain
    /// @dev If the owner want to revert a sequence of batches by sending multiple transactions,
    ///      make sure to revert recent batches first.
    function revertBatch(bytes calldata _batchHeader, uint256 _count) external {
        // if we are not in enforced mode, only owner can revert batches.
        // if we are in enforced mode, allow any users to revert batches.
        if (finalizationState.mode == 0) _checkOwner();

        require(_count > 0, "count must be nonzero");

        (uint256 memPtr, bytes32 _batchHash) = _loadBatchHeaderCalldata(_batchHeader);

        // check batch hash
        uint256 _batchIndex = BatchHeaderV0Codec.batchIndex(memPtr);
        require(committedBatches[_batchIndex] == _batchHash, "incorrect batch hash");
        // make sure no gap is left when reverting from the ending to the beginning.
        require(committedBatches[_batchIndex + _count] == bytes32(0), "reverting must start from the ending");

        // check finalization
        require(_batchIndex > lastFinalizedBatchIndex(), "can only revert unfinalized batch");

        while (_count > 0) {
            committedBatches[_batchIndex] = bytes32(0);

            emit RevertBatch(_batchIndex, _batchHash);

            unchecked {
                _batchIndex += 1;
                _count -= 1;
            }

            _batchHash = committedBatches[_batchIndex];
            if (_batchHash == bytes32(0)) break;
        }
    }

    /// @inheritdoc IScrollChain
    function finalizeBatchWithProof(
        bytes calldata _batchHeader,
        bytes32,
        bytes32 _postStateRoot,
        bytes32 _withdrawRoot,
        bytes calldata _aggrProof
    ) external override OnlyProver whenNotPaused {
        // compute batch hash and verify
        (uint256 memPtr, bytes32 _batchHash) = _loadBatchHeaderCalldata(_batchHeader);

        // finalize batch
        _finalizeBatch(memPtr, _batchHash, _postStateRoot, _withdrawRoot, _aggrProof);
    }

    /// @inheritdoc IScrollChain
    ///
    /// @dev This function can by used to commit and finalize a new batch in a
    /// single step if all previous batches are finalized. It can also be used
    /// to finalize the earliest pending batch. In this case, the provided batch
    /// should match the pending batch.
    ///
    /// If user choose to finalize a pending batch, the batch hash of current
    /// header should match with `committedBatches[currentIndex]`.
    /// Otherwise, `committedBatches[currentIndex]` should be `bytes32(0)`.
    function commitAndFinalizeBatchEnforced(
        bytes calldata _parentBatchHeader,
        bytes[] memory _chunks,
        bytes calldata _skippedL1MessageBitmap,
        bytes32 _postStateRoot,
        bytes32 _withdrawRoot,
        bytes calldata _withdrawRootProof,
        bytes calldata _aggrProof
    ) external {
        // check and enable enforced mode.
        if (finalizationState.mode == 0) {
            if (finalizationState.timestamp + maxFinalizationDelay < block.timestamp) {
                finalizationState.mode = 1;
            } else {
                revert("not allowed");
            }
        }

        (uint256 memPtr, bytes32 _batchHash) = _commitBatch(_parentBatchHeader, _chunks, _skippedL1MessageBitmap);

        (bytes32 stateRoot, bytes32 storageValue) = IZkTrieVerifier(zkTrieVerifier).verifyZkTrieProof(
            0x5300000000000000000000000000000000000000,
            bytes32(0),
            _withdrawRootProof
        );
        require(stateRoot == _postStateRoot, "state root mismatch");
        require(storageValue == _withdrawRoot, "withdraw root mismatch");

        _finalizeBatch(memPtr, _batchHash, _postStateRoot, _withdrawRoot, _aggrProof);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Add an account to the sequencer list.
    /// @param _account The address of account to add.
    function addSequencer(address _account) external onlyOwner {
        // @note Currently many external services rely on EOA sequencer to decode metadata directly from tx.calldata.
        // So we explicitly make sure the account is EOA.
        require(_account.code.length == 0, "not EOA");

        isSequencer[_account] = true;

        emit UpdateSequencer(_account, true);
    }

    /// @notice Remove an account from the sequencer list.
    /// @param _account The address of account to remove.
    function removeSequencer(address _account) external onlyOwner {
        isSequencer[_account] = false;

        emit UpdateSequencer(_account, false);
    }

    /// @notice Add an account to the prover list.
    /// @param _account The address of account to add.
    function addProver(address _account) external onlyOwner {
        // @note Currently many external services rely on EOA prover to decode metadata directly from tx.calldata.
        // So we explicitly make sure the account is EOA.
        require(_account.code.length == 0, "not EOA");
        isProver[_account] = true;

        emit UpdateProver(_account, true);
    }

    /// @notice Add an account from the prover list.
    /// @param _account The address of account to remove.
    function removeProver(address _account) external onlyOwner {
        isProver[_account] = false;

        emit UpdateProver(_account, false);
    }

    /// @notice Update the value of `maxNumTxInChunk`.
    /// @param _maxNumTxInChunk The new value of `maxNumTxInChunk`.
    function updateMaxNumTxInChunk(uint256 _maxNumTxInChunk) external onlyOwner {
        uint256 _oldMaxNumTxInChunk = maxNumTxInChunk;
        maxNumTxInChunk = _maxNumTxInChunk;

        emit UpdateMaxNumTxInChunk(_oldMaxNumTxInChunk, _maxNumTxInChunk);
    }

    /// @notice Pause the contract
    /// @param _status The pause status to update.
    function setPause(bool _status) external onlyOwner {
        if (_status) {
            _pause();
        } else {
            _unpause();
        }
    }

    function updateZkTrieVerifier(address _newZkTrieVerifier) external onlyOwner {
        _updateZkTrieVerifier(_newZkTrieVerifier);
    }

    function updateMaxFinalizationDelay(uint256 _newMaxFinalizationDelay) external onlyOwner {
        _updateMaxFinalizationDelay(_newMaxFinalizationDelay);
    }

    /**********************
     * Internal Functions *
     **********************/

    function _updateZkTrieVerifier(address _newZkTrieVerifier) internal {
        address _oldZkTrieVerifier = zkTrieVerifier;
        zkTrieVerifier = _newZkTrieVerifier;

        emit UpdateZkTrieVerifier(_oldZkTrieVerifier, _newZkTrieVerifier);
    }

    function _updateMaxFinalizationDelay(uint256 _newMaxFinalizationDelay) internal {
        uint256 _oldMaxFinalizationDelay = maxFinalizationDelay;
        maxFinalizationDelay = _newMaxFinalizationDelay;

        emit UpdateMaxFinalizationDelay(_oldMaxFinalizationDelay, _newMaxFinalizationDelay);
    }

    /// @dev Internal function to load batch header from calldata to memory.
    /// @param _batchHeader The batch header in calldata.
    /// @return memPtr The start memory offset of loaded batch header.
    /// @return _batchHash The hash of the loaded batch header.
    function _loadBatchHeaderCalldata(bytes calldata _batchHeader)
        internal
        pure
        returns (uint256 memPtr, bytes32 _batchHash)
    {
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

        // concatenate block contexts, use scope to avoid stack too deep
        {
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
            }
        }

        // It is used to compute the actual number of transactions in chunk.
        uint256 txHashStartDataPtr;
        assembly {
            txHashStartDataPtr := dataPtr
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
            require(_numTransactionsInBlock >= _numL1MessagesInBlock, "num txs less than num L1 msgs");
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

        // check the actual number of transactions in the chunk
        require((dataPtr - txHashStartDataPtr) / 32 <= maxNumTxInChunk, "too many txs in one chunk");

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
            uint256 rem;
            for (uint256 i = 0; i < _numL1Messages; i++) {
                uint256 quo = _totalL1MessagesPoppedInBatch >> 8;
                rem = _totalL1MessagesPoppedInBatch & 0xff;

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

            // check last L1 message is not skipped, _totalL1MessagesPoppedInBatch must > 0
            rem = (_totalL1MessagesPoppedInBatch - 1) & 0xff;
            require(((_bitmap >> rem) & 1) == 0, "cannot skip last L1 message");
        }

        return _ptr;
    }

    function _commitBatch(
        bytes calldata _parentBatchHeader,
        bytes[] memory _chunks,
        bytes calldata _skippedL1MessageBitmap
    ) internal returns (uint256 batchPtr, bytes32 batchHash) {
        // check whether the batch is empty
        require(_chunks.length > 0, "batch is empty");

        // The overall memory layout in this function is organized as follows
        // +---------------------+-------------------+------------------+
        // | parent batch header | chunk data hashes | new batch header |
        // +---------------------+-------------------+------------------+
        // ^                     ^                   ^
        // batchPtr              dataPtr             newBatchPtr (re-use var batchPtr)
        //
        // 1. We copy the parent batch header from calldata to memory starting at batchPtr
        // 2. We store `_chunks.length` number of Keccak hashes starting at `dataPtr`. Each Keccak
        //    hash corresponds to the data hash of a chunk. So we reserve the memory region from
        //    `dataPtr` to `dataPtr + _chunkLength * 32` for the chunk data hashes.
        // 3. The memory starting at `newBatchPtr` is used to store the new batch header and compute
        //    the batch hash.

        // the variable `batchPtr` will be reused later for the current batch
        bytes32 _parentBatchHash;
        (batchPtr, _parentBatchHash) = _loadBatchHeaderCalldata(_parentBatchHeader);

        uint256 _batchIndex = BatchHeaderV0Codec.batchIndex(batchPtr);
        uint256 _totalL1MessagesPoppedOverall = BatchHeaderV0Codec.totalL1MessagePopped(batchPtr);
        require(committedBatches[_batchIndex] == _parentBatchHash, "incorrect parent batch hash");
        require(committedBatches[_batchIndex + 1] == 0, "batch already committed");

        // compute the data hash for chunks
        (bytes32 _dataHash, uint256 _totalL1MessagesPoppedInBatch) = _commitChunks(
            _chunks,
            _totalL1MessagesPoppedOverall,
            _skippedL1MessageBitmap
        );
        _totalL1MessagesPoppedOverall += _totalL1MessagesPoppedInBatch;

        // reset `batchPtr` for current batch and reserve memory
        assembly {
            batchPtr := mload(0x40) // reset batchPtr
            mstore(0x40, add(batchPtr, add(89, _skippedL1MessageBitmap.length)))
            _batchIndex := add(_batchIndex, 1) // increase batch index
        }

        // store entries, the order matters
        BatchHeaderV0Codec.storeVersion(batchPtr, 0);
        BatchHeaderV0Codec.storeBatchIndex(batchPtr, _batchIndex);
        BatchHeaderV0Codec.storeL1MessagePopped(batchPtr, _totalL1MessagesPoppedInBatch);
        BatchHeaderV0Codec.storeTotalL1MessagePopped(batchPtr, _totalL1MessagesPoppedOverall);
        BatchHeaderV0Codec.storeDataHash(batchPtr, _dataHash);
        BatchHeaderV0Codec.storeParentBatchHash(batchPtr, _parentBatchHash);
        BatchHeaderV0Codec.storeSkippedBitmap(batchPtr, _skippedL1MessageBitmap);

        // compute batch hash
        batchHash = BatchHeaderV0Codec.computeBatchHash(batchPtr, 89 + _skippedL1MessageBitmap.length);

        bytes32 storedBatchHash = committedBatches[_batchIndex];
        if (finalizationState.mode == 1) {
            require(storedBatchHash == bytes32(0) || storedBatchHash == batchHash, "batch hash mismatch");
        } else {
            require(storedBatchHash == bytes32(0), "batch already committed");
        }
        if (storedBatchHash == bytes32(0)) {
            committedBatches[_batchIndex] = batchHash;
            emit CommitBatch(_batchIndex, batchHash);
        }
    }

    function _commitChunks(
        bytes[] memory _chunks,
        uint256 _totalL1MessagesPoppedOverall,
        bytes calldata _skippedL1MessageBitmap
    ) internal view returns (bytes32 dataHash, uint256 _totalL1MessagesPoppedInBatch) {
        uint256 _chunksLength = _chunks.length;
        // load `dataPtr` and reserve the memory region for chunk data hashes
        uint256 dataPtr;
        assembly {
            dataPtr := mload(0x40)
            mstore(0x40, add(dataPtr, mul(_chunksLength, 32)))
        }

        // compute the data hash for each chunk

        unchecked {
            for (uint256 i = 0; i < _chunksLength; i++) {
                uint256 _totalNumL1MessagesInChunk = _commitChunk(
                    dataPtr,
                    _chunks[i],
                    _totalL1MessagesPoppedInBatch,
                    _totalL1MessagesPoppedOverall,
                    _skippedL1MessageBitmap
                );
                _totalL1MessagesPoppedInBatch += _totalNumL1MessagesInChunk;
                _totalL1MessagesPoppedOverall += _totalNumL1MessagesInChunk;
                dataPtr += 32;
            }

            // check the length of bitmap
            require(
                ((_totalL1MessagesPoppedInBatch + 255) / 256) * 32 == _skippedL1MessageBitmap.length,
                "wrong bitmap length"
            );
        }

        assembly {
            let dataLen := mul(_chunksLength, 0x20)
            dataHash := keccak256(sub(dataPtr, dataLen), dataLen)
        }
    }

    function _finalizeBatch(
        uint256 memPtr,
        bytes32 _batchHash,
        bytes32 _postStateRoot,
        bytes32 _withdrawRoot,
        bytes calldata _aggrProof
    ) internal {
        require(_postStateRoot != bytes32(0), "new state root is zero");

        bytes32 _dataHash = BatchHeaderV0Codec.dataHash(memPtr);
        uint256 _batchIndex = BatchHeaderV0Codec.batchIndex(memPtr);
        require(committedBatches[_batchIndex] == _batchHash, "incorrect batch hash");

        // fetch previous state root from storage.
        bytes32 _prevStateRoot = finalizedStateRoots[_batchIndex - 1];

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
            FinalizationState memory cachedFinalizationState = finalizationState;
            require(uint256(cachedFinalizationState.lastIndex + 1) == _batchIndex, "incorrect batch index");
            cachedFinalizationState.lastIndex = uint128(_batchIndex);
            cachedFinalizationState.timestamp = uint64(block.timestamp);
            finalizationState = cachedFinalizationState;
        }

        // record state root and withdraw root
        finalizedStateRoots[_batchIndex] = _postStateRoot;
        withdrawRoots[_batchIndex] = _withdrawRoot;

        // Pop finalized and non-skipped message from L1MessageQueue.
        _popL1Messages(memPtr);

        emit FinalizeBatch(_batchIndex, _batchHash, _postStateRoot, _withdrawRoot);
    }

    function _popL1Messages(uint256 memPtr) internal {
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
    }
}
