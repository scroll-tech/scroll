package utils

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	backendabi "bridge-history-api/abi"
)

// Keccak2 compute the keccack256 of two concatenations of bytes32
func Keccak2(a common.Hash, b common.Hash) common.Hash {
	return common.BytesToHash(crypto.Keccak256(append(a.Bytes()[:], b.Bytes()[:]...)))
}

func GetSafeBlockNumber(ctx context.Context, client *ethclient.Client, confirmations uint64) (uint64, error) {
	number, err := client.BlockNumber(ctx)
	if err != nil || number <= confirmations {
		return 0, err
	}
	number = number - confirmations
	return number, nil
}

// UnpackLog unpacks a retrieved log into the provided output structure.
// @todo: add unit test.
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

// ComputeMessageHash compute the message hash
func ComputeMessageHash(
	sender common.Address,
	target common.Address,
	value *big.Int,
	messageNonce *big.Int,
	message []byte,
) common.Hash {
	data, _ := backendabi.L2ScrollMessengerABI.Pack("relayMessage", sender, target, value, messageNonce, message)
	return common.BytesToHash(crypto.Keccak256(data))
}
