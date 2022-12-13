package config

import (
	"github.com/scroll-tech/go-ethereum/common"
)

// L2Config loads l2geth configuration items.
type L2Config struct {
	// l2geth node url.
	Endpoint string `json:"endpoint"`
	// The messenger contract address deployed on layer 2 chain.
	L2MessengerAddress common.Address `json:"l2_messenger_address,omitempty"`
	// The relayer config
	RelayerConfig *RelayerConfig `json:"relayer_config"`
}
