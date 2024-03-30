// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

// solhint-disable no-inline-assembly

library WithdrawTrieVerifier {
    /// @dev Verify the merkle proof given root, leaf node and proof.
    ///
    /// Vulnerability:
    ///   The initially provided message hash can be hashed with the first hash of the proof,
    ///   thereby giving an intermediate node of the trie. This can then be used with a shortened
    ///   proof to pass the verification, which may lead to replayability.
    ///
    ///   However, it is designed to verify the withdraw trie in `L2MessageQueue`. The `_hash` given
    ///   in the parameter is always a leaf node. So we assume the length of proof is correct and
    ///   cannot be shortened.
    /// @param _root The expected root node hash of the withdraw trie.
    /// @param _hash The leaf node hash of the withdraw trie.
    /// @param _nonce The index of the leaf node from left to right, starting from 0.
    /// @param _proof The concatenated merkle proof verified the leaf node.
    function verifyMerkleProof(
        bytes32 _root,
        bytes32 _hash,
        uint256 _nonce,
        bytes memory _proof
    ) internal pure returns (bool) {
        require(_proof.length % 32 == 0, "Invalid proof");
        uint256 _length = _proof.length / 32;

        for (uint256 i = 0; i < _length; i++) {
            bytes32 item;
            assembly {
                item := mload(add(add(_proof, 0x20), mul(i, 0x20)))
            }
            if (_nonce % 2 == 0) {
                _hash = _efficientHash(_hash, item);
            } else {
                _hash = _efficientHash(item, _hash);
            }
            _nonce /= 2;
        }
        return _hash == _root;
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
