package utils

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestKeccak2(t *testing.T) {
	hash := Keccak2(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"), common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"))
	if hash != common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5") {
		t.Fatalf("Invalid keccak, want %s, got %s", "0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5", hash.Hex())
	}

	hash = Keccak2(common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"), common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"))
	if hash != common.HexToHash("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30") {
		t.Fatalf("Invalid keccak, want %s, got %s", "0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30", hash.Hex())
	}

	hash = Keccak2(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"), common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"))
	if hash != common.HexToHash("0xe90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0") {
		t.Fatalf("Invalid keccak, want %s, got %s", "0xe90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0", hash.Hex())
	}
}

func TestComputeMessageHash(t *testing.T) {
	hash := ComputeMessageHash(
		common.HexToAddress("0x1C5A77d9FA7eF466951B2F01F724BCa3A5820b63"),
		common.HexToAddress("0x4592D8f8D7B001e72Cb26A73e4Fa1806a51aC79d"),
		big.NewInt(0),
		big.NewInt(1),
		[]byte("testbridgecontract"),
	)
	assert.Equal(t, "0xda253c04595a49017bb54b1b46088c69752b5ad2f0c47971ac76b8b25abec202", hash.String())
}

func TestBufferToUint256Le(t *testing.T) {
	input := []byte{
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	expectedOutput := []*big.Int{big.NewInt(1)}
	result := BufferToUint256Le(input)
	assert.Equal(t, expectedOutput, result)
}
