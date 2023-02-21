// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

contract MockBridgeL1 {
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
    uint256 queueIndex,
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

  /// @dev The maximum number of transaction in on batch.
  uint256 public immutable maxNumTxInBatch;

  /// @dev The hash used for padding public inputs.
  bytes32 public immutable paddingTxHash;

  /***************************
   * Events from ScrollChain *
   ***************************/

  /// @notice Emitted when a new batch is commited.
  /// @param batchHash The hash of the batch
  event CommitBatch(bytes32 indexed batchHash);

  /// @notice Emitted when a batch is reverted.
  /// @param batchHash The identification of the batch.
  event RevertBatch(bytes32 indexed batchHash);

  /// @notice Emitted when a batch is finalized.
  /// @param batchHash The hash of the batch
  event FinalizeBatch(bytes32 indexed batchHash);

  /***********
   * Structs *
   ***********/

  struct BlockContext {
    // The hash of this block.
    bytes32 blockHash;
    // The parent hash of this block.
    bytes32 parentHash;
    // The height of this block.
    uint64 blockNumber;
    // The timestamp of this block.
    uint64 timestamp;
    // The base fee of this block.
    // Currently, it is not used, because we disable EIP-1559.
    // We keep it for future proof.
    uint256 baseFee;
    // The gas limit of this block.
    uint64 gasLimit;
    // The number of transactions in this block, both L1 & L2 txs.
    uint16 numTransactions;
    // The number of l1 messages in this block.
    uint16 numL1Messages;
  }

  struct Batch {
    // The list of blocks in this batch
    BlockContext[] blocks; // MAX_NUM_BLOCKS = 100, about 5 min
    // The state root of previous batch.
    // The first batch will use 0x0 for prevStateRoot
    bytes32 prevStateRoot;
    // The state root of the last block in this batch.
    bytes32 newStateRoot;
    // The withdraw trie root of the last block in this batch.
    bytes32 withdrawTrieRoot;
    // The index of the batch.
    uint64 batchIndex;
    // The parent batch hash.
    bytes32 parentBatchHash;
    // Concatenated raw data of RLP encoded L2 txs
    bytes l2Transactions;
  }

  struct L2MessageProof {
    // The hash of the batch where the message belongs to.
    bytes32 batchHash;
    // Concatenation of merkle proof for withdraw merkle trie.
    bytes merkleProof;
  }

  /*************
   * Variables *
   *************/

  /// @notice Message nonce, used to avoid relay attack.
  uint256 public messageNonce;

  /***************
   * Constructor *
   ***************/

  constructor() {
    maxNumTxInBatch = 4;
    paddingTxHash = 0xb5baa665b2664c3bfed7eb46e00ebc110ecf2ebd257854a9bf2b9dbc9b2c08f6;
  }

  /***********************************
   * Functions from L2GasPriceOracle *
   ***********************************/

  function setL2BaseFee(uint256) external {
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
      emit QueueTransaction(_sender, target, 0, messageNonce, gasLimit, _xDomainCalldata);
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

  function commitBatch(Batch memory _batch) external {
    _commitBatch(_batch);
  }

  function commitBatches(Batch[] memory _batches) external {
    for (uint256 i = 0; i < _batches.length; i++) {
      _commitBatch(_batches[i]);
    }
  }
  
  function revertBatch(bytes32 _batchHash) external {
    emit RevertBatch(_batchHash);
  }

  function finalizeBatchWithProof(
    bytes32 _batchHash,
    uint256[] memory,
    uint256[] memory
  ) external {
    emit FinalizeBatch(_batchHash);
  }

  /**********************
   * Internal Functions *
   **********************/

  function _commitBatch(Batch memory _batch) internal {
    bytes32 _batchHash = _computePublicInputHash(_batch);
    emit CommitBatch(_batchHash);
  }

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

  function applyL1ToL2Alias(address l1Address) internal pure returns (address l2Address) {
    uint160 offset = uint160(0x1111000000000000000000000000000000001111);
    unchecked {
      l2Address = address(uint160(l1Address) + offset);
    }
  }

  /// @dev Internal function to compute the public input hash.
  /// @param batch The batch to compute.
  function _computePublicInputHash(Batch memory batch)
    internal
    view
    returns (
      bytes32
    )
  {
    uint256 publicInputsPtr;
    // 1. append prevStateRoot, newStateRoot and withdrawTrieRoot to public inputs
    {
      bytes32 prevStateRoot = batch.prevStateRoot;
      bytes32 newStateRoot = batch.newStateRoot;
      bytes32 withdrawTrieRoot = batch.withdrawTrieRoot;
      // number of bytes in public inputs: 32 * 3 + 124 * blocks + 32 * MAX_NUM_TXS
      uint256 publicInputsSize = 32 * 3 + batch.blocks.length * 124 + 32 * maxNumTxInBatch;
      assembly {
        publicInputsPtr := mload(0x40)
        mstore(0x40, add(publicInputsPtr, publicInputsSize))
        mstore(publicInputsPtr, prevStateRoot)
        publicInputsPtr := add(publicInputsPtr, 0x20)
        mstore(publicInputsPtr, newStateRoot)
        publicInputsPtr := add(publicInputsPtr, 0x20)
        mstore(publicInputsPtr, withdrawTrieRoot)
        publicInputsPtr := add(publicInputsPtr, 0x20)
      }
    }

    uint64 numTransactionsInBatch;
    BlockContext memory _block;
    // 2. append block information to public inputs.
    for (uint256 i = 0; i < batch.blocks.length; i++) {
      // validate blocks, we won't check first block against previous batch.
      {
        BlockContext memory _currentBlock = batch.blocks[i];
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
          mstore(publicInputsPtr, blockHash)
          publicInputsPtr := add(publicInputsPtr, 0x20)
          mstore(publicInputsPtr, parentHash)
          publicInputsPtr := add(publicInputsPtr, 0x20)
        }
      }
      // append blockNumber and blockTimestamp to public inputs
      {
        uint256 blockNumber = _block.blockNumber;
        uint256 blockTimestamp = _block.timestamp;
        assembly {
          mstore(publicInputsPtr, shl(192, blockNumber))
          publicInputsPtr := add(publicInputsPtr, 0x8)
          mstore(publicInputsPtr, shl(192, blockTimestamp))
          publicInputsPtr := add(publicInputsPtr, 0x8)
        }
      }
      // append baseFee to public inputs
      {
        uint256 baseFee = _block.baseFee;
        assembly {
          mstore(publicInputsPtr, baseFee)
          publicInputsPtr := add(publicInputsPtr, 0x20)
        }
      }
      uint64 numTransactionsInBlock = _block.numTransactions;
      // gasLimit, numTransactions and numL1Messages to public inputs
      {
        uint256 gasLimit = _block.gasLimit;
        uint256 numL1MessagesInBlock = _block.numL1Messages;
        assembly {
          mstore(publicInputsPtr, shl(192, gasLimit))
          publicInputsPtr := add(publicInputsPtr, 0x8)
          mstore(publicInputsPtr, shl(240, numTransactionsInBlock))
          publicInputsPtr := add(publicInputsPtr, 0x2)
          mstore(publicInputsPtr, shl(240, numL1MessagesInBlock))
          publicInputsPtr := add(publicInputsPtr, 0x2)
        }
      }
      numTransactionsInBatch += numTransactionsInBlock;
    }
    require(numTransactionsInBatch <= maxNumTxInBatch, "Too many transactions in batch");

    // 3. append transaction hash to public inputs.
    uint256 _l2TxnPtr;
    {
      bytes memory l2Transactions = batch.l2Transactions;
      assembly {
        _l2TxnPtr := add(l2Transactions, 0x20)
      }
    }
    for (uint256 i = 0; i < batch.blocks.length; i++) {
      uint256 numL1MessagesInBlock = batch.blocks[i].numL1Messages;
      require(numL1MessagesInBlock == 0);
      uint256 numTransactionsInBlock = batch.blocks[i].numTransactions;
      for (uint256 j = numL1MessagesInBlock; j < numTransactionsInBlock; ++j) {
        bytes32 hash;
        assembly {
          let txPayloadLength := shr(224, mload(_l2TxnPtr))
          _l2TxnPtr := add(_l2TxnPtr, 4)
          _l2TxnPtr := add(_l2TxnPtr, txPayloadLength)
          hash := keccak256(sub(_l2TxnPtr, txPayloadLength), txPayloadLength)
          mstore(publicInputsPtr, hash)
          publicInputsPtr := add(publicInputsPtr, 0x20)
        }
      }
    }

    // 4. append padding transaction to public inputs.
    bytes32 txHashPadding = paddingTxHash;
    for (uint256 i = numTransactionsInBatch; i < maxNumTxInBatch; i++) {
      assembly {
        mstore(publicInputsPtr, txHashPadding)
        publicInputsPtr := add(publicInputsPtr, 0x20)
      }
    }

    // 5. compute public input hash
    bytes32 publicInputHash;
    {
      uint256 publicInputsSize = 32 * 3 + batch.blocks.length * 124 + 32 * maxNumTxInBatch;
      assembly {
        publicInputHash := keccak256(sub(publicInputsPtr, publicInputsSize), publicInputsSize)
      }
    }

    return publicInputHash;
  }
}
