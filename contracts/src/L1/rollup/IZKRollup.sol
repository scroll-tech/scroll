// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IZKRollup {
  /**************************************** Events ****************************************/

  event CommitBlock(bytes32 indexed _blockHash, uint64 indexed _blockHeight, bytes32 _parentHash);

  event RevertBlock(bytes32 indexed _blockHash);

  event FinalizeBlock(bytes32 indexed _blockHash, uint64 indexed _blockHeight);

  /// @dev The transanction struct
  struct Layer2Transaction {
    address caller;
    uint64 nonce;
    address target;
    uint64 gas;
    uint256 gasPrice;
    uint256 value;
    bytes data;
  }

  /// @dev The block header struct
  struct BlockHeader {
    bytes32 blockHash;
    bytes32 parentHash;
    uint256 baseFee;
    bytes32 stateRoot;
    uint64 blockHeight;
    uint64 gasUsed;
    uint64 timestamp;
    bytes extraData;
  }

  /**************************************** View Functions ****************************************/

  /// @notice Return the message hash by index.
  /// @param _index The index to query.
  function getMessageHashByIndex(uint256 _index) external view returns (bytes32);

  /// @notice Return the index of the first queue element not yet executed.
  function getNextQueueIndex() external view returns (uint256);

  /// @notice Return the layer 2 block gas limit.
  /// @param _blockNumber The block number to query
  function layer2GasLimit(uint256 _blockNumber) external view returns (uint256);

  /// @notice Verify a state proof for message relay.
  /// @dev add more fields.
  function verifyMessageStateProof(uint256 _blockNumber) external view returns (bool);

  /**************************************** Mutated Functions ****************************************/

  /// @notice Append a cross chain message to message queue.
  /// @dev This function should only be called by L1ScrollMessenger for safety.
  /// @param _sender The address of message sender in layer 1.
  /// @param _target The address of message recipient in layer 2.
  /// @param _value The amount of ether sent to recipient in layer 2.
  /// @param _fee The amount of ether paid to relayer in layer 2.
  /// @param _deadline The deadline of the message.
  /// @param _message The content of the message.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function appendMessage(
    address _sender,
    address _target,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    bytes memory _message,
    uint256 _gasLimit
  ) external returns (uint256);

  /// @notice commit block in layer 1
  /// @dev will add more parameters if needed.
  /// @param _header The block header.
  /// @param _txns The transactions included in the block.
  function commitBlock(BlockHeader memory _header, Layer2Transaction[] memory _txns) external;

  /// @notice revert a pending block.
  /// @dev one can only revert unfinalized blocks.
  /// @param _blockHash The block hash of the block.
  function revertBlock(bytes32 _blockHash) external;

  /// @notice finalize commited block in layer 1
  /// @dev will add more parameters if needed.
  /// @param _blockHash The block hash of the commited block.
  /// @param _proof The corresponding proof of the commited block.
  /// @param _instances Instance used to verify, generated from block.
  function finalizeBlockWithProof(
    bytes32 _blockHash,
    uint256[] memory _proof,
    uint256[] memory _instances
  ) external;
}
