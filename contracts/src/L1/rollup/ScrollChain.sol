// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {console2} from "forge-std/console2.sol";
import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import {IL1MessageQueue} from "./IL1MessageQueue.sol";
import {IScrollChain} from "./IScrollChain.sol";
import {BatchHeaderV0Codec} from "../../libraries/codec/BatchHeaderV0Codec.sol";
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
        require(isSequencer[msg.sender], "caller not sequencer");
        _;
    }

    /***************
     * Constructor *
     ***************/

    constructor(uint256 _chainId) {
        layer2ChainId = _chainId;
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
        require(BatchHeaderV0Codec.lastBlockHash(memPtr) != bytes32(0), "zero last block hash");
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
        bytes[] memory _chunks
    ) external override OnlySequencer {
        // check whether the batch is empty
        uint256 _chunksLength = _chunks.length;
        require(_chunksLength > 0, "batch is empty");

        // The variable `memPtr` will be reused for other purposes later.
        (uint256 memPtr, bytes32 _parentBatchHash) = _loadBatchHeader(_parentBatchHeader);

        uint256 _batchIndex = BatchHeaderV0Codec.batchIndex(memPtr);
        uint256 _totalL1MessagePopped = BatchHeaderV0Codec.totalL1MessagePopped(memPtr);
        bytes32 _parentBlockHash = BatchHeaderV0Codec.lastBlockHash(memPtr);
        require(committedBatches[_batchIndex] == _parentBatchHash, "incorrect parent batch bash");

        // compute data hash for each chunk
        // The list of data hashes are stored in memory start with `memPtr`.
        // The bitmap of all skipped messages are stored in memory start with `bitmapPtr`.
        uint256 bitmapPtr;
        assembly {
            memPtr := mload(0x40)
            mstore(0x40, add(memPtr, mul(_chunksLength, 32)))
            bitmapPtr := mload(0x40)
            mstore(0x40, add(bitmapPtr, mul(_chunksLength, 32)))
            mstore(bitmapPtr, 0) // clear memory entry
        }

        uint256 _numL1MessagesInBatch;
        for (uint256 i = 0; i < _chunksLength; i++) {
            uint256 _numL1MessagesInChunk;
            (_parentBlockHash, _numL1MessagesInChunk) = _commitChunk(
                memPtr,
                bitmapPtr,
                _numL1MessagesInBatch % 256,
                _chunks[i],
                _parentBlockHash,
                _totalL1MessagePopped
            );

            // load `numL1Messages` from memory
            assembly {
                _numL1MessagesInBatch := add(_numL1MessagesInBatch, _numL1MessagesInChunk)
                _totalL1MessagePopped := add(_totalL1MessagePopped, _numL1MessagesInChunk)
                memPtr := add(memPtr, 32)
            }
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
        BatchHeaderV0Codec.storeL1MessagePopped(memPtr, _numL1MessagesInBatch);
        BatchHeaderV0Codec.storeTotalL1MessagePopped(memPtr, _totalL1MessagePopped);
        BatchHeaderV0Codec.storeDataHash(memPtr, _dataHash);
        BatchHeaderV0Codec.storeLastBlockHash(memPtr, _parentBlockHash);
        BatchHeaderV0Codec.storeParentBatchHash(memPtr, _parentBatchHash);
        BatchHeaderV0Codec.storeBitMap(memPtr, bitmapPtr);

        // compute batch hash
        bytes32 _batchHash = _computeBatchHash(memPtr, 121 + ((_numL1MessagesInBatch + 255) / 256) * 32);

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
    function _loadBatchHeader(bytes calldata _batchHeader) internal view returns (uint256 memPtr, bytes32 _batchHash) {
        // load to memory
        uint256 _length;
        (memPtr, _length) = BatchHeaderV0Codec.loadAndValidate(_batchHeader);

        // compute batch hash
        _batchHash = _computeBatchHash(memPtr, _length);
    }

    /// @dev Internal function to commit a chunk.
    /// @param memPtr The start memory offset to store `dataHash`.
    /// @param bitmapPtr The start memory offset to store skippedL1MessageBitmap.
    /// @param bitmapBits The number of bits of the bitmap in memory `bitmapPtr`.
    /// @param _chunk The encoded chunk to commit.
    /// @param _prevBlockHash The block hash of parent block.
    /// @param _prevTotalL1MessagesPopped The total number of L1 message popped before current chunk.
    /// @return bytes32 The parent block hash.
    /// @return uint256 The total number of L1 message popped in current chunk
    function _commitChunk(
        uint256 memPtr,
        uint256 bitmapPtr,
        uint256 bitmapBits,
        bytes memory _chunk,
        bytes32 _prevBlockHash,
        uint256 _prevTotalL1MessagesPopped
    ) internal view returns (bytes32, uint256) {
        // should contain at least the number of the blocks
        require(_chunk.length > 0, "chunk length too small");

        uint256 chunkPtr;
        uint256 dataPtr;
        uint256 _numBlocks;
        assembly {
            dataPtr := mload(0x40)
            chunkPtr := add(_chunk, 0x20)
            _numBlocks := shr(248, mload(chunkPtr))
            chunkPtr := add(chunkPtr, 1)
        }
        // should contain at least one block
        require(_numBlocks > 0, "no block in chunk");

        // should contain at least the number of the blocks and block contexts
        require(_chunk.length >= 1 + _numBlocks * 156, "invalid chunk length");

        // concatenate block contexts
        for (uint256 i = 0; i < _numBlocks; i++) {
            bytes32 _parentHash;
            assembly {
                _parentHash := mload(add(chunkPtr, 0x20))
            }
            require(_parentHash == _prevBlockHash, "incorrect parent block hash");

            assembly {
                _prevBlockHash := mload(chunkPtr)
                for {
                    let j := 0
                } lt(j, 156) {
                    j := add(j, 0x20)
                } {
                    mstore(add(dataPtr, j), mload(add(chunkPtr, j)))
                }
                dataPtr := add(dataPtr, 156)
                chunkPtr := add(chunkPtr, 156)
            }
        }

        // concatenate tx hashes
        uint256 txPtr = chunkPtr;
        assembly {
            chunkPtr := add(_chunk, 0x21)
        }
        uint256 _totalNumL1MessagesInChunk;
        for (uint256 i = 0; i < _numBlocks; i++) {
            uint256 _numL1MessagesInBlock;
            uint256 _skippedL1MessageBitmapInBlock;
            assembly {
                _numL1MessagesInBlock := shr(240, mload(add(chunkPtr, 122)))
                _skippedL1MessageBitmapInBlock := mload(add(chunkPtr, 124))
            }
            require(_numL1MessagesInBlock <= 256, "block includes too much L1 messages");

            // concatenate l1 messages
            dataPtr = _loadL1Messages(
                dataPtr,
                _numL1MessagesInBlock,
                _prevTotalL1MessagesPopped,
                _skippedL1MessageBitmapInBlock
            );

            // update local chunk state variable
            assembly {
                // update bitmap entry
                mstore(bitmapPtr, or(mload(bitmapPtr), shl(bitmapBits, _skippedL1MessageBitmapInBlock)))
                bitmapBits := add(bitmapBits, _numL1MessagesInBlock)
                if gt(bitmapBits, 256) {
                    // cannot fit in single uint256, store extra parts to next entry
                    bitmapPtr := add(bitmapPtr, 32)
                    bitmapBits := sub(bitmapBits, 256)
                    mstore(bitmapPtr, shr(sub(_numL1MessagesInBlock, bitmapBits), _skippedL1MessageBitmapInBlock))
                }

                // update counts
                _totalNumL1MessagesInChunk := add(_totalNumL1MessagesInChunk, _numL1MessagesInBlock)
                _prevTotalL1MessagesPopped := add(_prevTotalL1MessagesPopped, _numL1MessagesInBlock)
            }

            uint256 _numTransactionsInBlock;
            assembly {
                _numTransactionsInBlock := shr(240, mload(add(chunkPtr, 120)))
                chunkPtr := add(chunkPtr, 156)
            }

            // concatenate l2 transactions
            for (uint256 j = _numL1MessagesInBlock; j < _numTransactionsInBlock; j++) {
                assembly {
                    // first 4 bytes indicate the length
                    let txPayloadLength := shr(224, mload(txPtr))
                    txPtr := add(txPtr, 4)
                    txPtr := add(txPtr, txPayloadLength)
                    let txHash := keccak256(sub(txPtr, txPayloadLength), txPayloadLength)
                    mstore(dataPtr, txHash)
                    dataPtr := add(dataPtr, 0x20)
                }
            }
        }

        // check chunk has correct length
        assembly {
            chunkPtr := add(_chunk, 0x20)
        }
        require(txPtr - chunkPtr == _chunk.length, "chunk length mismatch");

        // compute data hash and store to memory
        assembly {
            let startPtr := mload(0x40)
            let dataHash := keccak256(startPtr, sub(dataPtr, startPtr))

            mstore(memPtr, dataHash)
        }

        return (_prevBlockHash, _totalNumL1MessagesInChunk);
    }

    /// @dev Internal function to load L1 messages from message queue.
    /// @param _ptr The memory offset to store the transaction hash.
    /// @param _numL1Messages The number of L1 messages to load.
    /// @param _prevTotalL1MessagesPopped The total number of L1 messages to loaded before.
    /// @param _skippedL1MessageBitmap A bitmap indicates which message is skipped.
    /// @return uint256 The new memory offset after loading.
    function _loadL1Messages(
        uint256 _ptr,
        uint256 _numL1Messages,
        uint256 _prevTotalL1MessagesPopped,
        uint256 _skippedL1MessageBitmap
    ) internal view returns (uint256) {
        if (_numL1Messages == 0) return _ptr;
        IL1MessageQueue _messageQueue = IL1MessageQueue(messageQueue);

        for (uint256 j = 0; j < _numL1Messages; j++) {
            if (((_skippedL1MessageBitmap >> j) & 1) == 0) {
                bytes32 _hash = _messageQueue.getCrossDomainMessage(_prevTotalL1MessagesPopped + j);
                assembly {
                    mstore(_ptr, _hash)
                    _ptr := add(_ptr, 0x20)
                }
            }
        }
        return _ptr;
    }

    /// @dev Internal function to compute the batch hash.
    /// Caller should make sure that the encoded batch header is correct.
    ///
    /// @param _offset The memory offset of the encoded batch header.
    /// @param _length The length of the batch.
    /// @return _batchHash The hash of the corresponding batch.
    function _computeBatchHash(uint256 _offset, uint256 _length) internal pure returns (bytes32 _batchHash) {
        // in current version, the hash is: keccak(BatchHeader without timestamp)
        assembly {
            _batchHash := keccak256(_offset, _length)
        }
    }
}
