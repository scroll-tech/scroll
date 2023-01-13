package utils_test

import (
	"math/big"
	"testing"

	"scroll-tech/bridge/utils"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"
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
		common.HexToAddress("0xd7227113b92e537aeda220d5a2f201b836e5879d"),
		common.HexToAddress("0x47c02b023b6787ef4e503df42bbb1a94f451a1c0"),
		big.NewInt(5000000000000000),
		big.NewInt(0),
		big.NewInt(1674204924),
		common.Hex2Bytes("8eaac8a30000000000000000000000007138b17fc82d7e954b3bd2f98d8166d03e5e569b0000000000000000000000007138b17fc82d7e954b3bd2f98d8166d03e5e569b0000000000000000000000000000000000000000000000000011c37937e0800000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000000"),
		big.NewInt(30706),
	)
	assert.Equal(t, hash.String(), "0x920e59f62ca89a0f481d44961c55d299dd20c575693692d61fdf3ca579d8edf3")
}
