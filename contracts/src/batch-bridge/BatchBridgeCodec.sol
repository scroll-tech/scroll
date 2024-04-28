// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

library BatchBridgeCodec {
    /// @dev Encode the `token` and `batchIndex` to single `bytes32`.
    function encodeInitialNode(address token, uint64 batchIndex) internal pure returns (bytes32 node) {
        assembly {
            node := add(shl(96, token), batchIndex)
        }
    }

    /// @dev Encode the `sender` and `amount` to single `bytes32`.
    function encodeNode(address sender, uint96 amount) internal pure returns (bytes32 node) {
        assembly {
            node := add(shl(96, sender), amount)
        }
    }

    /// @dev Decode `bytes32` `node` to `receiver` and `amount`.
    function decodeNode(bytes32 node) internal pure returns (address receiver, uint256 amount) {
        receiver = address(uint160(uint256(node) >> 96));
        amount = uint256(node) & 0xffffffffffffffffffffffff;
    }

    /// @dev Compute `keccak256(concat(a, b))`.
    function hash(bytes32 a, bytes32 b) internal pure returns (bytes32 value) {
        // solhint-disable-next-line no-inline-assembly
        assembly {
            mstore(0x00, a)
            mstore(0x20, b)
            value := keccak256(0x00, 0x40)
        }
    }
}
