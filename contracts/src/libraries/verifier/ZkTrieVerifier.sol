// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

library ZkTrieVerifier {
  function verifyMerkleProof(
    bytes32 _root,
    bytes32 _hash,
    uint256 _nonce,
    bytes32[] memory _proofs
  ) internal pure returns (bool) {
    for (uint256 i = 0; i < _proofs.length; i++) {
      if (_nonce % 2 == 0) {
        _hash = _efficientHash(_hash, _proofs[i]);
      } else {
        _hash = _efficientHash(_proofs[i], _hash);
      }
      _nonce /= 2;
    }
    // _root = 0 means we don't want to verify.
    return _root == 0 || _hash == _root;
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
