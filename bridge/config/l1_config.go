package config

import (
	"scroll-tech/bridge/utils"

	"github.com/scroll-tech/go-ethereum/common"
)

// L1Config loads l1eth configuration items.
type L1Config struct {
	// Confirmations block height confirmations number.
	Confirmations utils.ConfirmationParams `json:"confirmations"`
	// l1 eth node url.
	Endpoint string `json:"endpoint"`
	// The start height to sync event from layer 1
	StartHeight uint64 `json:"start_height"`
	// The messenger contract address deployed on layer 1 chain.
	L1MessengerAddress common.Address `json:"l1_messenger_address"`
	// The rollup contract address deployed on layer 1 chain.
	RollupContractAddress common.Address `json:"rollup_contract_address"`
	// The relayer config
	RelayerConfig *RelayerConfig `json:"relayer_config"`
}
