package utils

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
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

// GetSafeBlockNumber get the safe block number, which is the current block number minus the confirmations
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

type commitBatchArgs struct {
	Version                uint8
	ParentBatchHeader      []byte
	Chunks                 [][]byte
	SkippedL1MessageBitmap []byte
}

// GetBatchRangeFromCalldataV2 find the block range from calldata, both inclusive.
func GetBatchRangeFromCalldataV2(calldata []byte) (uint64, uint64, uint64, error) {
	method := backendabi.ScrollChainV2ABI.Methods["commitBatch"]
	values, err := method.Inputs.Unpack(calldata[4:])
	if err != nil {
		return 0, 0, 0, err
	}
	args := commitBatchArgs{}
	err = method.Inputs.Copy(&args, values)
	if err != nil {
		return 0, 0, 0, err
	}

	var startBlock uint64
	var finishBlock uint64

	// decode batchIndex from ParentBatchHeader
	if len(args.ParentBatchHeader) < 9 {
		return 0, 0, 0, errors.New("invalid parent batch header")
	}
	batchIndex := binary.BigEndian.Uint64(args.ParentBatchHeader[1:9]) + 1

	// decode blocks from chunk and assume that there's no empty chunk
	// |   1 byte   | 60 bytes | ... | 60 bytes |
	// | num blocks |  block 1 | ... |  block n |
	if len(args.Chunks) == 0 {
		return 0, 0, 0, errors.New("invalid chunks")
	}
	chunk := args.Chunks[0]
	block := chunk[1:61] // first block in chunk
	startBlock = binary.BigEndian.Uint64(block[0:8])

	chunk = args.Chunks[len(args.Chunks)-1]
	lastBlockIndex := int(chunk[0]) - 1
	block = chunk[1+lastBlockIndex*60 : 1+lastBlockIndex*60+60] // last block in chunk
	finishBlock = binary.BigEndian.Uint64(block[0:8])

	return batchIndex, startBlock, finishBlock, err
}

// GetBatchRangeFromCalldataV1 find the block range from calldata, both inclusive.
func GetBatchRangeFromCalldataV1(calldata []byte) ([]uint64, []uint64, []uint64, error) {
	var batchIndices []uint64
	var startBlocks []uint64
	var finishBlocks []uint64
	if bytes.Equal(calldata[0:4], common.Hex2Bytes("cb905499")) {
		// commitBatches
		method := backendabi.ScrollChainABI.Methods["commitBatches"]
		values, err := method.Inputs.Unpack(calldata[4:])
		if err != nil {
			return batchIndices, startBlocks, finishBlocks, err
		}
		args := make([]backendabi.IScrollChainBatch, len(values))
		err = method.Inputs.Copy(&args, values)
		if err != nil {
			return batchIndices, startBlocks, finishBlocks, err
		}

		for i := 0; i < len(args); i++ {
			batchIndices = append(batchIndices, args[i].BatchIndex)
			startBlocks = append(startBlocks, args[i].Blocks[0].BlockNumber)
			finishBlocks = append(finishBlocks, args[i].Blocks[len(args[i].Blocks)-1].BlockNumber)
		}
	} else if bytes.Equal(calldata[0:4], common.Hex2Bytes("8c73235d")) {
		// commitBatch
		method := backendabi.ScrollChainABI.Methods["commitBatch"]
		values, err := method.Inputs.Unpack(calldata[4:])
		if err != nil {
			return batchIndices, startBlocks, finishBlocks, err
		}

		args := backendabi.IScrollChainBatch{}
		err = method.Inputs.Copy(&args, values)
		if err != nil {
			return batchIndices, startBlocks, finishBlocks, err
		}
		batchIndices = append(batchIndices, args.BatchIndex)
		startBlocks = append(startBlocks, args.Blocks[0].BlockNumber)
		finishBlocks = append(finishBlocks, args.Blocks[len(args.Blocks)-1].BlockNumber)
	} else {
		return batchIndices, startBlocks, finishBlocks, errors.New("invalid selector")
	}
	return batchIndices, startBlocks, finishBlocks, nil
}
