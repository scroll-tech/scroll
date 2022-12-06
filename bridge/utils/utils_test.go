package utils_test

import (
	"math/big"
	"testing"

	"scroll-tech/bridge/utils"

	"github.com/scroll-tech/go-ethereum/common"
)

func TestKeccak2(t *testing.T) {
	hash := utils.Keccak2(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"), common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"))
	if hash != common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5") {
		t.Fatalf("Invalid keccak, want %s, got %s", "0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5", hash.Hex())
	}

	hash = utils.Keccak2(common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"))
	if hash != common.HexToHash("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30") {
		t.Fatalf("Invalid keccak, want %s, got %s", "0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30", hash.Hex())
	}

	hash = utils.Keccak2(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"), common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"))
	if hash != common.HexToHash("0xe90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0") {
		t.Fatalf("Invalid keccak, want %s, got %s", "0xe90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0", hash.Hex())
	}
}

func TestComputeMessageHash(t *testing.T) {
	hash := utils.ComputeMessageHash(
		common.HexToAddress("0xdafea492d9c6733ae3d56b7ed1adb60692c98bc5"),
		common.HexToAddress("0xeafea492d9c6733ae3d56b7ed1adb60692c98bf7"),
		big.NewInt(1),
		big.NewInt(2),
		big.NewInt(1234567),
		common.Hex2Bytes("0011223344"),
		big.NewInt(3),
	)
	if hash != common.HexToHash("0x58c9a5abfd2a558bb6a6fd5192b36fe9325d98763bafd3a51a1ea28a5d0b990b") {
		t.Fatalf("Invalid ComputeMessageHash, want %s, got %s", "0x58c9a5abfd2a558bb6a6fd5192b36fe9325d98763bafd3a51a1ea28a5d0b990b", hash.Hex())
	}
}
