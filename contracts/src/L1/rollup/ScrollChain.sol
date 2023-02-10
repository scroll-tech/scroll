// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import { IScrollChain } from "./IScrollChain.sol";
import { RollupVerifier } from "../../libraries/verifier/RollupVerifier.sol";

// solhint-disable reason-string

/// @title ScrollChain
/// @notice This contract maintains essential data for scroll rollup, including:
///
/// 1. a list of pending messages, which will be relayed to layer 2;
/// 2. the block tree generated by layer 2 and it's status.
///
/// @dev the message queue is not used yet, the offline relayer only use events in `L1ScrollMessenger`.
contract ScrollChain is OwnableUpgradeable, IScrollChain {
  /**********
   * Events *
   **********/

  /// @notice Emitted when owner updates the status of sequencer.
  /// @param account The address of account updated.
  /// @param status The status of the account updated.
  event UpdateSequencer(address indexed account, bool status);

  /*************
   * Constants *
   *************/

  /// @dev The maximum number of transaction in on batch.
  uint256 private constant MAX_NUM_TX_IN_BATCH = 100;

  /// @dev The hash used for padding public inputs.
  bytes32 private constant PADDED_TX_HASH = bytes32(0);

  /// @notice The chain id of the corresponding layer 2 chain.
  uint256 public immutable layer2ChainId;

  /***********
   * Structs *
   ***********/

  // subject to change
  struct BatchStored {
    // The state root of previous batch.
    // The first batch will use 0x0 for prevStateRoot
    bytes32 prevStateRoot;
    // The state root of the last block in this batch.
    bytes32 newStateRoot;
    // The withdraw trie root of the last block in this batch.
    bytes32 withdrawTrieRoot;
    // The hash of public input.
    bytes32 publicInputHash;
    // The index of the batch.
    uint64 batchIndex;
    // The timestamp of the last block in this batch.
    uint64 timestamp;
    // The number of transactions in this batch, both L1 & L2 txs.
    uint64 numTransactions;
    // The number of l1 messages in this batch.
    uint64 numL1Messages;
    // Whether the batch is finalized.
    bool finalized;
    // do we need to store the parent hash of this batch?
  }

  /*************
   * Variables *
   *************/

  /// @notice Whether an account is a sequencer.
  mapping(address => bool) public isSequencer;

  /// @notice The latest finalized batch hash.
  bytes32 public lastFinalizedBatchHash;

  /// @notice Mapping from batch id to batch struct.
  mapping(bytes32 => BatchStored) public batches;

  /// @notice Mapping from batch index to finalized batch hash.
  mapping(uint256 => bytes32) public finalizedBatches;

  /**********************
   * Function Modifiers *
   **********************/

  modifier OnlySequencer() {
    // @todo In the decentralize mode, it should be only called by a list of validator.
    require(isSequencer[msg.sender], "caller not sequencer");
    _;
  }

  /***************
   * Constructor *
   ***************/

  constructor(uint256 _chainId) {
    layer2ChainId = _chainId;
  }

  function initialize() public initializer {
    OwnableUpgradeable.__Ownable_init();
  }

  /*************************
   * Public View Functions *
   *************************/

  /// @inheritdoc IScrollChain
  function isBatchFinalized(bytes32 _batchHash) external view override returns (bool) {
    return batches[_batchHash].finalized;
  }

  /// @inheritdoc IScrollChain
  function isBatchFinalized(uint256 _batchIndex) external view override returns (bool) {
    return finalizedBatches[_batchIndex] != bytes32(0);
  }

  /// @inheritdoc IScrollChain
  function layer2GasLimit(uint256) public view virtual override returns (uint256) {
    // hardcode for now
    return 30000000;
  }

  /// @inheritdoc IScrollChain
  function getL2MessageRoot(bytes32 _batchHash) external view override returns (bytes32) {
    return batches[_batchHash].withdrawTrieRoot;
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Import layer 2 genesis block
  function importGenesisBatch(Batch memory _genesisBatch) external {
    require(lastFinalizedBatchHash == bytes32(0), "Genesis batch imported");
    require(_genesisBatch.blocks.length == 1, "Not exact one block in genesis");
    require(_genesisBatch.prevStateRoot == bytes32(0), "Nonzero prevStateRoot");

    BlockContext memory _genesisBlock = _genesisBatch.blocks[0];

    require(_genesisBlock.blockHash != bytes32(0), "Block hash is zero");
    require(_genesisBlock.blockNumber == 0, "Block is not genesis");
    require(_genesisBlock.parentHash == bytes32(0), "Parent hash not empty");

    bytes32 _batchHash = _commitBatch(_genesisBatch);

    lastFinalizedBatchHash = _batchHash;
    finalizedBatches[0] = _batchHash;
    batches[_batchHash].finalized = true;

    emit FinalizeBatch(_batchHash);
  }

  /// @inheritdoc IScrollChain
  function commitBatch(Batch memory _batch) public override OnlySequencer {
    _commitBatch(_batch);
  }

  /// @inheritdoc IScrollChain
  function commitBatches(Batch[] memory _batches) public override OnlySequencer {
    for (uint256 i = 0; i < _batches.length; i++) {
      _commitBatch(_batches[i]);
    }
  }

  /// @inheritdoc IScrollChain
  function revertBatch(bytes32 _batchHash) external override OnlySequencer {
    BatchStored storage _batch = batches[_batchHash];

    require(_batch.publicInputHash != bytes32(0), "No such batch");
    require(!_batch.finalized, "Unable to revert verified batch");

    // delete commited batch
    delete batches[_batchHash];

    emit RevertBatch(_batchHash);
  }

  /// @inheritdoc IScrollChain
  function finalizeBatchWithProof(
    bytes32 _batchHash,
    uint256[] memory _proof,
    uint256[] memory _instances
  ) external override OnlySequencer {
    BatchStored storage _batch = batches[_batchHash];
    require(_batch.publicInputHash != bytes32(0), "No such batch");
    require(!_batch.finalized, "Batch already verified");

    // @note skip parent check for now, since we may not prove blocks in order.
    // bytes32 _parentHash = _block.header.parentHash;
    // require(lastFinalizedBlockHash == _parentHash, "parent not latest finalized");
    // this check below is not needed, just incase
    // require(blocks[_parentHash].verified, "parent not verified");

    // @todo add verification logic
    RollupVerifier.verify(_proof, _instances);

    uint256 _batchIndex = _batch.batchIndex;
    finalizedBatches[_batchIndex] = _batchHash;
    _batch.finalized = true;

    BatchStored storage _finalizedBatch = batches[lastFinalizedBatchHash];
    if (_batchIndex > _finalizedBatch.batchIndex) {
      lastFinalizedBatchHash = _batchHash;
    }

    emit FinalizeBatch(_batchHash);
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

  /**********************
   * Internal Functions *
   **********************/

  /// @dev Internal function to commit a batch.
  /// @param _batch The batch to commit.
  function _commitBatch(Batch memory _batch) internal returns (bytes32) {
    // check whether the batch is empty
    require(_batch.blocks.length > 0, "Batch is empty");

    uint256 publicInputsStartPtr;
    uint256 publicInputsStartOffset;
    uint256 publicInputsSize;
    // append prevStateRoot, newStateRoot and withdrawTrieRoot to public inputs
    {
      bytes32 prevStateRoot = _batch.prevStateRoot;
      bytes32 newStateRoot = _batch.newStateRoot;
      bytes32 withdrawTrieRoot = _batch.withdrawTrieRoot;
      // number of bytes in public inputs: 32 * 3 + 124 * blocks + 32 * MAX_NUM_TXS
      publicInputsSize = 32 * 3 + _batch.blocks.length * 124 + 32 * MAX_NUM_TX_IN_BATCH;
      assembly {
        publicInputsStartPtr := mload(0x40)
        publicInputsStartOffset := publicInputsStartPtr
        mstore(0x40, add(publicInputsStartPtr, publicInputsSize))

        mstore(publicInputsStartOffset, prevStateRoot)
        publicInputsStartOffset := add(publicInputsStartOffset, 0x20)
        mstore(publicInputsStartOffset, newStateRoot)
        publicInputsStartOffset := add(publicInputsStartOffset, 0x20)
        mstore(publicInputsStartOffset, withdrawTrieRoot)
        publicInputsStartOffset := add(publicInputsStartOffset, 0x20)
      }
    }

    uint256 numTransactionsInBatch;
    uint256 numL1MessagesInBatch;
    BlockContext memory _block;
    // append block information to public inputs.
    for (uint256 i = 0; i < _batch.blocks.length; i++) {
      // validate blocks
      // @todo also check first block against previous batch.
      {
        BlockContext memory _currentBlock = _batch.blocks[i];
        if (i > 0) {
          require(_block.blockHash == _currentBlock.parentHash, "Parent hash mismatch");
          require(_block.blockNumber + 1 == _currentBlock.blockNumber, "Block number mismatch");
        }
        _block = _currentBlock;
      }

      // append blockHash and parentHash to public inputs
      {
        bytes32 blockHash = _block.blockHash;
        bytes32 parentHash = _block.parentHash;
        assembly {
          mstore(publicInputsStartOffset, blockHash)
          publicInputsStartOffset := add(publicInputsStartOffset, 0x20)
          mstore(publicInputsStartOffset, parentHash)
          publicInputsStartOffset := add(publicInputsStartOffset, 0x20)
        }
      }
      // append blockNumber and blockTimestamp to public inputs
      {
        uint256 blockNumber = _block.blockNumber;
        uint256 blockTimestamp = _block.timestamp;
        assembly {
          mstore(publicInputsStartOffset, shl(192, blockNumber))
          publicInputsStartOffset := add(publicInputsStartOffset, 0x8)
          mstore(publicInputsStartOffset, shl(192, blockTimestamp))
          publicInputsStartOffset := add(publicInputsStartOffset, 0x8)
        }
      }
      // append baseFee to public inputs
      {
        uint256 baseFee = _block.baseFee;
        assembly {
          mstore(publicInputsStartOffset, baseFee)
          publicInputsStartOffset := add(publicInputsStartOffset, 0x20)
        }
      }
      uint256 numTransactionsInBlock = _block.numTransactions;
      uint256 numL1MessagesInBlock = _block.numL1Messages;
      // gasLimit, numTransactions and numL1Messages to public inputs
      {
        uint256 gasLimit = _block.gasLimit;
        assembly {
          mstore(publicInputsStartOffset, shl(192, gasLimit))
          publicInputsStartOffset := add(publicInputsStartOffset, 0x8)
          mstore(publicInputsStartOffset, shl(240, numTransactionsInBlock))
          publicInputsStartOffset := add(publicInputsStartOffset, 0x2)
          mstore(publicInputsStartOffset, shl(240, numL1MessagesInBlock))
          publicInputsStartOffset := add(publicInputsStartOffset, 0x2)
        }
      }

      unchecked {
        numTransactionsInBatch += numTransactionsInBlock;
        numL1MessagesInBatch += numL1MessagesInBlock;
      }
    }

    require(numTransactionsInBatch <= MAX_NUM_TX_IN_BATCH, "Too many transactions in batch");

    // @todo append transaction information to public inputs.
    // @note it is complicated while dealing rlp encoding, ignore it for now.
    bytes32 txHashPadding = PADDED_TX_HASH;
    for (uint256 i = 0; i < numTransactionsInBatch; i++) {
      assembly {
        mstore(publicInputsStartOffset, txHashPadding)
        publicInputsStartOffset := add(publicInputsStartOffset, 0x20)
      }
    }

    // compute batch hash
    bytes32 publicInputHash;
    assembly {
      publicInputHash := keccak256(publicInputsStartPtr, publicInputsSize)
    }

    // @todo maybe use publicInputHash as batchHash later.
    bytes32 _batchHash = _computeBatchId(_block.blockHash, _batch.blocks[0].parentHash, _batch.batchIndex);

    BatchStored storage _batchInStorage = batches[_batchHash];

    // @todo maybe add parent batch check later.
    require(_batchInStorage.publicInputHash == bytes32(0), "Batch already commited");
    _batchInStorage.prevStateRoot = _batch.prevStateRoot;
    _batchInStorage.newStateRoot = _batch.newStateRoot;
    _batchInStorage.withdrawTrieRoot = _batch.withdrawTrieRoot;
    _batchInStorage.publicInputHash = publicInputHash;
    _batchInStorage.batchIndex = _batch.batchIndex;
    _batchInStorage.timestamp = _block.timestamp;
    _batchInStorage.numTransactions = uint64(numTransactionsInBatch);
    _batchInStorage.numL1Messages = uint64(numL1MessagesInBatch);

    emit CommitBatch(_batchHash);

    return _batchHash;
  }

  /// @dev Internal function to compute a unique batch id for mapping.
  /// @param _lastBlockHash The block hash of the last block in the batch.
  /// @param _parentHash The parent block hash of the first block the batch.
  /// @param _batchIndex The index of the batch.
  /// @return Return the computed batch id.
  function _computeBatchId(
    bytes32 _lastBlockHash,
    bytes32 _parentHash,
    uint256 _batchIndex
  ) internal pure returns (bytes32) {
    return keccak256(abi.encode(_lastBlockHash, _parentHash, _batchIndex));
  }
}
