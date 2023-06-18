// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import {BatchHeaderV0Codec} from "../../contracts/src/libraries/codec/BatchHeaderV0Codec.sol";

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

  /***************************
   * Events from ScrollChain *
   ***************************/

  /// @notice Emitted when a new batch is committed.
  /// @param batchHash The hash of the batch.
  event CommitBatch(bytes32 indexed batchHash);

  /// @notice Emitted when a batch is finalized.
  /// @param batchHash The hash of the batch
  /// @param stateRoot The state root in layer 2 after this batch.
  /// @param withdrawRoot The merkle root in layer2 after this batch.
  event FinalizeBatch(bytes32 indexed batchHash, bytes32 stateRoot, bytes32 withdrawRoot);

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

  /// @notice Message nonce, used to avoid relay attack.
  uint256 public messageNonce;

  mapping(uint256 => bytes32) public committedBatches;

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

  function commitBatch(
    uint8 version,
    bytes calldata parentBatchHeader,
    bytes[] memory chunks,
    bytes calldata /* skippedL1MessageBitmap */
  ) external {
    require(version == 0, "invalid version");

    // check whether the batch is empty
    uint256 _chunksLength = chunks.length;
    require(_chunksLength > 0, "batch is empty");

    // the variable `batchPtr` will be reused later for the current batch
    (uint256 batchPtr,) = _loadBatchHeader(parentBatchHeader);

    uint256 _batchIndex = BatchHeaderV0Codec.batchIndex(batchPtr);
    bytes32 _batchHash = bytes32(_batchIndex + 1);

    committedBatches[_batchIndex] = _batchHash;
    emit CommitBatch(_batchHash);
  }

  function finalizeBatchWithProof(
    bytes calldata batchHeader,
    bytes32 /*prevStateRoot*/,
    bytes32 postStateRoot,
    bytes32 withdrawRoot,
    bytes calldata /*aggrProof*/
  ) external {
    (, bytes32 _batchHash) = _loadBatchHeader(batchHeader);
    emit FinalizeBatch(_batchHash, postStateRoot, withdrawRoot);
  }

  /**********************
   * Internal Functions *
   **********************/

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

  /// @notice Utility function that converts the address in the L1 that submitted a tx to
  /// the inbox to the msg.sender viewed in the L2
  /// @param l1Address the address in the L1 that triggered the tx to L2
  /// @return l2Address L2 address as viewed in msg.sender
  function applyL1ToL2Alias(address l1Address) internal pure returns (address l2Address) {
    uint160 offset = uint160(0x1111000000000000000000000000000000001111);
    unchecked {
      l2Address = address(uint160(l1Address) + offset);
    }
  }

  /// @dev Internal function to load batch header from calldata to memory.
  /// @param _batchHeader The batch header in calldata.
  /// @return memPtr The start memory offset of loaded batch header.
  /// @return _batchHash The hash of the loaded batch header.
  function _loadBatchHeader(bytes calldata _batchHeader) internal pure returns (uint256 memPtr, bytes32 _batchHash) {
    // load to memory
    uint256 _length;
    (memPtr, _length) = BatchHeaderV0Codec.loadAndValidate(_batchHeader);

    uint256 _batchIndex = BatchHeaderV0Codec.batchIndex(memPtr);
    _batchHash = bytes32(_batchIndex + 1);
  }
}
