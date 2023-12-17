package utils

import (
	"github.com/scroll-tech/go-ethereum/common"
)

// MaxHeight is the maximum possible height of withdrawal trie
const MaxHeight = 40

// WithdrawTrie is an append only merkle trie
type WithdrawTrie struct {
	// used to rebuild the merkle tree
	NextMessageNonce uint64

	height int // current height of withdraw trie

	branches []common.Hash
	zeroes   []common.Hash
}

// NewWithdrawTrie will return a new instance of WithdrawTrie
func NewWithdrawTrie() *WithdrawTrie {
	zeroes := make([]common.Hash, MaxHeight)
	branches := make([]common.Hash, MaxHeight)

	zeroes[0] = common.Hash{}
	for i := 1; i < MaxHeight; i++ {
		zeroes[i] = Keccak2(zeroes[i-1], zeroes[i-1])
	}

	return &WithdrawTrie{
		zeroes:           zeroes,
		branches:         branches,
		height:           -1,
		NextMessageNonce: 0,
	}
}

// Initialize will initialize the merkle trie with rightest leaf node
func (w *WithdrawTrie) Initialize(currentMessageNonce uint64, msgHash common.Hash, proofBytes []byte) {
	proof := decodeBytesToMerkleProof(proofBytes)
	branches := recoverBranchFromProof(proof, currentMessageNonce, msgHash)
	w.height = len(proof)
	w.branches = branches
	w.NextMessageNonce = currentMessageNonce + 1
}

// AppendMessages appends a list of new messages as leaf nodes to the rightest of the tree and returns the proofs for all messages.
func (w *WithdrawTrie) AppendMessages(hashes []common.Hash) [][]byte {
	length := len(hashes)
	proofs := make([][]byte, length)
	for i := 0; i < length; i++ {
		merkleProof := updateBranchWithNewMessage(w.zeroes, w.branches, w.NextMessageNonce, hashes[i])
		w.NextMessageNonce++
		w.height = len(merkleProof)
		proofs[i] = encodeMerkleProofToBytes(merkleProof)
	}
	return proofs
}

// MessageRoot return the current root hash of withdraw trie.
func (w *WithdrawTrie) MessageRoot() common.Hash {
	if w.height == -1 {
		return common.Hash{}
	}
	return w.branches[w.height]
}

// decodeBytesToMerkleProof transfer byte array to bytes32 array. The caller should make sure the length is matched.
func decodeBytesToMerkleProof(proofBytes []byte) []common.Hash {
	proof := make([]common.Hash, len(proofBytes)/32)
	for i := 0; i < len(proofBytes); i += 32 {
		proof[i/32] = common.BytesToHash(proofBytes[i : i+32])
	}
	return proof
}

// encodeMerkleProofToBytes transfer byte32 array to byte array by concatenation.
func encodeMerkleProofToBytes(proof []common.Hash) []byte {
	var proofBytes []byte
	for i := 0; i < len(proof); i++ {
		proofBytes = append(proofBytes, proof[i][:]...)
	}
	return proofBytes
}

// updateBranchWithNewMessage update the branches to latest with new message and return the merkle proof for the message.
func updateBranchWithNewMessage(zeroes []common.Hash, branches []common.Hash, index uint64, msgHash common.Hash) []common.Hash {
	root := msgHash
	var merkleProof []common.Hash
	var height uint64
	for height = 0; index > 0; height++ {
		if index%2 == 0 {
			// it may be used in next round.
			branches[height] = root
			merkleProof = append(merkleProof, zeroes[height])
			// it's a left child, the right child must be null
			root = Keccak2(root, zeroes[height])
		} else {
			// it's a right child, use previously computed hash
			root = Keccak2(branches[height], root)
			merkleProof = append(merkleProof, branches[height])
		}
		index >>= 1
	}
	branches[height] = root
	return merkleProof
}

// recoverBranchFromProof will recover latest branches from merkle proof and message hash
func recoverBranchFromProof(proof []common.Hash, index uint64, msgHash common.Hash) []common.Hash {
	branches := make([]common.Hash, 64)
	root := msgHash
	var height uint64
	for height = 0; index > 0; height++ {
		if index%2 == 0 {
			branches[height] = root
			// it's a left child, the right child must be null
			root = Keccak2(root, proof[height])
		} else {
			// it's a right child, use previously computed hash
			branches[height] = proof[height]
			root = Keccak2(proof[height], root)
		}
		index >>= 1
	}
	branches[height] = root
	for height++; height < 64; height++ {
		branches[height] = common.Hash{}
	}
	return branches
}
