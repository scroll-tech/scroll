// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

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
        require(zeroHashes[1] != bytes32(0), "call before initialization");

        uint256 _currentMessageIndex = nextMessageIndex;
        bytes32 _hash = _messageHash;
        uint256 _height = 0;

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
