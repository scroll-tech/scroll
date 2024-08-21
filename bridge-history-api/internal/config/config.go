package config

import (
	"scroll-tech/common/database"

	"github.com/spf13/viper"
)

// FetcherConfig is the configuration of Layer1 or Layer2 fetcher.
type FetcherConfig struct {
	Confirmation             uint64 `json:"confirmation"`
	Endpoint                 string `json:"endpoint"`
	StartHeight              uint64 `json:"startHeight"` // Can only be configured to contract deployment height, message proof should be updated from the very beginning.
	BlockTime                int64  `json:"blockTime"`
	FetchLimit               uint64 `json:"fetchLimit"`
	MessengerAddr            string `json:"MessengerAddr"`
	ETHGatewayAddr           string `json:"ETHGatewayAddr"`
	StandardERC20GatewayAddr string `json:"StandardERC20GatewayAddr"`
	CustomERC20GatewayAddr   string `json:"CustomERC20GatewayAddr"`
	WETHGatewayAddr          string `json:"WETHGatewayAddr"`
	DAIGatewayAddr           string `json:"DAIGatewayAddr"`
	USDCGatewayAddr          string `json:"USDCGatewayAddr"`
	LIDOGatewayAddr          string `json:"LIDOGatewayAddr"`
	PufferGatewayAddr        string `json:"PufferGatewayAddr"`
	ERC721GatewayAddr        string `json:"ERC721GatewayAddr"`
	ERC1155GatewayAddr       string `json:"ERC1155GatewayAddr"`
	ScrollChainAddr          string `json:"ScrollChainAddr"`
	GatewayRouterAddr        string `json:"GatewayRouterAddr"`
	MessageQueueAddr         string `json:"MessageQueueAddr"`
	BatchBridgeGatewayAddr   string `json:"BatchBridgeGatewayAddr"`
	GasTokenGatewayAddr      string `json:"GasTokenGatewayAddr"`
	WrappedTokenGatewayAddr  string `json:"WrappedTokenGatewayAddr"`
}

// RedisConfig redis config
type RedisConfig struct {
	Address       string `json:"address"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	DB            int    `json:"db"`
	Local         bool   `json:"local"`
	MinIdleConns  int    `json:"minIdleConns"`
	ReadTimeoutMs int    `json:"readTimeoutMs"`
}

// Config is the configuration of the bridge history backend
type Config struct {
	L1    *FetcherConfig   `json:"L1"`
	L2    *FetcherConfig   `json:"L2"`
	DB    *database.Config `json:"db"`
	Redis *RedisConfig     `json:"redis"`
}

// NewConfig returns a new instance of Config.
func NewConfig(file string) (*Config, error) {
	viper.SetConfigFile(file)
	viper.SetEnvPrefix("SCROLL_BRIDGE_HISTORY")
	viper.AutomaticEnv()
	viper.SetConfigType("json")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
