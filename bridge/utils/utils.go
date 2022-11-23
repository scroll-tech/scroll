package utils

import (
	"math/big"

	"github.com/iden3/go-iden3-crypto/keccak256"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
)

// Keccak2 compute the keccack256 of two concatenations of bytes32
func Keccak2(a common.Hash, b common.Hash) common.Hash {
	return common.BytesToHash(keccak256.Hash(append(a.Bytes()[:], b.Bytes()[:]...)))
}

// ComputeMessageHash compute the message hash
func ComputeMessageHash(
	target common.Address,
	sender common.Address,
	value *big.Int,
	fee *big.Int,
	deadline *big.Int,
	message []byte,
	messageNonce *big.Int,
) common.Hash {
	addressType, _ := abi.NewType("address", "address", nil)
	uint256Type, _ := abi.NewType("uint256", "uint256", nil)
	bytesType, _ := abi.NewType("bytes", "bytes", nil)
	args := abi.Arguments{
		{
			Name:    "sender",
			Type:    addressType,
			Indexed: false,
		},
		{
			Name:    "target",
			Type:    addressType,
			Indexed: false,
		},
		{
			Name:    "value",
			Type:    uint256Type,
			Indexed: false,
		},
		{
			Name:    "fee",
			Type:    uint256Type,
			Indexed: false,
		},
		{
			Name:    "deadline",
			Type:    uint256Type,
			Indexed: false,
		},
		{
			Name:    "nonce",
			Type:    uint256Type,
			Indexed: false,
		},
		{
			Name:    "message",
			Type:    bytesType,
			Indexed: false,
		},
	}
	packed, _ := args.Pack(sender, target, value, fee, deadline, messageNonce, message)
	return common.BytesToHash(keccak256.Hash(packed))
}

//nolint:unused
// BufferToUint256Be convert bytes array to uint256 array assuming big-endian
func BufferToUint256Be(buffer []byte) []*big.Int {
	buffer256 := make([]*big.Int, len(buffer)/32)
	for i := 0; i < len(buffer)/32; i++ {
		buffer256[i] = big.NewInt(0)
		for j := 0; j < 32; j++ {
			buffer256[i] = buffer256[i].Lsh(buffer256[i], 8)
			buffer256[i] = buffer256[i].Add(buffer256[i], big.NewInt(int64(buffer[i*32+j])))
		}
	}
	return buffer256
}

// BufferToUint256Le convert bytes array to uint256 array assuming little-endian
func BufferToUint256Le(buffer []byte) []*big.Int {
	buffer256 := make([]*big.Int, len(buffer)/32)
	for i := 0; i < len(buffer)/32; i++ {
		v := big.NewInt(0)
		shft := big.NewInt(1)
		for j := 0; j < 32; j++ {
			v = new(big.Int).Add(v, new(big.Int).Mul(shft, big.NewInt(int64(buffer[i*32+j]))))
			shft = new(big.Int).Mul(shft, big.NewInt(256))
		}
		buffer256[i] = v
	}
	return buffer256
}
