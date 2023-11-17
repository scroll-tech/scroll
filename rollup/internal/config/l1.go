package config

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/rpc"
)

// L1Config loads l1eth configuration items.
type L1Config struct {
	// Confirmations block height confirmations number.
	Confirmations rpc.BlockNumber `json:"confirmations"`
	// l1 eth node url.
	Endpoint string `json:"endpoint"`
	// The start height to sync event from layer 1
	StartHeight uint64 `json:"start_height"`
	// The L1MessageQueue contract address deployed on layer 1 chain.
	L1MessageQueueAddress common.Address `json:"l1_message_queue_address"`
	// The ScrollChain contract address deployed on layer 1 chain.
	ScrollChainContractAddress common.Address `json:"scroll_chain_address"`
	// The relayer config
	RelayerConfig *RelayerConfig `json:"relayer_config"`
	// The L1ViewOracle contract address deployed on layer 1 chain.
	L1ViewOracleAddress common.Address `json:"l1_view_oracle_address"`
}
