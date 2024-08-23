package forks

import (
	"math/big"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/params"
)

// GetHardforkName returns the name of the hardfork active at the given block height and timestamp.
// It checks the chain configuration to determine which hardfork is active.
func GetHardforkName(config *params.ChainConfig, blockHeight, blockTimestamp uint64) string {
	if !config.IsBernoulli(new(big.Int).SetUint64(blockHeight)) {
		return "homestead"
	} else if !config.IsCurie(new(big.Int).SetUint64(blockHeight)) {
		return "bernoulli"
	} else if !config.IsDarwin(blockTimestamp) {
		return "curie"
	} else if !config.IsDarwinV2(blockTimestamp) {
		return "darwin"
	} else {
		return "darwinV2"
	}
}

// GetCodecVersion returns the encoding codec version for the given block height and timestamp.
// It determines the appropriate codec version based on the active hardfork.
func GetCodecVersion(config *params.ChainConfig, blockHeight, blockTimestamp uint64) encoding.CodecVersion {
	if !config.IsBernoulli(new(big.Int).SetUint64(blockHeight)) {
		return encoding.CodecV0
	} else if !config.IsCurie(new(big.Int).SetUint64(blockHeight)) {
		return encoding.CodecV1
	} else if !config.IsDarwin(blockTimestamp) {
		return encoding.CodecV2
	} else if !config.IsDarwinV2(blockTimestamp) {
		return encoding.CodecV3
	} else {
		return encoding.CodecV4
	}
}

// GetMaxChunksPerBatch returns the maximum number of chunks allowed per batch for the given block height and timestamp.
// This value may change depending on the active hardfork.
func GetMaxChunksPerBatch(config *params.ChainConfig, blockHeight, blockTimestamp uint64) uint64 {
	if !config.IsBernoulli(new(big.Int).SetUint64(blockHeight)) {
		return 15
	} else if !config.IsCurie(new(big.Int).SetUint64(blockHeight)) {
		return 15
	} else if !config.IsDarwin(blockTimestamp) {
		return 45
	} else if !config.IsDarwinV2(blockTimestamp) {
		return 45
	} else {
		return 45
	}
}
