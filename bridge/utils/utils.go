package utils

import (
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"

	bridgeabi "scroll-tech/bridge/abi"
)

// Keccak2 compute the keccack256 of two concatenations of bytes32
func Keccak2(a common.Hash, b common.Hash) common.Hash {
	return common.BytesToHash(crypto.Keccak256(append(a.Bytes()[:], b.Bytes()[:]...)))
}

// ComputeMessageHash compute the message hash
func ComputeMessageHash(
	sender common.Address,
	target common.Address,
	value *big.Int,
	messageNonce *big.Int,
	message []byte,
) common.Hash {
	data, _ := bridgeabi.L2ScrollMessengerABI.Pack("relayMessage", sender, target, value, messageNonce, message)
	return common.BytesToHash(crypto.Keccak256(data))
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
