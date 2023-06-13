package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DBConfig db config
type DBConfig struct {
	// data source name
	DSN        string `json:"dsn"`
	DriverName string `json:"driverName"`

	MaxOpenNum int `json:"maxOpenNum"`
	MaxIdleNum int `json:"maxIdleNum"`
}

type LayerConfig struct {
	Confirmation           uint64 `json:"confirmation"`
	Endpoint               string `json:"endpoint"`
	StartHeight            uint64 `json:"startHeight"`
	BlockTime              int64  `json:"blockTime"`
	MessengerAddr          string `json:"MessengerAddr"`
	ETHGatewayAddr         string `json:"ETHGatewayAddr"`
	WETHGatewayAddr        string `json:"WETHGatewayAddr"`
	StandardERC20Gateway   string `json:"StandardERC20Gateway"`
	ERC721GatewayAddr      string `json:"ERC721GatewayAddr"`
	ERC1155GatewayAddr     string `json:"ERC1155GatewayAddr"`
	CustomERC20GatewayAddr string `json:"CustomERC20GatewayAddr"`
}

type ServerConfig struct {
	HostPort string `json:"hostPort"`
}

// Config is the configuration of the bridge history backend
type Config struct {
	// chain config
	L1 *LayerConfig `json:"l1"`
	L2 *LayerConfig `json:"l2"`

	// data source name
	DB     *DBConfig     `json:"db"`
	Server *ServerConfig `json:"server"`
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
