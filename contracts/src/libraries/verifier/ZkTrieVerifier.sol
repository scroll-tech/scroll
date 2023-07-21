// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

interface PoseidonUnit2 {
    function poseidon(uint256[2] memory) external view returns (uint256);
}

library ZkTrieVerifier {
    /// @notice Internal function to validates a proof from eth_getProof.
    /// @param poseidon The address of poseidon hash contract.
    /// @param account The address of the contract.
    /// @param storageKey The storage slot to verify.
    /// @param proof The rlp encoding result of eth_getProof.
    /// @return stateRoot The computed state root. Must be checked by the caller.
    /// @return storageValue The value of `storageKey`.
    ///
    /// @dev The code is based on
    /// 1. https://github.com/scroll-tech/go-ethereum/blob/staging/trie/zk_trie.go#L176
    /// 2. https://github.com/scroll-tech/zktrie/blob/main/trie/zk_trie_proof.go#L30
    ///
    /// The encoding order of `proof` is
    /// ```text
    /// |        1 byte        |      ...      |        1 byte        |      ...      |
    /// | account proof length | account proof | storage proof length | storage proof |
    /// ```
    function verifyZkTrieProof(
        address poseidon,
        address account,
        bytes32 storageKey,
        bytes calldata proof
    ) internal view returns (bytes32 stateRoot, bytes32 storageValue) {
        assembly {
            // reverts with error `msg`.
            // make sure the length of error string <= 32
            function revertWith(msg) {
                // keccak("Error(string)")
                mstore(0x00, 0x08c379a000000000000000000000000000000000000000000000000000000000)
                mstore(0x04, 0x20) // str.offset
                mstore(0x44, msg)
                let msgLen
                for {

                } msg {

                } {
                    msg := shl(8, msg)
                    msgLen := add(msgLen, 1)
                }
                mstore(0x24, msgLen) // str.length
                revert(0x00, 0x64)
            }
            // reverts with `msg` when condition is not matched.
            // make sure the length of error string <= 32
            function require(cond, msg) {
                if iszero(cond) {
                    revertWith(msg)
                }
            }
            // compute poseidon hash of two uint256
            function poseidon_hash(hasher, v0, v1) -> r {
                let x := mload(0x40)
                // keccack256("poseidon(uint256[2])")
                mstore(x, 0x29a5f2f600000000000000000000000000000000000000000000000000000000)
                mstore(add(x, 0x04), v0)
                mstore(add(x, 0x24), v1)
                let success := staticcall(gas(), hasher, x, 0x44, 0x20, 0x20)
                require(success, "poseidon hash failed")
                r := mload(0x20)
            }
            // compute poseidon hash of 1 uint256
            function hash_uint256(hasher, v) -> r {
                r := poseidon_hash(hasher, shr(128, v), and(v, 0xffffffffffffffffffffffffffffffff))
            }

            // traverses the tree from the root to the node before the leaf.
            // based on https://github.com/ethereum/go-ethereum/blob/master/trie/proof.go#L114
            function walkTree(hasher, key, _ptr) -> ptr, rootHash, expectedHash {
                ptr := _ptr

                // the first byte is the number of nodes + 1
                let nodes := sub(byte(0, calldataload(ptr)), 1)
                ptr := add(ptr, 1)

                // treat the leaf node with different logic
                for {
                    let depth := 1
                } lt(depth, nodes) {
                    depth := add(depth, 1)
                } {
                    // must be a parent node with two children
                    let nodeType := byte(0, calldataload(ptr))
                    ptr := add(ptr, 1)
                    require(eq(nodeType, 0), "Invalid parent node")

                    // load left/right child hash
                    let childHashL := calldataload(ptr)
                    ptr := add(ptr, 0x20)
                    let childHashR := calldataload(ptr)
                    ptr := add(ptr, 0x20)
                    let hash := poseidon_hash(hasher, childHashL, childHashR)

                    // first item is considered the root node.
                    // Otherwise verifies that the hash of the current node
                    // is the same as the previous choosen one.
                    switch depth
                    case 1 {
                        rootHash := hash
                    }
                    default {
                        require(eq(hash, expectedHash), "Hash mismatch")
                    }

                    // decide which path to walk based on key
                    switch and(key, 1)
                    case 0 {
                        expectedHash := childHashL
                    }
                    default {
                        expectedHash := childHashR
                    }
                    key := shr(1, key)
                }
            }

            function checkProofMagicBytes(hasher, _ptr) -> ptr {
                ptr := _ptr
                let x := mload(0x40)
                calldatacopy(x, ptr, 0x2d)
                x := keccak256(x, 0x2d)
                require(
                    eq(x, 0x950654da67865a81bc70e45f3230f5179f08e29c66184bf746f71050f117b3b8),
                    "Invalid ProofMagicBytes"
                )
                ptr := add(ptr, 0x2d) // skip ProofMagicBytes
            }

            // shared variable names
            let storageHash
            // starting point
            let ptr := proof.offset

            // verify account proof
            {
                let leafHash
                let key := hash_uint256(poseidon, shl(96, account))

                // `stateRoot` is a return value and must be checked by the caller
                ptr, stateRoot, leafHash := walkTree(poseidon, key, ptr)

                require(eq(1, byte(0, calldataload(ptr))), "Invalid leaf node")
                ptr := add(ptr, 0x01) // skip NodeType
                require(eq(calldataload(ptr), key), "Node key mismatch")
                ptr := add(ptr, 0x20) // skip NodeKey
                {
                    let valuePreimageLength := and(shr(224, calldataload(ptr)), 0xffff)
                    // @todo check CompressedFlag
                    ptr := add(ptr, 0x04) // skip CompressedFlag
                    ptr := add(ptr, valuePreimageLength) // skip ValuePreimage
                }

                // compute value hash for State Account Leaf Node
                {
                    let tmpHash1 := calldataload(ptr)
                    ptr := add(ptr, 0x20) // skip nonce/codesize/0
                    tmpHash1 := poseidon_hash(poseidon, tmpHash1, calldataload(ptr))
                    ptr := add(ptr, 0x20) // skip balance
                    storageHash := calldataload(ptr)
                    ptr := add(ptr, 0x20) // skip StorageRoot
                    let tmpHash2 := hash_uint256(poseidon, calldataload(ptr))
                    ptr := add(ptr, 0x20) // skip KeccakCodeHash
                    tmpHash2 := poseidon_hash(poseidon, storageHash, tmpHash2)
                    tmpHash2 := poseidon_hash(poseidon, tmpHash1, tmpHash2)
                    tmpHash2 := poseidon_hash(poseidon, tmpHash2, calldataload(ptr))
                    ptr := add(ptr, 0x20) // skip PoseidonCodeHash

                    tmpHash1 := poseidon_hash(poseidon, 1, key)
                    tmpHash1 := poseidon_hash(poseidon, tmpHash1, tmpHash2)

                    require(eq(leafHash, tmpHash1), "Invalid leaf node hash")
                }

                require(eq(0x20, byte(0, calldataload(ptr))), "Invalid KeyPreimage length")
                ptr := add(ptr, 0x01) // skip KeyPreimage length
                require(eq(shl(96, account), calldataload(ptr)), "Invalid KeyPreimage")
                ptr := add(ptr, 0x20) // skip KeyPreimage

                // compare ProofMagicBytes
                ptr := checkProofMagicBytes(poseidon, ptr)
            }

            // verify storage proof
            {
                let leafHash
                let key := hash_uint256(poseidon, storageKey)
                {
                    let rootHash
                    ptr, rootHash, leafHash := walkTree(poseidon, key, ptr)

                    switch rootHash
                    case 0 {
                        // in the case that the leaf is the only element, then
                        // the hash of the leaf must match the value from the account leaf
                        require(eq(leafHash, storageHash), "Storage root mismatch")
                    }
                    default {
                        // otherwise the root hash of the storage tree
                        // must match the value from the account leaf
                        require(eq(rootHash, storageHash), "Storage root mismatch")
                    }
                }

                switch byte(0, calldataload(ptr))
                case 1 {
                    ptr := add(ptr, 0x01) // skip NodeType
                    require(eq(calldataload(ptr), key), "Node key mismatch")
                    ptr := add(ptr, 0x20) // skip NodeKey
                    {
                        let valuePreimageLength := and(shr(224, calldataload(ptr)), 0xffff)
                        // @todo check CompressedFlag
                        ptr := add(ptr, 0x04) // skip CompressedFlag
                        ptr := add(ptr, valuePreimageLength) // skip ValuePreimage
                    }

                    storageValue := calldataload(ptr)
                    ptr := add(ptr, 0x20) // skip StorageValue

                    mstore(0x00, hash_uint256(poseidon, storageValue))
                    key := poseidon_hash(poseidon, 1, key)
                    mstore(0x00, poseidon_hash(poseidon, key, mload(0x00)))
                    require(eq(leafHash, mload(0x00)), "Invalid leaf node hash")

                    require(eq(0x20, byte(0, calldataload(ptr))), "Invalid KeyPreimage length")
                    ptr := add(ptr, 0x01) // skip KeyPreimage length
                    require(eq(storageKey, calldataload(ptr)), "Invalid KeyPreimage")
                    ptr := add(ptr, 0x20) // skip KeyPreimage
                }
                case 2 {
                    ptr := add(ptr, 0x01) // skip NodeType
                    require(eq(leafHash, 0), "Invalid empty node hash")
                }
                default {
                    revertWith("Invalid leaf node")
                }

                // compare ProofMagicBytes
                ptr := checkProofMagicBytes(poseidon, ptr)
            }

            // the one and only boundary check
            // in case an attacker crafted a malicous payload
            // and succeeds in the prior verification steps
            // then this should catch any bogus accesses
            if iszero(eq(ptr, add(proof.offset, proof.length))) {
                revertWith("Proof length mismatch")
            }
        }
    }
}
