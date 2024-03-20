package utils

import (
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"

	"scroll-tech/common/types/encoding"
	"scroll-tech/common/types/encoding/codecv0"
	"scroll-tech/common/types/encoding/codecv1"

	bridgeAbi "scroll-tech/rollup/abi"
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
	data, _ := bridgeAbi.L2ScrollMessengerABI.Pack("relayMessage", sender, target, value, messageNonce, message)
	return common.BytesToHash(crypto.Keccak256(data))
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

// ChunkMetrics indicates the metrics for proposing a chunk.
type ChunkMetrics struct {
	// common metrics
	NumBlocks           uint64
	TxNum               uint64
	CrcMax              uint64
	FirstBlockTimestamp uint64

	// codecv0 metrics, default 0 for codecv1
	L1CommitCalldataSize uint64
	L1CommitGas          uint64

	// codecv1 metrics, default 0 for codecv0
	L1CommitBlobSize uint64
}

// CalculateChunkMetrics calculates chunk metrics.
func CalculateChunkMetrics(chunk *encoding.Chunk, useCodecv0 bool) (*ChunkMetrics, error) {
	var err error
	metrics := &ChunkMetrics{
		TxNum:               chunk.NumTransactions(),
		NumBlocks:           uint64(len(chunk.Blocks)),
		FirstBlockTimestamp: chunk.Blocks[0].Header.Time,
	}
	metrics.CrcMax, err = chunk.CrcMax()
	if err != nil {
		return metrics, fmt.Errorf("failed to get crc max: %w", err)
	}
	if useCodecv0 {
		metrics.L1CommitCalldataSize, err = codecv0.EstimateChunkL1CommitCalldataSize(chunk)
		if err != nil {
			return metrics, fmt.Errorf("failed to estimate chunk L1 commit calldata size: %w", err)
		}
		metrics.L1CommitGas, err = codecv0.EstimateChunkL1CommitGas(chunk)
		if err != nil {
			return metrics, fmt.Errorf("failed to estimate chunk L1 commit gas: %w", err)
		}
	} else {
		metrics.L1CommitBlobSize, err = codecv1.EstimateChunkL1CommitBlobSize(chunk)
		if err != nil {
			return metrics, fmt.Errorf("failed to estimate chunk L1 commit blob size: %w", err)
		}
	}
	return metrics, nil
}

// BatchMetrics indicates the metrics for proposing a batch.
type BatchMetrics struct {
	// common metrics
	NumChunks           uint64
	FirstBlockTimestamp uint64

	// codecv0 metrics, default 0 for codecv1
	L1CommitCalldataSize uint64
	L1CommitGas          uint64

	// codecv1 metrics, default 0 for codecv0
	L1CommitBlobSize uint64
}

// CalculateBatchMetrics calculates batch metrics.
func CalculateBatchMetrics(batch *encoding.Batch, useCodecv0 bool) (*BatchMetrics, error) {
	var err error
	metrics := &BatchMetrics{}
	metrics.NumChunks = uint64(len(batch.Chunks))
	metrics.FirstBlockTimestamp = batch.Chunks[0].Blocks[0].Header.Time
	if useCodecv0 {
		metrics.L1CommitGas, err = codecv0.EstimateBatchL1CommitGas(batch)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate batch L1 commit gas: %w", err)
		}
		metrics.L1CommitCalldataSize, err = codecv0.EstimateBatchL1CommitCalldataSize(batch)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate batch L1 commit calldata size: %w", err)
		}
	} else {
		metrics.L1CommitBlobSize, err = codecv1.EstimateBatchL1CommitBlobSize(batch)
		if err != nil {
			return metrics, fmt.Errorf("failed to estimate chunk L1 commit blob size: %w", err)
		}
	}
	return metrics, nil
}
