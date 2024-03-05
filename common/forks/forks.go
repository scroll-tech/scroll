package forks

import (
	"math/big"
	"sort"

	"github.com/scroll-tech/go-ethereum/params"
)

// CollectSortedForkHeights returns a sorted set of block numbers that one or more forks are activated on
func CollectSortedForkHeights(config *params.ChainConfig) ([]uint64, map[uint64]bool) {
	forkHeightsMap := make(map[uint64]bool)
	for _, fork := range []*big.Int{
		config.HomesteadBlock,
		config.DAOForkBlock,
		config.EIP150Block,
		config.EIP155Block,
		config.EIP158Block,
		config.ByzantiumBlock,
		config.ConstantinopleBlock,
		config.PetersburgBlock,
		config.IstanbulBlock,
		config.MuirGlacierBlock,
		config.BerlinBlock,
		config.LondonBlock,
		config.ArrowGlacierBlock,
		config.ArchimedesBlock,
		config.ShanghaiBlock,
	} {
		if fork == nil {
			continue
		} else if height := fork.Uint64(); height != 0 {
			forkHeightsMap[height] = true
		}
	}

	var forkHeights []uint64
	for height := range forkHeightsMap {
		forkHeights = append(forkHeights, height)
	}
	sort.Slice(forkHeights, func(i, j int) bool {
		return forkHeights[i] < forkHeights[j]
	})
	return forkHeights, forkHeightsMap
}
