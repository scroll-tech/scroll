package utils

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"github.com/iden3/go-iden3-crypto/keccak256"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/math"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient/gethclient"
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

// UnpackLog unpacks a retrieved log into the provided output structure.
func UnpackLog(c *abi.ABI, out interface{}, event string, log types.Log) error {
	if log.Topics[0] != c.Events[event].ID {
		return fmt.Errorf("event signature mismatch")
	}
	if len(log.Data) > 0 {
		if err := c.UnpackIntoInterface(out, event, log.Data); err != nil {
			return err
		}
	}
	var indexed abi.Arguments
	for _, arg := range c.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return abi.ParseTopics(out, indexed, log.Topics[1:])
}

// UnpackLogIntoMap unpacks a retrieved log into the provided map.
func UnpackLogIntoMap(c *abi.ABI, out map[string]interface{}, event string, log types.Log) error {
	if log.Topics[0] != c.Events[event].ID {
		return fmt.Errorf("event signature mismatch")
	}
	if len(log.Data) > 0 {
		if err := c.UnpackIntoMap(out, event, log.Data); err != nil {
			return err
		}
	}
	var indexed abi.Arguments
	for _, arg := range c.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return abi.ParseTopicsIntoMap(out, indexed, log.Topics[1:])
}

// GetStorageProof will fetch storage proof from geth client
func GetL1MessageProof(client *gethclient.Client, account common.Address, hashes []common.Hash, height uint64) ([][]byte, error) {
	slot := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	var keys []string
	for _, hash := range hashes {
		keys = append(keys, Keccak2(hash, slot).String())
	}
	results, err := client.GetProof(context.Background(), account, keys, big.NewInt(int64(height)))
	if err != nil {
		return make([][]byte, 0), err
	}

	accountProof := results.AccountProof
	var proofs [][]byte
	for i := 0; i < len(hashes); i++ {
		var proof []byte
		proof = append(proof, big.NewInt(int64(len(accountProof))).Bytes()...)
		for _, item := range results.AccountProof {
			// remove 0x prefix
			proof = append(proof, common.Hex2Bytes(item[2:])...)
		}

		// the storage proof should have the same order with `hashes`
		storageProof := results.StorageProof[i]
		proof = append(proof, big.NewInt(int64(len(storageProof.Proof))).Bytes()...)
		for _, item := range storageProof.Proof {
			// remove 0x prefix
			proof = append(proof, common.Hex2Bytes(item[2:])...)
		}

		proofs = append(proofs, proof)
	}

	return proofs, nil
}
