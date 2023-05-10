// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import {IL1MessageQueue} from "./IL1MessageQueue.sol";
import {IScrollChain} from "./IScrollChain.sol";
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

    /***********
     * Structs *
     ***********/

    // subject to change
    struct BatchStored {
        // The state root of the last block in this batch.
        bytes32 newStateRoot;
        // The withdraw trie root of the last block in this batch.
        bytes32 withdrawTrieRoot;
        // The parent batch hash.
        bytes32 parentBatchHash;
        // The index of the batch.
        uint64 batchIndex;
        // The timestamp of the last block in this batch.
        uint64 timestamp;
        // The number of transactions in this batch, both L1 & L2 txs.
        uint64 numTransactions;
        // The total number of L1 messages included after this batch.
        uint64 totalL1Messages;
        // Whether the batch is finalized.
        bool finalized;
    }

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
    function importGenesisBatch(
        bytes calldata _batchHeader,
        bytes32 _stateRoot,
        bytes32 _withdrawRoot
    ) external {
        // check parent batch length
        require(_batchHeader.length == 161, "invalid batch header length");
        require(_stateRoot != bytes32(0), "zero state root");

        // check whether the genesis batch is imported
        require(finalizedStateRoots[0] == bytes32(0), "Genesis batch imported");

        // load batch header to memory
        uint256 _batchHeaderOffset;
        assembly {
            _batchHeaderOffset := mload(0x40)
            mstore(0x40, add(_batchHeaderOffset, 161))
            // copy parent batch header to memory.
            calldatacopy(_batchHeaderOffset, _batchHeader.offset, 161)
        }

        // check all fields except `dataHash` are zero
        {
            uint256 _sumOfFields;
            bytes32 _dataHash;
            bytes32 _lastBlockHash;
            assembly {
                // load `version` from batch header
                _sumOfFields := add(_sumOfFields, shr(248, mload(_batchHeaderOffset)))
                // load `batchIndex` from batch header
                _sumOfFields := add(_sumOfFields, shr(192, mload(add(_batchHeaderOffset, 1))))
                // load `l1MessagePopped` from batch header
                _sumOfFields := add(_sumOfFields, shr(192, mload(add(_batchHeaderOffset, 9))))
                // load `totalL1MessagePopped` from batch header
                _sumOfFields := add(_sumOfFields, shr(192, mload(add(_batchHeaderOffset, 17))))
                // load `dataHash` from batch header
                _dataHash := mload(add(_batchHeaderOffset, 25))
                // load `lastBlockHash` from batch header
                _lastBlockHash := mload(add(_batchHeaderOffset, 57))
                // store timestamp for current batch header
                mstore(add(_batchHeaderOffset, 153), shl(192, timestamp()))
            }
            require(_sumOfFields == 0, "not all fields are zero");
            require(_dataHash != bytes32(0), "zero data hash");
            require(_lastBlockHash == bytes32(0), "nonzero last block hash");
        }

        // compute parent batch hash and check
        bytes32 _batchHash = _computeBatchHash(_batchHeaderOffset);

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
        require(_chunks.length > 0, "batch is empty");
        // check parent batch length
        require(_parentBatchHeader.length == 161, "invalid batch header length");

        bytes32 _batchHash;
        uint256 _batchIndex;
        uint256 _totalL1MessagePopped;
        bytes32 _parantBlockHash;

        // @note This memory is first used to store parent batch header and then current batch header
        uint256 _batchHeaderOffset;
        assembly {
            _batchHeaderOffset := mload(0x40)
            mstore(0x40, add(_batchHeaderOffset, 161))
            // copy parent batch header to memory.
            calldatacopy(_batchHeaderOffset, _parentBatchHeader.offset, 161)
            // load parent batch index from parent batch header
            _batchIndex := shr(192, mload(add(_batchHeaderOffset, 1)))
            // load total l1 message popped from parent batch header
            _totalL1MessagePopped := shr(192, mload(add(_batchHeaderOffset, 17)))
            // load parent block hash from parent batch header
            _parantBlockHash := mload(add(_batchHeaderOffset, 57))
        }

        // compute parent batch hash and check
        _batchHash = _computeBatchHash(_batchHeaderOffset);
        require(committedBatches[_batchIndex] == _batchHash, "incorrect parent batch bash");

        // compute data hash for each chunk
        bytes32[] memory _dataHashes = new bytes32[](_chunks.length);
        uint256 _l1MessagePopped;
        for (uint256 i = 0; i < _chunks.length; i++) {
            uint256 _numPopped;
            (_dataHashes[i], _parantBlockHash, _numPopped) = _commitChunk(
                _chunks[i],
                _parantBlockHash,
                _totalL1MessagePopped
            );
            unchecked {
                _l1MessagePopped += _numPopped;
                _totalL1MessagePopped += _numPopped;
            }
        }

        // compute current batch hash
        assembly {
            _batchIndex := add(_batchIndex, 1)
            let _dataHash := keccak256(add(_dataHashes, 0x20), mload(_dataHashes))
            // store version for current batch header
            mstore(_batchHeaderOffset, shl(248, _version))
            // store batchIndex for current batch header
            mstore(add(_batchHeaderOffset, 1), shl(192, _batchIndex))
            // store l1MessagePopped for current batch header
            mstore(add(_batchHeaderOffset, 9), shl(192, _l1MessagePopped))
            // store totalL1MessagePopped for current batch header
            mstore(add(_batchHeaderOffset, 17), shl(192, _totalL1MessagePopped))
            // store dataHash for current batch header
            mstore(add(_batchHeaderOffset, 25), _dataHash)
            // store lastBlockHash for current batch header
            mstore(add(_batchHeaderOffset, 57), _parantBlockHash)
            // store timestamp for current batch header
            mstore(add(_batchHeaderOffset, 153), shl(192, timestamp()))
        }
        _batchHash = _computeBatchHash(_batchHeaderOffset);

        committedBatches[_batchIndex] = _batchHash;
        emit CommitBatch(_batchHash);
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
        bytes32 _batchHash;
        bytes32 _dataHash;
        uint256 _batchIndex;
        {
            uint256 _offset;
            assembly {
                _offset := mload(0x40)
                // copy parent batch header to memory.
                calldatacopy(_offset, _batchHeader.offset, 161)
                // load parent batch index.
                _batchIndex := shr(192, mload(add(_offset, 1)))
                // load data hash
                _dataHash := mload(add(_offset, 25))
            }
            _batchHash = _computeBatchHash(_offset);
            require(committedBatches[_batchIndex] == _batchHash, "incorrect batch bash");
        }

        // verify previous state root.
        require(finalizedStateRoots[_batchIndex - 1] == _prevStateRoot, "incorrect previous state root");

        // avoid duplicated verification
        require(finalizedStateRoots[_batchIndex] == bytes32(0), "batch already verified");

        // compute public input hash
        bytes32 _publicInuptHash = keccak256(abi.encode(_prevStateRoot, _newStateRoot, _withdrawRoot, _dataHash));

        // verify batch
        IRollupVerifier(verifier).verifyAggregateProof(_aggrProof, _publicInuptHash);

        // record state root and withdraw root
        finalizedStateRoots[_batchIndex] = _newStateRoot;
        withdrawRoots[_batchIndex] = _withdrawRoot;

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

    /// @dev Internal function to commit a chunk.
    /// @param _chunk The encoded chunk to commit.
    /// @param _prevBlockHash The block hash of parent block.
    /// @param _prevTotalL1MessagesPopped The total number of L1 message popped before.
    function _commitChunk(
        bytes memory _chunk,
        bytes32 _prevBlockHash,
        uint256 _prevTotalL1MessagesPopped
    )
        internal
        view
        returns (
            bytes32,
            bytes32,
            uint256
        )
    {
        // should contain at least the number of the blocks
        require(_chunk.length > 0, "chunk length too small");

        uint256 _chunkPtr;
        uint256 _numBlocks;
        uint256 _dataPtr;
        assembly {
            _dataPtr := mload(0x40)
            _chunkPtr := add(_chunk, 0x20)
            _numBlocks := mload(_chunkPtr)
        }
        // should contain at least the number of the blocks and block contexts
        require(_chunk.length >= 1 + _numBlocks * 156, "invalid chunk length");

        // concatenate block contexts
        for (uint256 i = 0; i < _numBlocks; i++) {
            bytes32 _parentHash;
            assembly {
                _parentHash := mload(add(_chunkPtr, 0x20))
            }
            require(_parentHash == _prevBlockHash, "incorrect parent hash");

            assembly {
                _prevBlockHash := mload(_chunkPtr)
                for {
                    let j := 0
                } lt(j, 156) {
                    j := add(j, 0x20)
                } {
                    mstore(add(_dataPtr, j), mload(add(_chunkPtr, j)))
                }
                _dataPtr := add(_dataPtr, 156)
                _chunkPtr := add(_chunkPtr, 156)
            }
        }

        // concatenate tx hashes
        uint256 _txPtr = _chunkPtr;
        assembly {
            _chunkPtr := add(_chunk, 0x20)
        }
        uint256 _totalNumL1MessagesInChunk;
        for (uint256 i = 0; i < _numBlocks; i++) {
            uint256 _numTransactionsInBlock;
            uint256 _numL1MessagesInBlock;
            uint256 _skippedL1MessageBitmap;
            assembly {
                _numTransactionsInBlock := shr(240, mload(add(_chunkPtr, 120)))
                _numL1MessagesInBlock := shr(240, mload(add(_chunkPtr, 122)))
                _skippedL1MessageBitmap := mload(add(_chunkPtr, 124))
                _chunkPtr := add(_chunkPtr, 156)
            }
            require(_numL1MessagesInBlock <= 256, "include too much L1 messages");

            // concatenate l1 messages
            _dataPtr = _loadL1Messages(
                _dataPtr,
                _numL1MessagesInBlock,
                _prevTotalL1MessagesPopped,
                _skippedL1MessageBitmap
            );
            unchecked {
                _totalNumL1MessagesInChunk += _numL1MessagesInBlock;
                _prevTotalL1MessagesPopped += _numL1MessagesInBlock;
            }

            // concatenate l2 transactions
            for (uint256 j = _numL1MessagesInBlock; j < _numTransactionsInBlock; j++) {
                assembly {
                    // first 4 bytes indicate the length
                    let txPayloadLength := shr(224, mload(_txPtr))
                    _txPtr := add(_txPtr, 4)
                    _txPtr := add(_txPtr, txPayloadLength)
                    let txHash := keccak256(sub(_txPtr, txPayloadLength), txPayloadLength)
                    mstore(_dataPtr, txHash)
                    _dataPtr := add(_dataPtr, 0x20)
                }
            }
        }

        // check chunk has correct length
        assembly {
            _chunkPtr := add(_chunk, 0x20)
        }
        require(_txPtr - _chunkPtr == _chunk.length, "chunk length mismatch");

        // compute data hash
        bytes32 _dataHash;
        assembly {
            let _startPtr := mload(0x40)
            _dataHash := keccak256(_startPtr, sub(_dataPtr, _startPtr))
        }

        return (_dataHash, _prevBlockHash, _totalNumL1MessagesInChunk);
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
    /// @return _batchHash The hash of the corresponding batch.
    function _computeBatchHash(uint256 _offset) internal pure returns (bytes32 _batchHash) {
        assembly {
            // mstore: batch hash without timestamp, 89 = 1 + 8 + 8 + 8 + 32 + 32
            mstore(0x00, keccak256(_offset, 89))
            // mstore: block timestamp of the batch
            mstore(0x20, shr(192, mload(add(_offset, 153))))
            // compute batch hash
            _batchHash := keccak256(0x00, 0x40)
        }
    }
}
