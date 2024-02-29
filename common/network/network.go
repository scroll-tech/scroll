package network

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/scroll-tech/go-ethereum/params"
)

// Network represents a known network
type Network string

var (
	// Mainnet network
	Mainnet Network = "mainnet"
	// Sepolia network
	Sepolia Network = "sepolia"
	// Alpha network
	Alpha Network = "alpha"
)

// IsKnown returns if the network is indeed known
func (n Network) IsKnown() bool {
	return n == Mainnet || n == Sepolia || n == Alpha
}

// GenesisConfig returns the chain config of a known network
func (n Network) GenesisConfig() *params.ChainConfig {
	switch n {
	case Mainnet:
		return params.ScrollMainnetChainConfig
	case Sepolia:
		return params.ScrollSepoliaChainConfig
	case Alpha:
		return params.ScrollAlphaChainConfig
	default:
		panic(fmt.Sprintf("unknown network (%s), check configuration", n))
	}
}

// CollectSortedForkHeights returns a sorted set of block numbers that one or more forks are activated on
func CollectSortedForkHeights(config *params.ChainConfig) []uint64 {
	var forkHeights []uint64
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
			forkHeights = append(forkHeights, height)
		}
	}
	slices.Sort(forkHeights)
	return slices.Compact(forkHeights) // Remove duplicates
}
