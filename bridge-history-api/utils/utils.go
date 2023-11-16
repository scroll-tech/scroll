package utils

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"

	backendabi "bridge-history-api/abi"
)

// Keccak2 compute the keccack256 of two concatenations of bytes32
func Keccak2(a common.Hash, b common.Hash) common.Hash {
	return common.BytesToHash(crypto.Keccak256(append(a.Bytes()[:], b.Bytes()[:]...)))
}

// GetBlockNumber get the current block number minus the confirmations
func GetBlockNumber(ctx context.Context, client *ethclient.Client, confirmations uint64) (uint64, error) {
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
	data, _ := backendabi.IL2ScrollMessengerABI.Pack("relayMessage", sender, target, value, messageNonce, message)
	return common.BytesToHash(crypto.Keccak256(data))
}

type commitBatchArgs struct {
	Version                uint8
	ParentBatchHeader      []byte
	Chunks                 [][]byte
	SkippedL1MessageBitmap []byte
}

// GetBatchRangeFromCalldata find the block range from calldata, both inclusive.
func GetBatchRangeFromCalldata(calldata []byte) (uint64, uint64, error) {
	method := backendabi.IScrollChainABI.Methods["commitBatch"]
	values, err := method.Inputs.Unpack(calldata[4:])
	if err != nil {
		// special case: import genesis batch
		method = backendabi.IScrollChainABI.Methods["importGenesisBatch"]
		_, err2 := method.Inputs.Unpack(calldata[4:])
		if err2 == nil {
			// genesis batch
			return 0, 0, nil
		}
		// none of "commitBatch" and "importGenesisBatch" match, give up
		return 0, 0, err
	}
	args := commitBatchArgs{}
	err = method.Inputs.Copy(&args, values)
	if err != nil {
		return 0, 0, err
	}

	var startBlock uint64
	var finishBlock uint64

	// decode blocks from chunk and assume that there's no empty chunk
	// |   1 byte   | 60 bytes | ... | 60 bytes |
	// | num blocks |  block 1 | ... |  block n |
	if len(args.Chunks) == 0 {
		return 0, 0, errors.New("invalid chunks")
	}
	chunk := args.Chunks[0]
	block := chunk[1:61] // first block in chunk
	startBlock = binary.BigEndian.Uint64(block[0:8])

	chunk = args.Chunks[len(args.Chunks)-1]
	lastBlockIndex := int(chunk[0]) - 1
	block = chunk[1+lastBlockIndex*60 : 1+lastBlockIndex*60+60] // last block in chunk
	finishBlock = binary.BigEndian.Uint64(block[0:8])

	return startBlock, finishBlock, err
}
