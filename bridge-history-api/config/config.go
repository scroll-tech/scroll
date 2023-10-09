package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// BatchInfoFetcherConfig is the configuration of BatchInfoFetcher
type BatchInfoFetcherConfig struct {
	BatchIndexStartBlock uint64 `json:"batchIndexStartBlock"`
	ScrollChainAddr      string `json:"ScrollChainAddr"`
}

// DBConfig db config
type DBConfig struct {
	// data source name
	DSN        string `json:"dsn"`
	DriverName string `json:"driverName"`

	MaxOpenNum int `json:"maxOpenNum"`
	MaxIdleNum int `json:"maxIdleNum"`
}

// LayerConfig is the configuration of Layer1/Layer2
type LayerConfig struct {
	Confirmation           uint64 `json:"confirmation"`
	Endpoint               string `json:"endpoint"`
	StartHeight            uint64 `json:"startHeight"`
	BlockTime              int64  `json:"blockTime"`
	MessengerAddr          string `json:"MessengerAddr"`
	ETHGatewayAddr         string `json:"ETHGatewayAddr"`
	WETHGatewayAddr        string `json:"WETHGatewayAddr"`
	USDCGatewayAddr        string `json:"USDCGatewayAddr"`
	LIDOGatewayAddr        string `json:"LIDOGatewayAddr"`
	DAIGatewayAddr         string `json:"DAIGatewayAddr"`
	StandardERC20Gateway   string `json:"StandardERC20Gateway"`
	ERC721GatewayAddr      string `json:"ERC721GatewayAddr"`
	ERC1155GatewayAddr     string `json:"ERC1155GatewayAddr"`
	CustomERC20GatewayAddr string `json:"CustomERC20GatewayAddr"`
}

// ServerConfig is the configuration of the bridge history backend server port
type ServerConfig struct {
	HostPort string `json:"hostPort"`
}

// Config is the configuration of the bridge history backend
type Config struct {
	// chain config
	L1 *LayerConfig `json:"l1"`
	L2 *LayerConfig `json:"l2"`

	// data source name
	DB               *DBConfig               `json:"db"`
	Server           *ServerConfig           `json:"server"`
	BatchInfoFetcher *BatchInfoFetcherConfig `json:"batchInfoFetcher"`
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
