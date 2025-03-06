package utils

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"golang.org/x/sync/errgroup"

	backendabi "scroll-tech/bridge-history-api/abi"
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
		return errors.New("event signature mismatch")
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

// GetBatchRangeFromCalldata find the block range from calldata, both inclusive.
func GetBatchRangeFromCalldata(txData []byte) (uint64, uint64, error) {
	const methodIDLength = 4
	if len(txData) < methodIDLength {
		return 0, 0, fmt.Errorf("transaction data is too short, length of tx data: %v, minimum length required: %v", len(txData), methodIDLength)
	}
	method, err := backendabi.IScrollChainABI.MethodById(txData[:methodIDLength])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get method by ID, ID: %v, err: %w", txData[:methodIDLength], err)
	}
	values, err := method.Inputs.Unpack(txData[methodIDLength:])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to unpack transaction data using ABI, tx data: %v, err: %w", txData, err)
	}

	var chunks [][]byte

	if method.Name == "importGenesisBatch" {
		return 0, 0, nil
	} else if method.Name == "commitBatch" {
		type commitBatchArgs struct {
			Version                uint8
			ParentBatchHeader      []byte
			Chunks                 [][]byte
			SkippedL1MessageBitmap []byte
		}

		var args commitBatchArgs
		if err = method.Inputs.Copy(&args, values); err != nil {
			return 0, 0, fmt.Errorf("failed to decode calldata into commitBatch args, values: %+v, err: %w", values, err)
		}

		chunks = args.Chunks

	} else if method.Name == "commitBatchWithBlobProof" {
		type commitBatchWithBlobProofArgs struct {
			Version                uint8
			ParentBatchHeader      []byte
			Chunks                 [][]byte
			SkippedL1MessageBitmap []byte
			BlobDataProof          []byte
		}

		var args commitBatchWithBlobProofArgs
		if err = method.Inputs.Copy(&args, values); err != nil {
			return 0, 0, fmt.Errorf("failed to decode calldata into commitBatchWithBlobProofArgs args, values: %+v, err: %w", values, err)
		}

		chunks = args.Chunks
	}

	var startBlock uint64
	var finishBlock uint64

	// decode blocks from chunk and assume that there's no empty chunk
	// |   1 byte   | 60 bytes | ... | 60 bytes |
	// | num blocks |  block 1 | ... |  block n |
	if len(chunks) == 0 {
		return 0, 0, errors.New("invalid chunks")
	}
	chunk := chunks[0]
	block := chunk[1:61] // first block in chunk
	startBlock = binary.BigEndian.Uint64(block[0:8])

	chunk = chunks[len(chunks)-1]
	lastBlockIndex := int(chunk[0]) - 1
	block = chunk[1+lastBlockIndex*60 : 1+lastBlockIndex*60+60] // last block in chunk
	finishBlock = binary.BigEndian.Uint64(block[0:8])

	return startBlock, finishBlock, err
}

// GetBlocksInRange gets a batch of blocks for a block range [start, end] inclusive.
func GetBlocksInRange(ctx context.Context, cli *ethclient.Client, start, end uint64) ([]*types.Block, error) {
	var (
		eg          errgroup.Group
		blocks      = make([]*types.Block, end-start+1)
		concurrency = 32
		sem         = make(chan struct{}, concurrency)
	)

	for i := start; i <= end; i++ {
		sem <- struct{}{} // Acquire a slot in the semaphore
		blockNum := int64(i)
		index := i - start
		eg.Go(func() error {
			defer func() { <-sem }() // Release the slot when done
			block, err := cli.BlockByNumber(ctx, big.NewInt(blockNum))
			if err != nil {
				log.Error("Failed to fetch block number", "number", blockNum, "error", err)
				return err
			}
			blocks[index] = block
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		log.Error("Error waiting for block fetching routines", "error", err)
		return nil, err
	}
	return blocks, nil
}

// ConvertBigIntArrayToString convert the big int array to string
func ConvertBigIntArrayToString(array []*big.Int) string {
	stringArray := make([]string, len(array))
	for i, num := range array {
		stringArray[i] = num.String()
	}

	result := strings.Join(stringArray, ", ")
	return result
}

// ConvertStringToStringArray takes a string with values separated by commas and returns a slice of strings
func ConvertStringToStringArray(s string) []string {
	if s == "" {
		return []string{}
	}
	stringParts := strings.Split(s, ",")
	for i, part := range stringParts {
		stringParts[i] = strings.TrimSpace(part)
	}
	return stringParts
}

// GetSkippedQueueIndices gets the skipped queue indices
func GetSkippedQueueIndices(startIndex uint64, skippedBitmap *big.Int) []uint64 {
	var indices []uint64
	for i := 0; i < 256; i++ {
		index := startIndex + uint64(i)
		bit := new(big.Int).Rsh(skippedBitmap, uint(i))
		if bit.Bit(0) == 0 {
			continue
		}
		indices = append(indices, index)
	}
	return indices
}
