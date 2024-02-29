package network

import (
	"fmt"

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
