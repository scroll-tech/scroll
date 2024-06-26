package watcher

import (
	"math/big"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/params"
)

const contractEventsBlocksFetchLimit = int64(10)

const maxBlobSize = uint64(131072)

func getCodecVersion(chainCfg *params.ChainConfig, blockHeight, blockTimestamp uint64) encoding.CodecVersion {
	if !chainCfg.IsBernoulli(new(big.Int).SetUint64(blockHeight)) {
		return encoding.CodecV0
	} else if !chainCfg.IsCurie(new(big.Int).SetUint64(blockHeight)) {
		return encoding.CodecV1
	} else if !chainCfg.IsDarwin(blockTimestamp) {
		return encoding.CodecV2
	} else {
		return encoding.CodecV3
	}
}

func getMaxChunksPerBatch(chainCfg *params.ChainConfig, blockHeight, blockTimestamp uint64) uint64 {
	if !chainCfg.IsBernoulli(new(big.Int).SetUint64(blockHeight)) {
		return 15
	} else if !chainCfg.IsCurie(new(big.Int).SetUint64(blockHeight)) {
		return 15
	} else if !chainCfg.IsDarwin(blockTimestamp) {
		return 45
	} else {
		return 45
	}
}
