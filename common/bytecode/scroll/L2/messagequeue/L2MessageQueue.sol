// File: src/libraries/common/AppendOnlyMerkleTree.sol

// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

abstract contract AppendOnlyMerkleTree {
  /// @dev The maximum height of the withdraw merkle tree.
  uint256 private constant MAX_TREE_HEIGHT = 40;

  /// @notice The merkle root of the current merkle tree.
  /// @dev This is actual equal to `branches[n]`.
  bytes32 public messageRoot;

  /// @notice The next unused message index.
  uint256 public nextMessageIndex;

  /// @notice The list of zero hash in each height.
  bytes32[MAX_TREE_HEIGHT] private zeroHashes;

  /// @notice The list of minimum merkle proofs needed to compute next root.
  /// @dev Only first `n` elements are used, where `n` is the minimum value that `2^{n-1} >= currentMaxNonce + 1`.
  /// It means we only use `currentMaxNonce + 1` leaf nodes to construct the merkle tree.
  bytes32[MAX_TREE_HEIGHT] public branches;

  function _initializeMerkleTree() internal {
    // Compute hashes in empty sparse Merkle tree
    for (uint256 height = 0; height + 1 < MAX_TREE_HEIGHT; height++) {
      zeroHashes[height + 1] = _efficientHash(zeroHashes[height], zeroHashes[height]);
    }
  }

  function _appendMessageHash(bytes32 _messageHash) internal returns (uint256, bytes32) {
    uint256 _currentMessageIndex = nextMessageIndex;
    bytes32 _hash = _messageHash;
    uint256 _height = 0;
    // @todo it can be optimized, since we only need the newly added branch.
    while (_currentMessageIndex != 0) {
      if (_currentMessageIndex % 2 == 0) {
        // it may be used in next round.
        branches[_height] = _hash;
        // it's a left child, the right child must be null
        _hash = _efficientHash(_hash, zeroHashes[_height]);
      } else {
        // it's a right child, use previously computed hash
        _hash = _efficientHash(branches[_height], _hash);
      }
      unchecked {
        _height += 1;
      }
      _currentMessageIndex >>= 1;
    }

    branches[_height] = _hash;
    messageRoot = _hash;

    _currentMessageIndex = nextMessageIndex;
    unchecked {
      nextMessageIndex = _currentMessageIndex + 1;
    }

    return (_currentMessageIndex, _hash);
  }

  function _efficientHash(bytes32 a, bytes32 b) private pure returns (bytes32 value) {
    // solhint-disable-next-line no-inline-assembly
    assembly {
      mstore(0x00, a)
      mstore(0x20, b)
      value := keccak256(0x00, 0x40)
    }
  }
}

// File: src/libraries/common/OwnableBase.sol



pragma solidity ^0.8.0;

abstract contract OwnableBase {
  /**********
   * Events *
   **********/

  /// @notice Emitted when owner is changed by current owner.
  /// @param _oldOwner The address of previous owner.
  /// @param _newOwner The address of new owner.
  event OwnershipTransferred(address indexed _oldOwner, address indexed _newOwner);

  /*************
   * Variables *
   *************/

  /// @notice The address of the current owner.
  address public owner;

  /**********************
   * Function Modifiers *
   **********************/

  /// @dev Throws if called by any account other than the owner.
  modifier onlyOwner() {
    require(owner == msg.sender, "caller is not the owner");
    _;
  }

  /************************
   * Restricted Functions *
   ************************/

  /// @notice Leaves the contract without owner. It will not be possible to call
  /// `onlyOwner` functions anymore. Can only be called by the current owner.
  ///
  /// @dev Renouncing ownership will leave the contract without an owner,
  /// thereby removing any functionality that is only available to the owner.
  function renounceOwnership() public onlyOwner {
    _transferOwnership(address(0));
  }

  /// @notice Transfers ownership of the contract to a new account (`newOwner`).
  /// Can only be called by the current owner.
  function transferOwnership(address _newOwner) public onlyOwner {
    require(_newOwner != address(0), "new owner is the zero address");
    _transferOwnership(_newOwner);
  }

  /**********************
   * Internal Functions *
   **********************/

  /// @dev Transfers ownership of the contract to a new account (`newOwner`).
  /// Internal function without access restriction.
  function _transferOwnership(address _newOwner) internal {
    address _oldOwner = owner;
    owner = _newOwner;
    emit OwnershipTransferred(_oldOwner, _newOwner);
  }
}

// File: src/L2/predeploys/L2MessageQueue.sol



pragma solidity ^0.8.0;


/// @title L2MessageQueue
/// @notice The original idea is from Optimism, see [OVM_L2ToL1MessagePasser](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L2/predeploys/OVM_L2ToL1MessagePasser.sol).
/// The L2 to L1 Message Passer is a utility contract which facilitate an L1 proof of the
/// of a message on L2. The L1 Cross Domain Messenger performs this proof in its
/// _verifyStorageProof function, which verifies the existence of the transaction hash in this
/// contract's `sentMessages` mapping.
contract L2MessageQueue is AppendOnlyMerkleTree, OwnableBase {
  /// @notice Emitted when a new message is added to the merkle tree.
  /// @param index The index of the corresponding message.
  /// @param messageHash The hash of the corresponding message.
  event AppendMessage(uint256 index, bytes32 messageHash);

  /// @notice The address of L2ScrollMessenger contract.
  address public messenger;

  constructor(address _owner) {
    _transferOwnership(_owner);
  }

  function initialize() external {
    _initializeMerkleTree();
  }

  /// @notice record the message to merkle tree and compute the new root.
  /// @param _messageHash The hash of the new added message.
  function appendMessage(bytes32 _messageHash) external returns (bytes32) {
    require(msg.sender == messenger, "only messenger");

    (uint256 _currentNonce, bytes32 _currentRoot) = _appendMessageHash(_messageHash);

    // We can use the event to compute the merkle tree locally.
    emit AppendMessage(_currentNonce, _messageHash);

    return _currentRoot;
  }

  /// @notice Update the address of messenger.
  /// @dev You are not allowed to update messenger when there are some messages appended.
  /// @param _messenger The address of messenger to update.
  function updateMessenger(address _messenger) external onlyOwner {
    require(nextMessageIndex == 0, "cannot update messenger");

    messenger = _messenger;
  }
}
