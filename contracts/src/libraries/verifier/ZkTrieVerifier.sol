// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

// solhint-disable no-inline-assembly

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
    ///
    /// Possible attack vector:
    ///   + Malicious users can influence how many levels the proof must go through by predicting addresses
    ///     (or storage slots) that would branch the Trie until a certain depth. Even though artificially 
    ///     increasing the proof's depth of a certain account or storage will not cause a DoS scenario, since
    ///     the depth can still reach the maximum depth size in the worst-case scenario, artificially increasing
    ///     the proof's depth will increase the number of iterations the `walkTree` method has to perform in order
    ///     to reach the respective leaf. If protocols that use this verifier limit the gas used on-chain to perform
    ///     such a verification (to a reasonable value), then a malicious user might be able to increase it for a
    ///     particular transaction by reaching a similar hashed key to a certain depth in the Trie.
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
            function poseidonHash(hasher, v0, v1, domain) -> r {
                let x := mload(0x40)
                // keccack256("poseidon(uint256[2],uint256)")
                mstore(x, 0xa717016c00000000000000000000000000000000000000000000000000000000)
                mstore(add(x, 0x04), v0)
                mstore(add(x, 0x24), v1)
                mstore(add(x, 0x44), domain)
                let success := staticcall(gas(), hasher, x, 0x64, 0x20, 0x20)
                require(success, "poseidon hash failed")
                r := mload(0x20)
            }
            // compute poseidon hash of 1 uint256
            function hashUint256(hasher, v) -> r {
                r := poseidonHash(hasher, shr(128, v), and(v, 0xffffffffffffffffffffffffffffffff), 512)
            }

            // traverses the tree from the root to the node before the leaf.
            // based on https://github.com/ethereum/go-ethereum/blob/master/trie/proof.go#L114
            function walkTree(hasher, key, _ptr) -> ptr, rootHash, expectedHash {
                ptr := _ptr

                // the first byte is the number of nodes + 1
                let nodes := sub(byte(0, calldataload(ptr)), 1)
                require(lt(nodes, 249), "InvalidNodeDepth")
                ptr := add(ptr, 0x01)

                // treat the leaf node with different logic
                for {
                    let depth := 1
                } lt(depth, nodes) {
                    depth := add(depth, 1)
                } {
                    // must be a parent node with two children
                    let nodeType := byte(0, calldataload(ptr))
                    // 6 <= nodeType && nodeType < 10
                    require(lt(sub(nodeType, 6), 4), "InvalidBranchNodeType")
                    ptr := add(ptr, 0x01)

                    // load left/right child hash
                    let childHashL := calldataload(ptr)
                    ptr := add(ptr, 0x20)
                    let childHashR := calldataload(ptr)
                    ptr := add(ptr, 0x20)
                    let hash := poseidonHash(hasher, childHashL, childHashR, nodeType)

                    // first item is considered the root node.
                    // Otherwise verifies that the hash of the current node
                    // is the same as the previous choosen one.
                    switch depth
                    case 1 {
                        rootHash := hash
                    }
                    default {
                        require(eq(hash, expectedHash), "BranchHashMismatch")
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
                    "InvalidProofMagicBytes"
                )
                ptr := add(ptr, 0x2d) // skip ProofMagicBytes
            }

            function verifyAccountProof(hasher, _account, _ptr) -> ptr, storageRootHash, _stateRoot {
                ptr := _ptr

                let leafHash
                let key := hashUint256(hasher, shl(96, _account))

                // `stateRoot` is a return value and must be checked by the caller
                ptr, _stateRoot, leafHash := walkTree(hasher, key, ptr)

                switch byte(0, calldataload(ptr))
                case 4 {
                    // nonempty leaf node
                    ptr := add(ptr, 0x01) // skip NodeType
                    require(eq(calldataload(ptr), key), "AccountKeyMismatch")
                    ptr := add(ptr, 0x20) // skip NodeKey
                    require(eq(shr(224, calldataload(ptr)), 0x05080000), "InvalidAccountCompressedFlag")
                    ptr := add(ptr, 0x04) // skip CompressedFlag

                    // compute value hash for State Account Leaf Node, details can be found in
                    // https://github.com/scroll-tech/mpt-circuit/blob/v0.7/spec/mpt-proof.md#account-segmenttypes
                    // [nonce||codesize||0, balance, storage_root, keccak codehash, poseidon codehash]
                    mstore(0x00, calldataload(ptr))
                    ptr := add(ptr, 0x20) // skip nonce||codesize||0
                    mstore(0x00, poseidonHash(hasher, mload(0x00), calldataload(ptr), 1280))
                    ptr := add(ptr, 0x20) // skip balance
                    storageRootHash := calldataload(ptr)
                    ptr := add(ptr, 0x20) // skip StorageRoot
                    let tmpHash := hashUint256(hasher, calldataload(ptr))
                    ptr := add(ptr, 0x20) // skip KeccakCodeHash
                    tmpHash := poseidonHash(hasher, storageRootHash, tmpHash, 1280)
                    tmpHash := poseidonHash(hasher, mload(0x00), tmpHash, 1280)
                    tmpHash := poseidonHash(hasher, tmpHash, calldataload(ptr), 1280)
                    ptr := add(ptr, 0x20) // skip PoseidonCodeHash

                    tmpHash := poseidonHash(hasher, key, tmpHash, 4)
                    require(eq(leafHash, tmpHash), "InvalidAccountLeafNodeHash")

                    require(eq(0x20, byte(0, calldataload(ptr))), "InvalidAccountKeyPreimageLength")
                    ptr := add(ptr, 0x01) // skip KeyPreimage length
                    require(eq(shl(96, _account), calldataload(ptr)), "InvalidAccountKeyPreimage")
                    ptr := add(ptr, 0x20) // skip KeyPreimage
                }
                case 5 {
                    ptr := add(ptr, 0x01) // skip NodeType
                }
                default {
                    revertWith("InvalidAccountLeafNodeType")
                }

                // compare ProofMagicBytes
                ptr := checkProofMagicBytes(hasher, ptr)
            }

            function verifyStorageProof(hasher, _storageKey, storageRootHash, _ptr) -> ptr, _storageValue {
                ptr := _ptr

                let leafHash
                let key := hashUint256(hasher, _storageKey)
                let rootHash
                ptr, rootHash, leafHash := walkTree(hasher, key, ptr)

                // The root hash of the storage tree must match the value from the account leaf.
                // But when the leaf node is the same as the root node, the function `walkTree` will return
                // `rootHash=0` and `leafHash=0`. In such case, we don't need to check the value of `rootHash`.
                // And the value of `leafHash` should be the same as `storageRootHash`.
                switch rootHash
                case 0 {
                    leafHash := storageRootHash
                }
                default {
                    require(eq(rootHash, storageRootHash), "StorageRootMismatch")
                }

                switch byte(0, calldataload(ptr))
                case 4 {
                    ptr := add(ptr, 0x01) // skip NodeType
                    require(eq(calldataload(ptr), key), "StorageKeyMismatch")
                    ptr := add(ptr, 0x20) // skip NodeKey
                    require(eq(shr(224, calldataload(ptr)), 0x01010000), "InvalidStorageCompressedFlag")
                    ptr := add(ptr, 0x04) // skip CompressedFlag
                    _storageValue := calldataload(ptr)
                    ptr := add(ptr, 0x20) // skip StorageValue

                    // compute leaf node hash and compare, details can be found in
                    // https://github.com/scroll-tech/mpt-circuit/blob/v0.7/spec/mpt-proof.md#storage-segmenttypes
                    mstore(0x00, hashUint256(hasher, _storageValue))
                    mstore(0x00, poseidonHash(hasher, key, mload(0x00), 4))
                    require(eq(leafHash, mload(0x00)), "InvalidStorageLeafNodeHash")

                    require(eq(0x20, byte(0, calldataload(ptr))), "InvalidStorageKeyPreimageLength")
                    ptr := add(ptr, 0x01) // skip KeyPreimage length
                    require(eq(_storageKey, calldataload(ptr)), "InvalidStorageKeyPreimage")
                    ptr := add(ptr, 0x20) // skip KeyPreimage
                }
                case 5 {
                    ptr := add(ptr, 0x01) // skip NodeType
                    require(eq(leafHash, 0), "InvalidStorageEmptyLeafNodeHash")
                }
                default {
                    revertWith("InvalidStorageLeafNodeType")
                }

                // compare ProofMagicBytes
                ptr := checkProofMagicBytes(hasher, ptr)
            }

            let storageRootHash
            let ptr := proof.offset

            // check the correctness of account proof
            ptr, storageRootHash, stateRoot := verifyAccountProof(poseidon, account, ptr)

            // check the correctness of storage proof
            ptr, storageValue := verifyStorageProof(poseidon, storageKey, storageRootHash, ptr)

            // the one and only boundary check
            // in case an attacker crafted a malicous payload
            // and succeeds in the prior verification steps
            // then this should catch any bogus accesses
            if iszero(eq(ptr, add(proof.offset, proof.length))) {
                revertWith("ProofLengthMismatch")
            }
        }
    }
}
