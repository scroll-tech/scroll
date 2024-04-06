// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

library PatriciaMerkleTrieVerifier {
    /// @notice Internal function to validates a proof from eth_getProof.
    /// @param account The address of the contract.
    /// @param storageKey The storage slot to verify.
    /// @param proof The rlp encoding result of eth_getProof.
    /// @return stateRoot The computed state root. Must be checked by the caller.
    /// @return storageValue The value of `storageKey`.
    ///
    /// @dev The code is based on
    /// 1. https://eips.ethereum.org/EIPS/eip-1186
    /// 2. https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/
    /// 3. https://github.com/ethereum/go-ethereum/blob/master/trie/proof.go#L114
    /// 4. https://github.com/privacy-scaling-explorations/zkevm-chain/blob/master/contracts/templates/PatriciaValidator.sol
    ///
    /// The encoding order of `proof` is
    /// ```text
    /// |        1 byte        |      ...      |        1 byte        |      ...      |
    /// | account proof length | account proof | storage proof length | storage proof |
    /// ```
    function verifyPatriciaProof(
        address account,
        bytes32 storageKey,
        bytes calldata proof
    ) internal pure returns (bytes32 stateRoot, bytes32 storageValue) {
        assembly {
            // hashes 32 bytes of `v`
            function keccak_32(v) -> r {
                mstore(0x00, v)
                r := keccak256(0x00, 0x20)
            }
            // hashes the last 20 bytes of `v`
            function keccak_20(v) -> r {
                mstore(0x00, v)
                r := keccak256(0x0c, 0x14)
            }
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

            // special function for decoding the storage value
            // because of the prefix truncation if value > 31 bytes
            // see `loadValue`
            function decodeItem(word, len) -> ret {
                // default
                ret := word

                // RLP single byte
                if lt(word, 0x80) {
                    leave
                }

                // truncated
                if gt(len, 32) {
                    leave
                }

                // value is >= 0x80 and <= 32 bytes.
                // `len` should be at least 2 (prefix byte + value)
                // otherwise the RLP is malformed.
                let bits := mul(len, 8)
                // sub 8 bits - the prefix
                bits := sub(bits, 8)
                let mask := shl(bits, 0xff)
                // invert the mask
                mask := not(mask)
                // should hold the value - prefix byte
                ret := and(ret, mask)
            }

            // returns the `len` of the whole RLP list at `ptr`
            // and the offset for the first value inside the list.
            function decodeListLength(ptr) -> len, startOffset {
                let b0 := byte(0, calldataload(ptr))
                // In most cases, it is a long list. So we reorder the branch to reduce branch prediction miss.

                // 0xf8 - 0xff, long list, length > 55
                if gt(b0, 0xf7) {
                    // the RLP encoding consists of a single byte with value 0xf7
                    // plus the length in bytes of the length of the payload in binary form,
                    // followed by the length of the payload, followed by the concatenation
                    // of the RLP encodings of the items.
                    // the extended length is ignored
                    let lengthBytes := sub(b0, 0xf7)
                    if gt(lengthBytes, 32) {
                        invalid()
                    }

                    // load the extended length
                    startOffset := add(ptr, 1)
                    let extendedLen := calldataload(startOffset)
                    let bits := sub(256, mul(lengthBytes, 8))
                    extendedLen := shr(bits, extendedLen)

                    len := add(extendedLen, lengthBytes)
                    len := add(len, 1)
                    startOffset := add(startOffset, lengthBytes)
                    leave
                }
                // 0xc0 - 0xf7, short list, length <= 55
                if gt(b0, 0xbf) {
                    // the RLP encoding consists of a single byte with value 0xc0
                    // plus the length of the list followed by the concatenation of
                    // the RLP encodings of the items.
                    len := sub(b0, 0xbf)
                    startOffset := add(ptr, 1)
                    leave
                }
                revertWith("Not list")
            }

            // returns the kind, calldata offset of the value and the length in bytes
            // for the RLP encoded data item at `ptr`. used in `decodeFlat`
            // kind = 0 means string/bytes, kind = 1 means list.
            function decodeValue(ptr) -> kind, dataLen, valueOffset {
                let b0 := byte(0, calldataload(ptr))

                // 0x00 - 0x7f, single byte
                if lt(b0, 0x80) {
                    // for a single byte whose value is in the [0x00, 0x7f] range,
                    // that byte is its own RLP encoding.
                    dataLen := 1
                    valueOffset := ptr
                    leave
                }

                // 0x80 - 0xb7, short string/bytes, length <= 55
                if lt(b0, 0xb8) {
                    // the RLP encoding consists of a single byte with value 0x80
                    // plus the length of the string followed by the string.
                    dataLen := sub(b0, 0x80)
                    valueOffset := add(ptr, 1)
                    leave
                }

                // 0xb8 - 0xbf, long string/bytes, length > 55
                if lt(b0, 0xc0) {
                    // the RLP encoding consists of a single byte with value 0xb7
                    // plus the length in bytes of the length of the string in binary form,
                    // followed by the length of the string, followed by the string.
                    let lengthBytes := sub(b0, 0xb7)
                    if gt(lengthBytes, 4) {
                        invalid()
                    }

                    // load the extended length
                    valueOffset := add(ptr, 1)
                    let extendedLen := calldataload(valueOffset)
                    let bits := sub(256, mul(lengthBytes, 8))
                    extendedLen := shr(bits, extendedLen)

                    dataLen := extendedLen
                    valueOffset := add(valueOffset, lengthBytes)
                    leave
                }

                kind := 1
                // 0xc0 - 0xf7, short list, length <= 55
                if lt(b0, 0xf8) {
                    // intentionally ignored
                    // dataLen := sub(firstByte, 0xc0)
                    valueOffset := add(ptr, 1)
                    leave
                }

                // 0xf8 - 0xff, long list, length > 55
                {
                    // the extended length is ignored
                    dataLen := sub(b0, 0xf7)
                    valueOffset := add(ptr, 1)
                    leave
                }
            }

            // decodes all RLP encoded data and stores their DATA items
            // [length - 128 bits | calldata offset - 128 bits] in a continuous memory region.
            // Expects that the RLP starts with a list that defines the length
            // of the whole RLP region.
            function decodeFlat(_ptr) -> ptr, memStart, nItems, hash {
                ptr := _ptr

                // load free memory ptr
                // doesn't update the ptr and leaves the memory region dirty
                memStart := mload(0x40)

                let payloadLen, startOffset := decodeListLength(ptr)
                // reuse memStart region and hash
                calldatacopy(memStart, ptr, payloadLen)
                hash := keccak256(memStart, payloadLen)

                let memPtr := memStart
                let ptrStop := add(ptr, payloadLen)
                ptr := startOffset

                // decode until the end of the list
                for {

                } lt(ptr, ptrStop) {

                } {
                    let kind, len, valuePtr := decodeValue(ptr)
                    ptr := add(len, valuePtr)

                    if iszero(kind) {
                        // store the length of the data and the calldata offset
                        // low -------> high
                        // |     128 bits    |   128 bits   |
                        // | calldata offset | value length |
                        mstore(memPtr, or(shl(128, len), valuePtr))
                        memPtr := add(memPtr, 0x20)
                    }
                }

                if iszero(eq(ptr, ptrStop)) {
                    invalid()
                }

                nItems := div(sub(memPtr, memStart), 32)
            }

            // prefix gets truncated to 256 bits
            // `depth` is untrusted and can lead to bogus
            // shifts/masks. In that case, the remaining verification
            // steps must fail or lead to an invalid stateRoot hash
            // if the proof data is 'spoofed but valid'
            function derivePath(key, depth) -> path {
                path := key

                let bits := mul(depth, 4)
                {
                    let mask := not(0)
                    mask := shr(bits, mask)
                    path := and(path, mask)
                }

                // even prefix
                let prefix := 0x20
                if mod(depth, 2) {
                    // odd
                    prefix := 0x3
                }

                // the prefix may be shifted outside bounds
                // this is intended, see `loadValue`
                bits := sub(256, bits)
                prefix := shl(bits, prefix)
                path := or(prefix, path)
            }

            // loads and aligns a value from calldata
            // given the `len|offset` stored at `memPtr`
            function loadValue(memPtr, idx) -> value {
                let tmp := mload(add(memPtr, mul(32, idx)))
                // assuming 0xffffff is sufficient for storing calldata offset
                let offset := and(tmp, 0xffffff)
                let len := shr(128, tmp)

                if gt(len, 31) {
                    // special case - truncating the value is intended.
                    // this matches the behavior in `derivePath` that truncates to 256 bits.
                    offset := add(offset, sub(len, 32))
                    value := calldataload(offset)
                    leave
                }

                // everything else is
                // < 32 bytes - align the value
                let bits := mul(sub(32, len), 8)
                value := calldataload(offset)
                value := shr(bits, value)
            }

            // loads and aligns a value from calldata
            // given the `len|offset` stored at `memPtr`
            // Same as `loadValue` except it returns also the size
            // of the value.
            function loadValueLen(memPtr, idx) -> value, len {
                let tmp := mload(add(memPtr, mul(32, idx)))
                // assuming 0xffffff is sufficient for storing calldata offset
                let offset := and(tmp, 0xffffff)
                len := shr(128, tmp)

                if gt(len, 31) {
                    // special case - truncating the value is intended.
                    // this matches the behavior in `derivePath` that truncates to 256 bits.
                    offset := add(offset, sub(len, 32))
                    value := calldataload(offset)
                    leave
                }

                // everything else is
                // < 32 bytes - align the value
                let bits := mul(sub(32, len), 8)
                value := calldataload(offset)
                value := shr(bits, value)
            }

            function loadPair(memPtr, idx) -> offset, len {
                let tmp := mload(add(memPtr, mul(32, idx)))
                // assuming 0xffffff is sufficient for storing calldata offset
                offset := and(tmp, 0xffffff)
                len := shr(128, tmp)
            }

            // decodes RLP at `_ptr`.
            // reverts if the number of DATA items doesn't match `nValues`.
            // returns the RLP data items at pos `v0`, `v1`
            // and the size of `v1out`
            function hashCompareSelect(_ptr, nValues, v0, v1) -> ptr, hash, v0out, v1out, v1outlen {
                ptr := _ptr

                let memStart, nItems
                ptr, memStart, nItems, hash := decodeFlat(ptr)

                if iszero(eq(nItems, nValues)) {
                    revertWith("Node items mismatch")
                }

                v0out, v1outlen := loadValueLen(memStart, v0)
                v1out, v1outlen := loadValueLen(memStart, v1)
            }

            // traverses the tree from the root to the node before the leaf.
            // based on https://github.com/ethereum/go-ethereum/blob/master/trie/proof.go#L114
            function walkTree(key, _ptr) -> ptr, rootHash, expectedHash, path {
                ptr := _ptr

                // the first byte is the number of nodes
                let nodes := byte(0, calldataload(ptr))
                ptr := add(ptr, 1)

                // keeps track of ascend/descend - however you may look at a tree
                let depth

                // treat the leaf node with different logic
                for {
                    let i := 1
                } lt(i, nodes) {
                    i := add(i, 1)
                } {
                    let memStart, nItems, hash
                    ptr, memStart, nItems, hash := decodeFlat(ptr)

                    // first item is considered the root node.
                    // Otherwise verifies that the hash of the current node
                    // is the same as the previous choosen one.
                    switch i
                    case 1 {
                        rootHash := hash
                    }
                    default {
                        require(eq(hash, expectedHash), "Hash mismatch")
                    }

                    switch nItems
                    case 2 {
                        // extension node
                        // load the second item.
                        // this is the hash of the next node.
                        let value, len := loadValueLen(memStart, 1)
                        expectedHash := value

                        // get the byte length of the first item
                        // Note: the value itself is not validated
                        // and it is instead assumed that any invalid
                        // value is invalidated by comparing the root hash.
                        let offset := mload(memStart)
                        let prefixLen := shr(128, offset)
                        // assuming 0xffffff is sufficient for storing calldata offset
                        offset := and(offset, 0xffffff)
                        let flag := shr(252, calldataload(offset))
                        switch flag 
                        case 0 {
                            // extension with even length
                            depth := add(depth, mul(2, sub(prefixLen, 1)))
                        }
                        case 1 {
                            // extension with odd length
                            depth := add(depth, sub(mul(2, prefixLen), 1))
                        }
                        default {
                            // everything else is unexpected
                            revertWith("Invalid extension node")
                        }
                    }
                    case 17 {
                        let bits := sub(252, mul(depth, 4))
                        let nibble := and(shr(bits, key), 0xf)

                        // load the value at pos `nibble`
                        let value, len := loadValueLen(memStart, nibble)

                        expectedHash := value
                        depth := add(depth, 1)
                    }
                    default {
                        // everything else is unexpected
                        revertWith("Invalid node")
                    }
                }

                // lastly, derive the path of the choosen one (TM)
                path := derivePath(key, depth)
            }

            // shared variable names
            let storageHash
            let encodedPath
            let path
            let hash
            let vlen
            // starting point
            let ptr := proof.offset

            {
                // account proof
                // Note: this doesn't work if there are no intermediate nodes before the leaf.
                // This is not possible in practice because of the fact that there must be at least
                // 2 accounts in the tree to make a transaction to a existing contract possible.
                // Thus, 2 leaves.
                let prevHash
                let key := keccak_20(account)
                // `stateRoot` is a return value and must be checked by the caller
                ptr, stateRoot, prevHash, path := walkTree(key, ptr)

                let memStart, nItems
                ptr, memStart, nItems, hash := decodeFlat(ptr)

                // the hash of the leaf must match the previous hash from the node
                require(eq(hash, prevHash), "Account leaf hash mismatch")

                // 2 items
                // - encoded path
                // - account leaf RLP (4 items)
                require(eq(nItems, 2), "Account leaf node mismatch")

                encodedPath := loadValue(memStart, 0)
                // the calculated path must match the encoded path in the leaf
                require(eq(path, encodedPath), "Account encoded path mismatch")

                // Load the position, length of the second element (RLP encoded)
                let leafPtr, leafLen := loadPair(memStart, 1)
                leafPtr, memStart, nItems, hash := decodeFlat(leafPtr)

                // the account leaf should contain 4 values,
                // we want:
                // - storageHash @ 2
                require(eq(nItems, 4), "Account leaf items mismatch")
                storageHash := loadValue(memStart, 2)
            }

            {
                // storage proof
                let rootHash
                let key := keccak_32(storageKey)
                ptr, rootHash, hash, path := walkTree(key, ptr)

                // leaf should contain 2 values
                // - encoded path @ 0
                // - storageValue @ 1
                ptr, hash, encodedPath, storageValue, vlen := hashCompareSelect(ptr, 2, 0, 1)
                // the calculated path must match the encoded path in the leaf
                require(eq(path, encodedPath), "Storage encoded path mismatch")

                switch rootHash
                case 0 {
                    // in the case that the leaf is the only element, then
                    // the hash of the leaf must match the value from the account leaf
                    require(eq(hash, storageHash), "Storage root mismatch")
                }
                default {
                    // otherwise the root hash of the storage tree
                    // must match the value from the account leaf
                    require(eq(rootHash, storageHash), "Storage root mismatch")
                }

                // storageValue is a return value
                storageValue := decodeItem(storageValue, vlen)
            }

            // the one and only boundary check
            // in case an attacker crafted a malicious payload
            // and succeeds in the prior verification steps
            // then this should catch any bogus accesses
            if iszero(eq(ptr, add(proof.offset, proof.length))) {
                revertWith("Proof length mismatch")
            }
        }
    }
}
