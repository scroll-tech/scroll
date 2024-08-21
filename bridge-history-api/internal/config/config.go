package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"scroll-tech/common/database"
)

// FetcherConfig is the configuration of Layer1 or Layer2 fetcher.
type FetcherConfig struct {
	Confirmation             uint64 `json:"confirmation"`
	Endpoint                 string `json:"endpoint"`
	StartHeight              uint64 `json:"startHeight"`
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
	buf, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = json.Unmarshal(buf, cfg)
	if err != nil {
		return nil, err
	}

	// Override config with environment variables
	err = overrideConfigWithEnv(cfg, "SCROLL_BRIDGE_HISTORY")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// overrideConfigWithEnv recursively overrides config values with environment variables
func overrideConfigWithEnv(cfg interface{}, prefix string) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil
	}
	v = v.Elem()

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		tag := field.Tag.Get("json")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}

		envKey := prefix + "_" + strings.ToUpper(tag)

		switch fieldValue.Kind() {
		case reflect.Ptr:
			if !fieldValue.IsNil() {
				err := overrideConfigWithEnv(fieldValue.Interface(), envKey)
				if err != nil {
					return err
				}
			}
		case reflect.Struct:
			err := overrideConfigWithEnv(fieldValue.Addr().Interface(), envKey)
			if err != nil {
				return err
			}
		default:
			if envValue, exists := os.LookupEnv(envKey); exists {
				err := setField(fieldValue, envValue)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// setField sets the value of a field based on the environment variable value
func setField(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)
	default:
		return fmt.Errorf("unsupported type: %v", field.Kind())
	}
	return nil
}