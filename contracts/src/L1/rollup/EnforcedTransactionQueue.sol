// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import { IEnforcedTransactionQueue } from "./IEnforcedTransactionQueue.sol";

contract EnforcedTransactionQueue is OwnableUpgradeable, IEnforcedTransactionQueue {
  /**********
   * Events *
   **********/

  /*************
   * Constants *
   *************/

  /// @dev The number of seconds can wait until the transaction is confirmed in L2.
  uint256 private constant FORCE_INCLUSION_DELAY = 1 days;

  /***********
   * Structs *
   ***********/

  struct Transaction {
    bytes32 hash;
    uint256 deadline;
  }

  /*************
   * Variables *
   *************/

  /// @notice The address of ZK Rollup contract.
  address public rollup;

  /// @notice The list of enforced transaction.
  Transaction[] public transactionQueue;

  /// @notice The index of the earliest unincluded enforced transaction.
  uint256 public nextUnincluedIndex;

  /***************
   * Constructor *
   ***************/

  function initialize(address _rollup) external initializer {
    OwnableUpgradeable.__Ownable_init();

    rollup = _rollup;
  }

  /*************************
   * Public View Functions *
   *************************/

  /// @inheritdoc IEnforcedTransactionQueue
  function isTransactionExpired(
    uint256 _l2Timestamp,
    uint256 _index,
    bytes32 _transactionHash
  ) external view override returns (bool) {
    return
      // transaction already included
      _index < nextUnincluedIndex ||
      // transaction hash mismatch
      transactionQueue[_index].hash != _transactionHash ||
      // transaction expired
      transactionQueue[_index].deadline < _l2Timestamp;
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @inheritdoc IEnforcedTransactionQueue
  function enqueueTransaction(bytes calldata _rawTx) external payable override {
    _validateRawTransaction(_rawTx);
    // @todo prevent spam attacks

    bytes32 _hash = keccak256(_rawTx);
    transactionQueue.push(Transaction(_hash, block.timestamp + FORCE_INCLUSION_DELAY));

    emit EnqueueTransaction(_hash, _rawTx);
  }

  /// @inheritdoc IEnforcedTransactionQueue
  function includeTransaction(uint256 _nextIndex) external override {
    require(msg.sender == rollup, "sender not rollup");

    uint256 _nextUnincluedIndex = nextUnincluedIndex;
    require(_nextIndex > nextUnincluedIndex, "index too small");

    nextUnincluedIndex = _nextIndex;

    emit IncludeTransaction(_nextUnincluedIndex, _nextIndex);
  }

  /************************
   * Restricted Functions *
   ************************/

  /// @notice Force include enforced transaction in L2 by owner
  /// @param _nextIndex The next unincluded transaction index.
  function forceIncludeTransaction(uint256 _nextIndex) external onlyOwner {
    uint256 _nextUnincluedIndex = nextUnincluedIndex;
    require(_nextIndex > _nextUnincluedIndex, "index too small");

    nextUnincluedIndex = _nextIndex;

    emit ForceIncludeTransaction(_nextUnincluedIndex, _nextIndex);
  }

  /**********************
   * Internal Functions *
   **********************/

  /// @dev Internal function to validate the transaction RLP encoding.
  /// @param _rawTx The RLP encoding of the enforced transation.
  function _validateRawTransaction(bytes calldata _rawTx) internal view {
    // @todo finish logic
  }
}
