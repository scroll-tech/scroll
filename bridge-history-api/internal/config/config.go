package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"scroll-tech/common/database"
)

// LayerConfig is the configuration of Layer1/Layer2
type LayerConfig struct {
	Confirmation             uint64 `json:"confirmation"`
	Endpoint                 string `json:"endpoint"`
	StartHeight              uint64 `json:"startHeight"` // Can only be configured to contract deployment height, otherwise in the current implementation, the message proof could not be successfully updated.
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
	ERC721GatewayAddr        string `json:"ERC721GatewayAddr"`
	ERC1155GatewayAddr       string `json:"ERC1155GatewayAddr"`
	ScrollChainAddr          string `json:"ScrollChainAddr"`
	GatewayRouterAddr        string `json:"GatewayRouterAddr"`
	MessageQueueAddr         string `json:"MessageQueueAddr"`
}

// RedisConfig redis config
type RedisConfig struct {
	Address  string `json:"address"`
	Username string `json:"username"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	Local    bool   `json:"local"`
}

// Config is the configuration of the bridge history backend
type Config struct {
	L1    *LayerConfig     `json:"L1"`
	L2    *LayerConfig     `json:"L2"`
	DB    *database.Config `json:"db"`
	Redis *RedisConfig     `json:"redis"`
}

// NewConfig returns a new instance of Config.
func NewConfig(file string) (*Config, error) {
	buf, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = json.Unmarshal(buf, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
