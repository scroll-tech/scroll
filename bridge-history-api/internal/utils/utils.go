package utils

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
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

// GetBatchVersionAndBlockRangeFromCalldata find the block range from calldata, both inclusive.
func GetBatchVersionAndBlockRangeFromCalldata(txData []byte) (uint8, uint64, uint64, error) {
	const methodIDLength = 4
	if len(txData) < methodIDLength {
		return 0, 0, 0, fmt.Errorf("transaction data is too short, length of tx data: %v, minimum length required: %v", len(txData), methodIDLength)
	}
	method, err := backendabi.IScrollChainABI.MethodById(txData[:methodIDLength])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get method by ID, ID: %v, err: %w", txData[:methodIDLength], err)
	}
	values, err := method.Inputs.Unpack(txData[methodIDLength:])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to unpack transaction data using ABI, tx data: %v, err: %w", txData, err)
	}

	var chunks [][]byte
	var version uint8

	if method.Name == "importGenesisBatch" {
		return 0, 0, 0, nil
	} else if method.Name == "commitBatch" {
		type commitBatchArgs struct {
			Version                uint8
			ParentBatchHeader      []byte
			Chunks                 [][]byte
			SkippedL1MessageBitmap []byte
		}

		var args commitBatchArgs
		if err = method.Inputs.Copy(&args, values); err != nil {
			return 0, 0, 0, fmt.Errorf("failed to decode calldata into commitBatch args, values: %+v, err: %w", values, err)
		}

		chunks = args.Chunks
		version = args.Version
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
			return 0, 0, 0, fmt.Errorf("failed to decode calldata into commitBatchWithBlobProofArgs args, values: %+v, err: %w", values, err)
		}

		chunks = args.Chunks
		version = args.Version
	} else if method.Name == "commitBatches" || method.Name == "commitAndFinalizeBatch" {
		if len(values) < 3 {
			return 0, 0, 0, fmt.Errorf("insufficient arguments for %s, expected 3, got %d", method.Name, len(values))
		}

		var ok bool
		version, ok = values[0].(uint8)
		if !ok {
			return 0, 0, 0, fmt.Errorf("invalid version type: %T", values[0])
		}
		return version, 0, 0, nil
	}

	var startBlock uint64
	var finishBlock uint64

	// decode blocks from chunk and assume that there's no empty chunk
	// |   1 byte   | 60 bytes | ... | 60 bytes |
	// | num blocks |  block 1 | ... |  block n |
	if len(chunks) == 0 {
		return 0, 0, 0, errors.New("invalid chunks")
	}
	chunk := chunks[0]
	block := chunk[1:61] // first block in chunk
	startBlock = binary.BigEndian.Uint64(block[0:8])

	chunk = chunks[len(chunks)-1]
	lastBlockIndex := int(chunk[0]) - 1
	block = chunk[1+lastBlockIndex*60 : 1+lastBlockIndex*60+60] // last block in chunk
	finishBlock = binary.BigEndian.Uint64(block[0:8])

	return version, startBlock, finishBlock, err
}

// GetParentBatchHashFromCalldata gets the parent batch hash from calldata.
// It only supports commitBatches and commitAndFinalizeBatch, which only accept batches >= v7.
func GetParentBatchHashFromCalldata(txData []byte) (common.Hash, error) {
	const methodIDLength = 4
	if len(txData) < methodIDLength {
		return common.Hash{}, fmt.Errorf("transaction data is too short, length of tx data: %v, minimum length required: %v", len(txData), methodIDLength)
	}
	method, err := backendabi.IScrollChainABI.MethodById(txData[:methodIDLength])
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get method by ID, ID: %v, err: %w", txData[:methodIDLength], err)
	}
	values, err := method.Inputs.Unpack(txData[methodIDLength:])
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to unpack transaction data using ABI, tx data: %v, err: %w", txData, err)
	}

	if method.Name == "commitBatches" || method.Name == "commitAndFinalizeBatch" {
		if len(values) < 3 {
			return common.Hash{}, fmt.Errorf("insufficient arguments for %s, expected 3, got %d", method.Name, len(values))
		}
		parentBatchHash, ok := values[1].([32]byte)
		if !ok {
			return common.Hash{}, fmt.Errorf("invalid parentBatchHash type: %T", values[1])
		}
		return common.BytesToHash(parentBatchHash[:]), nil
	}
	return common.Hash{}, fmt.Errorf("method %s does not support parent batch header", method.Name)
}

// Block contains the block fields that the fetchers use.
// Hash comes from the RPC node, not from types.Header.Hash(). The SDK hashes its own Header struct,
// which follows Scroll L2 rules: it lacks newer L1 fields (e.g. EIP-7685 requestsHash) and encodes
// Scroll-only fields after baseFee. For post-Shanghai L1 blocks the computed hash is therefore wrong.
type Block struct {
	Number       uint64
	Hash         common.Hash
	ParentHash   common.Hash
	Time         uint64
	Transactions types.Transactions
}

type rpcBlock struct {
	Number       *hexutil.Uint64      `json:"number"`
	Hash         *common.Hash         `json:"hash"`
	ParentHash   *common.Hash         `json:"parentHash"`
	Time         *hexutil.Uint64      `json:"timestamp"`
	TxHash       *common.Hash         `json:"transactionsRoot"`
	Transactions []*types.Transaction `json:"transactions"`
}

// GetBlockByNumber gets a block with its transactions and its RPC-reported hash.
func GetBlockByNumber(ctx context.Context, cli *rpc.Client, number uint64) (*Block, error) {
	var block *rpcBlock
	if err := cli.CallContext(ctx, &block, "eth_getBlockByNumber", hexutil.EncodeUint64(number), true); err != nil {
		return nil, err
	}
	if block == nil {
		return nil, ethereum.NotFound
	}
	if block.Number == nil || block.Hash == nil || block.ParentHash == nil || block.Time == nil || block.TxHash == nil {
		return nil, fmt.Errorf("block %d response is missing required fields", number)
	}
	if *block.Hash == (common.Hash{}) {
		return nil, fmt.Errorf("block %d has a zero hash", number)
	}
	// Same sanity checks as ethclient: a missing transaction list would silently drop reverted txs.
	if (*block.TxHash == types.EmptyRootHash) != (len(block.Transactions) == 0) {
		return nil, fmt.Errorf("block %d transaction list does not match transactions root", number)
	}
	return &Block{
		Number:       uint64(*block.Number),
		Hash:         *block.Hash,
		ParentHash:   *block.ParentHash,
		Time:         uint64(*block.Time),
		Transactions: block.Transactions,
	}, nil
}

// GetBlocksInRange gets a batch of blocks for a block range [start, end] inclusive.
func GetBlocksInRange(ctx context.Context, cli *rpc.Client, start, end uint64) ([]*Block, error) {
	var (
		eg          errgroup.Group
		blocks      = make([]*Block, end-start+1)
		concurrency = 32
		sem         = make(chan struct{}, concurrency)
	)

	for i := start; i <= end; i++ {
		sem <- struct{}{} // Acquire a slot in the semaphore
		blockNum := i
		index := i - start
		eg.Go(func() error {
			defer func() { <-sem }() // Release the slot when done
			block, err := GetBlockByNumber(ctx, cli, blockNum)
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
