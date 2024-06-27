package forks

import (
	"math"
	"math/big"
	"sort"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/params"
)

// CollectSortedForkHeights returns a sorted set of block numbers that one or more forks are activated on
func CollectSortedForkHeights(config *params.ChainConfig) ([]uint64, map[uint64]bool, map[string]uint64) {
	type nameFork struct {
		name  string
		block *big.Int
	}

	forkHeightNameMap := make(map[uint64]string)

	for _, fork := range []nameFork{
		{name: "homestead", block: config.HomesteadBlock},
		{name: "daoFork", block: config.DAOForkBlock},
		{name: "eip150", block: config.EIP150Block},
		{name: "eip155", block: config.EIP155Block},
		{name: "eip158", block: config.EIP158Block},
		{name: "byzantium", block: config.ByzantiumBlock},
		{name: "constantinople", block: config.ConstantinopleBlock},
		{name: "petersburg", block: config.PetersburgBlock},
		{name: "istanbul", block: config.IstanbulBlock},
		{name: "muirGlacier", block: config.MuirGlacierBlock},
		{name: "berlin", block: config.BerlinBlock},
		{name: "london", block: config.LondonBlock},
		{name: "arrowGlacier", block: config.ArrowGlacierBlock},
		{name: "archimedes", block: config.ArchimedesBlock},
		{name: "shanghai", block: config.ShanghaiBlock},
		{name: "bernoulli", block: config.BernoulliBlock},
		{name: "curie", block: config.CurieBlock},
	} {
		if fork.block == nil {
			continue
		}
		height := fork.block.Uint64()

		// only keep latest fork for at each height, discard the rest
		forkHeightNameMap[height] = fork.name
	}

	forkHeightsMap := make(map[uint64]bool)
	forkNameHeightMap := make(map[string]uint64)

	for height, name := range forkHeightNameMap {
		forkHeightsMap[height] = true
		forkNameHeightMap[name] = height
	}

	var forkHeights []uint64
	for height := range forkHeightsMap {
		forkHeights = append(forkHeights, height)
	}
	sort.Slice(forkHeights, func(i, j int) bool {
		return forkHeights[i] < forkHeights[j]
	})
	return forkHeights, forkHeightsMap, forkNameHeightMap
}

// BlockRange returns the block range of the hard fork
// Need ensure the forkHeights is incremental
func BlockRange(currentForkHeight uint64, forkHeights []uint64) (from, to uint64) {
	to = math.MaxInt64
	for _, height := range forkHeights {
		if currentForkHeight < height {
			to = height
			return
		}
		from = height
	}
	return
}

// GetHardforkName returns the name of the hardfork active at the given block height and timestamp.
// It checks the chain configuration to determine which hardfork is active.
func GetHardforkName(config *params.ChainConfig, blockHeight, blockTimestamp uint64) string {
	if !config.IsBernoulli(new(big.Int).SetUint64(blockHeight)) {
		return "homestead"
	} else if !config.IsCurie(new(big.Int).SetUint64(blockHeight)) {
		return "bernoulli"
	} else if !config.IsDarwin(blockTimestamp) {
		return "curie"
	} else {
		return "darwin"
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
	} else {
		return encoding.CodecV3
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
	} else {
		return 45
	}
}
