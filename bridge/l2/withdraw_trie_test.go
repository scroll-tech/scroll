package l2_test

import (
	"testing"

	"github.com/scroll-tech/go-ethereum/common"

	"scroll-tech/bridge/l2"
	"scroll-tech/bridge/utils"
)

func TestUpdateBranchWithNewMessage(t *testing.T) {
	zeroes := make([]common.Hash, 64)
	branches := make([]common.Hash, 64)
	zeroes[0] = common.Hash{}
	for i := 1; i < 64; i++ {
		zeroes[i] = utils.Keccak2(zeroes[i-1], zeroes[i-1])
	}

	l2.UpdateBranchWithNewMessage(zeroes, branches, 0, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"))
	if branches[0] != common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001") {
		t.Fatalf("Invalid root, want %s, got %s", "0x0000000000000000000000000000000000000000000000000000000000000001", branches[0].Hex())
	}

	l2.UpdateBranchWithNewMessage(zeroes, branches, 1, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"))
	if branches[1] != common.HexToHash("0xe90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0") {
		t.Fatalf("Invalid root, want %s, got %s", "0xe90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0", branches[1].Hex())
	}

	l2.UpdateBranchWithNewMessage(zeroes, branches, 2, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003"))
	if branches[2] != common.HexToHash("0x222ff5e0b5877792c2bc1670e2ccd0c2c97cd7bb1672a57d598db05092d3d72c") {
		t.Fatalf("Invalid root, want %s, got %s", "0x222ff5e0b5877792c2bc1670e2ccd0c2c97cd7bb1672a57d598db05092d3d72c", branches[2].Hex())
	}

	l2.UpdateBranchWithNewMessage(zeroes, branches, 3, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"))
	if branches[2] != common.HexToHash("0xa9bb8c3f1f12e9aa903a50c47f314b57610a3ab32f2d463293f58836def38d36") {
		t.Fatalf("Invalid root, want %s, got %s", "0xa9bb8c3f1f12e9aa903a50c47f314b57610a3ab32f2d463293f58836def38d36", branches[2].Hex())
	}
}

func TestDecodeEncodeMerkleProof(t *testing.T) {
	proof := l2.DecodeBytesToMerkleProof(common.Hex2Bytes("2ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d49012ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d49022ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d49032ebffc1a6671c51e30777a680904b103992630ec995b6e6ff76a04d5259d4904"))
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

	bytes := l2.EncodeMerkleProofToBytes(proof)
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

	proof := l2.UpdateBranchWithNewMessage(zeroes, branches, 0, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"))
	t_branches := l2.RecoverBranchFromProof(proof, 0, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"))
	for i := 0; i < 64; i++ {
		if t_branches[i] != branches[i] {
			t.Fatalf("Invalid branch, want %s, got %s", branches[i].Hex(), t_branches[i].Hex())
		}
	}

	proof = l2.UpdateBranchWithNewMessage(zeroes, branches, 1, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"))
	t_branches = l2.RecoverBranchFromProof(proof, 1, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"))
	for i := 0; i < 64; i++ {
		if t_branches[i] != branches[i] {
			t.Fatalf("Invalid branch, want %s, got %s", branches[i].Hex(), t_branches[i].Hex())
		}
	}

	proof = l2.UpdateBranchWithNewMessage(zeroes, branches, 2, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003"))
	t_branches = l2.RecoverBranchFromProof(proof, 2, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003"))
	for i := 0; i < 64; i++ {
		if t_branches[i] != branches[i] {
			t.Fatalf("Invalid branch, want %s, got %s", branches[i].Hex(), t_branches[i].Hex())
		}
	}

	proof = l2.UpdateBranchWithNewMessage(zeroes, branches, 3, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"))
	t_branches = l2.RecoverBranchFromProof(proof, 3, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"))
	for i := 0; i < 64; i++ {
		if t_branches[i] != branches[i] {
			t.Fatalf("Invalid branch, want %s, got %s", branches[i].Hex(), t_branches[i].Hex())
		}
	}
}
