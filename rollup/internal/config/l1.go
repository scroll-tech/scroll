package config

// L1Config loads l1eth configuration items.
type L1Config struct {
	// l1 eth node url.
	Endpoint string `json:"endpoint"`
	// The start height to sync event from layer 1
	StartHeight uint64 `json:"start_height"`
	// The relayer config
	RelayerConfig *RelayerConfig `json:"relayer_config"`
}
