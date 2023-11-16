package messageproof

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"bridge-history-api/utils"
)

func TestUpdateBranchWithNewMessage(t *testing.T) {
	zeroes := make([]common.Hash, 64)
	branches := make([]common.Hash, 64)
	zeroes[0] = common.Hash{}
	for i := 1; i < 64; i++ {
		zeroes[i] = utils.Keccak2(zeroes[i-1], zeroes[i-1])
	}

	UpdateBranchWithNewMessage(zeroes, branches, 0, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"))
	if branches[0] != common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001") {
		t.Fatalf("Invalid root, want %s, got %s", "0x0000000000000000000000000000000000000000000000000000000000000001", branches[0].Hex())
	}

	UpdateBranchWithNewMessage(zeroes, branches, 1, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"))
	if branches[1] != common.HexToHash("0xe90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0") {
		t.Fatalf("Invalid root, want %s, got %s", "0xe90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0", branches[1].Hex())
	}

	UpdateBranchWithNewMessage(zeroes, branches, 2, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003"))
	if branches[2] != common.HexToHash("0x222ff5e0b5877792c2bc1670e2ccd0c2c97cd7bb1672a57d598db05092d3d72c") {
		t.Fatalf("Invalid root, want %s, got %s", "0x222ff5e0b5877792c2bc1670e2ccd0c2c97cd7bb1672a57d598db05092d3d72c", branches[2].Hex())
	}

	UpdateBranchWithNewMessage(zeroes, branches, 3, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"))
	if branches[2] != common.HexToHash("0xa9bb8c3f1f12e9aa903a50c47f314b57610a3ab32f2d463293f58836def38d36") {
		t.Fatalf("Invalid root, want %s, got %s", "0xa9bb8c3f1f12e9aa903a50c47f314b57610a3ab32f2d463293f58836def38d36", branches[2].Hex())
	}
}

func TestDecodeEncodeMerkleProof(t *testing.T) {
	proof := DecodeBytesToMerkleProof(common.Hex2Bytes("2ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d49012ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d49022ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d49032ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d4904"))
	if len(proof) != 4 {
		t.Fatalf("proof length mismatch, want %d, got %d", 4, len(proof))
	}
	if proof[0] != common.HexToHash("0x2ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d4901") {
		t.Fatalf("proof[0] mismatch, want %s, got %s", "0x2ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d4901", proof[0].Hex())
	}
	if proof[1] != common.HexToHash("0x2ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d4902") {
		t.Fatalf("proof[1] mismatch, want %s, got %s", "0x2ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d4902", proof[0].Hex())
	}
	if proof[2] != common.HexToHash("0x2ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d4903") {
		t.Fatalf("proof[2] mismatch, want %s, got %s", "0x2ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d4903", proof[0].Hex())
	}
	if proof[3] != common.HexToHash("0x2ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d4904") {
		t.Fatalf("proof[3] mismatch, want %s, got %s", "0x2ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d4904", proof[0].Hex())
	}

	bytes := EncodeMerkleProofToBytes(proof)
	if common.Bytes2Hex(bytes) != "2ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d49012ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d49022ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d49032ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d4904" {
		t.Fatalf("wrong encoded bytes")
	}
}

func TestRecoverBranchFromProof(t *testing.T) {
	zeroes := make([]common.Hash, 64)
	branches := make([]common.Hash, 64)
	zeroes[0] = common.Hash{}
	for i := 1; i < 64; i++ {
		zeroes[i] = utils.Keccak2(zeroes[i-1], zeroes[i-1])
	}

	proof := UpdateBranchWithNewMessage(zeroes, branches, 0, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"))
	tmpBranches := RecoverBranchFromProof(proof, 0, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"))
	for i := 0; i < 64; i++ {
		if tmpBranches[i] != branches[i] {
			t.Fatalf("Invalid branch, want %s, got %s", branches[i].Hex(), tmpBranches[i].Hex())
		}
	}

	proof = UpdateBranchWithNewMessage(zeroes, branches, 1, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"))
	tmpBranches = RecoverBranchFromProof(proof, 1, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"))
	for i := 0; i < 64; i++ {
		if tmpBranches[i] != branches[i] {
			t.Fatalf("Invalid branch, want %s, got %s", branches[i].Hex(), tmpBranches[i].Hex())
		}
	}

	proof = UpdateBranchWithNewMessage(zeroes, branches, 2, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003"))
	tmpBranches = RecoverBranchFromProof(proof, 2, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003"))
	for i := 0; i < 64; i++ {
		if tmpBranches[i] != branches[i] {
			t.Fatalf("Invalid branch, want %s, got %s", branches[i].Hex(), tmpBranches[i].Hex())
		}
	}

	proof = UpdateBranchWithNewMessage(zeroes, branches, 3, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"))
	tmpBranches = RecoverBranchFromProof(proof, 3, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"))
	for i := 0; i < 64; i++ {
		if tmpBranches[i] != branches[i] {
			t.Fatalf("Invalid branch, want %s, got %s", branches[i].Hex(), tmpBranches[i].Hex())
		}
	}
}

func TestWithdrawTrieOneByOne(t *testing.T) {
	for initial := 0; initial < 128; initial++ {
		withdrawTrie := NewWithdrawTrie()
		var hashes []common.Hash
		for i := 0; i < initial; i++ {
			hash := common.BigToHash(big.NewInt(int64(i + 1)))
			hashes = append(hashes, hash)
			withdrawTrie.AppendMessages([]common.Hash{
				hash,
			})
		}

		for i := initial; i < 128; i++ {
			hash := common.BigToHash(big.NewInt(int64(i + 1)))
			hashes = append(hashes, hash)
			expectedRoot := computeMerkleRoot(hashes)
			proofBytes := withdrawTrie.AppendMessages([]common.Hash{
				hash,
			})
			assert.Equal(t, withdrawTrie.NextMessageNonce, uint64(i+1))
			assert.Equal(t, expectedRoot.String(), withdrawTrie.MessageRoot().String())
			proof := DecodeBytesToMerkleProof(proofBytes[0])
			verifiedRoot := verifyMerkleProof(uint64(i), hash, proof)
			assert.Equal(t, expectedRoot.String(), verifiedRoot.String())
		}
	}
}

func TestWithdrawTrieMultiple(t *testing.T) {
	var expectedRoots []common.Hash

	{
		var hashes []common.Hash
		for i := 0; i < 128; i++ {
			hash := common.BigToHash(big.NewInt(int64(i + 1)))
			hashes = append(hashes, hash)
			expectedRoots = append(expectedRoots, computeMerkleRoot(hashes))
		}
	}

	for initial := 0; initial < 100; initial++ {
		var hashes []common.Hash
		for i := 0; i < initial; i++ {
			hash := common.BigToHash(big.NewInt(int64(i + 1)))
			hashes = append(hashes, hash)
		}

		for finish := initial; finish < 100; finish++ {
			withdrawTrie := NewWithdrawTrie()
			withdrawTrie.AppendMessages(hashes)

			var newHashes []common.Hash
			for i := initial; i <= finish; i++ {
				hash := common.BigToHash(big.NewInt(int64(i + 1)))
				newHashes = append(newHashes, hash)
			}
			proofBytes := withdrawTrie.AppendMessages(newHashes)
			assert.Equal(t, withdrawTrie.NextMessageNonce, uint64(finish+1))
			assert.Equal(t, expectedRoots[finish].String(), withdrawTrie.MessageRoot().String())

			for i := initial; i <= finish; i++ {
				hash := common.BigToHash(big.NewInt(int64(i + 1)))
				proof := DecodeBytesToMerkleProof(proofBytes[i-initial])
				verifiedRoot := verifyMerkleProof(uint64(i), hash, proof)
				assert.Equal(t, expectedRoots[finish].String(), verifiedRoot.String())
			}
		}
	}
}

func verifyMerkleProof(index uint64, leaf common.Hash, proof []common.Hash) common.Hash {
	root := leaf
	for _, h := range proof {
		if index%2 == 0 {
			root = utils.Keccak2(root, h)
		} else {
			root = utils.Keccak2(h, root)
		}
		index >>= 1
	}
	return root
}

func computeMerkleRoot(hashes []common.Hash) common.Hash {
	if len(hashes) == 0 {
		return common.Hash{}
	}

	zeroHash := common.Hash{}
	for {
		if len(hashes) == 1 {
			break
		}
		var newHashes []common.Hash
		for i := 0; i < len(hashes); i += 2 {
			if i+1 < len(hashes) {
				newHashes = append(newHashes, utils.Keccak2(hashes[i], hashes[i+1]))
			} else {
				newHashes = append(newHashes, utils.Keccak2(hashes[i], zeroHash))
			}
		}
		hashes = newHashes
		zeroHash = utils.Keccak2(zeroHash, zeroHash)
	}
	return hashes[0]
}
