package utils

import (
	"bytes"
	"context"
	"math/big"
	"time"

	"github.com/iden3/go-iden3-crypto/keccak256"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/math"
)

// Keccak2 compute the keccack256 of two concatenations of bytes32
func Keccak2(a common.Hash, b common.Hash) common.Hash {
	return common.BytesToHash(keccak256.Hash(append(a.Bytes()[:], b.Bytes()[:]...)))
}

func encodePacked(input ...[]byte) []byte {
	return bytes.Join(input, nil)
}

// ComputeMessageHash compute the message hash
func ComputeMessageHash(
	sender common.Address,
	target common.Address,
	value *big.Int,
	fee *big.Int,
	deadline *big.Int,
	message []byte,
	messageNonce *big.Int,
) common.Hash {
	packed := encodePacked(
		sender.Bytes(),
		target.Bytes(),
		math.U256Bytes(value),
		math.U256Bytes(fee),
		math.U256Bytes(deadline),
		math.U256Bytes(messageNonce),
		message,
	)
	return common.BytesToHash(keccak256.Hash(packed))
}

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

// Loop Run the f func periodically.
func Loop(ctx context.Context, tick *time.Ticker, f func(ctx context.Context)) {
	defer tick.Stop()
	for ; ; <-tick.C {
		select {
		case <-ctx.Done():
			return
		default:
			f(ctx)
		}
	}
}
