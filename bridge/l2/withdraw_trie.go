package l2

import (
	"context"
	"errors"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/bridge/utils"
	"scroll-tech/database"
)

// WithdrawTrie is an append only merkle trie
type WithdrawTrie struct {
	// used to rebuild the merkle tree
	nextMessageNonce uint64

	messageRoot common.Hash

	branches []common.Hash
	zeroes   []common.Hash

	ctx context.Context
	orm database.OrmFactory
}

// NewWithdrawTrie will return a new instance of WithdrawTrie
func NewWithdrawTrie(ctx context.Context, orm database.OrmFactory) *WithdrawTrie {
	zeroes := make([]common.Hash, 64)
	branches := make([]common.Hash, 64)

	zeroes[0] = common.Hash{}
	for i := 1; i < 64; i++ {
		zeroes[i] = utils.Keccak2(zeroes[i-1], zeroes[i-1])
	}

	for i := 0; i < 64; i++ {
		branches[i] = common.Hash{}
	}

	return &WithdrawTrie{
		zeroes:           zeroes,
		branches:         branches,
		nextMessageNonce: 0,
		ctx:              ctx,
		orm:              orm,
	}
}

// initialize will initialize the merkle trie with rightest leaf node
func (w *WithdrawTrie) initialize(currentMessageNonce uint64, msgHash common.Hash, proof_bytes []byte) {
	proof := DecodeBytesToMerkleProof(proof_bytes)
	branches := RecoverBranchFromProof(proof, currentMessageNonce, msgHash)
	w.branches = branches
	w.nextMessageNonce = currentMessageNonce + 1
}

// appendMessage will append a new message as the rightest leaf node
func (w *WithdrawTrie) appendMessage(msgNonce uint64, msgHash common.Hash) error {
	if w.nextMessageNonce != msgNonce {
		return errors.New("message nonce mismtach")
	}
	proof := UpdateBranchWithNewMessage(w.zeroes, w.branches, w.nextMessageNonce, msgHash)
	proof_bytes := EncodeMerkleProofToBytes(proof)
	err := w.orm.UpdateMessageProof(w.ctx, msgNonce, common.Bytes2Hex(proof_bytes))
	if err != nil {
		log.Error("Failed to update message proof in db", "err", err)
		return err
	}
	w.nextMessageNonce++
	return nil
}

// DecodeBytesToMerkleProof transfer byte array to bytes32 array. The caller should make sure the length is matched.
func DecodeBytesToMerkleProof(proof_bytes []byte) []common.Hash {
	proof := make([]common.Hash, len(proof_bytes)/32)
	for i := 0; i < len(proof_bytes); i += 32 {
		proof[i/32] = common.BytesToHash(proof_bytes[i : i+32])
	}
	return proof
}

// EncodeMerkleProofToBytes transfer byte32 array to byte array by concatenation.
func EncodeMerkleProofToBytes(proof []common.Hash) []byte {
	var proof_bytes []byte
	for i := 0; i < len(proof); i++ {
		proof_bytes = append(proof_bytes, proof[i][:]...)
	}
	return proof_bytes
}

// UpdateBranchWithNewMessage update the branches to latest with new message and return the merkle proof for the message.
func UpdateBranchWithNewMessage(zeroes []common.Hash, branches []common.Hash, index uint64, msgHash common.Hash) []common.Hash {
	root := msgHash
	var merkleProof []common.Hash
	var height uint64
	for height = 0; index > 0; height++ {
		if index%2 == 0 {
			// it may be used in next round.
			branches[height] = root
			merkleProof = append(merkleProof, zeroes[height])
			// it's a left child, the right child must be null
			root = utils.Keccak2(root, zeroes[height])
		} else {
			// it's a right child, use previously computed hash
			root = utils.Keccak2(branches[height], root)
			merkleProof = append(merkleProof, branches[height])
		}
		index >>= 1
	}
	branches[height] = root
	return merkleProof
}

// RecoverBranchFromProof will recover latest branches from merkle proof and message hash
func RecoverBranchFromProof(proof []common.Hash, index uint64, msgHash common.Hash) []common.Hash {
	branches := make([]common.Hash, 64)
	root := msgHash
	var height uint64
	for height = 0; index > 0; height++ {
		if index%2 == 0 {
			branches[height] = root
			// it's a left child, the right child must be null
			root = utils.Keccak2(root, proof[height])
		} else {
			// it's a right child, use previously computed hash
			branches[height] = proof[height]
			root = utils.Keccak2(proof[height], root)
		}
		index >>= 1
	}
	branches[height] = root
	for height++; height < 64; height++ {
		branches[height] = common.Hash{}
	}
	return branches
}
