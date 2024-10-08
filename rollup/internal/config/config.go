package config

import (
	"fmt"
	"reflect"
	"scroll-tech/common/database"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/spf13/viper"
)

// Config load configuration items.
type Config struct {
	L1Config *L1Config        `json:"l1_config"`
	L2Config *L2Config        `json:"l2_config"`
	DBConfig *database.Config `json:"db_config"`
}

// NewConfig returns a new instance of Config.
func NewConfig(file string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(file)
	v.SetConfigType("json")

	v.SetEnvPrefix("SCROLL_ROLLUP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &Config{}

	decoderConfig := &mapstructure.DecoderConfig{
		TagName: "json",
		Result:  cfg,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
				if to == reflect.TypeOf(rpc.BlockNumber(0)) {
					var bn rpc.BlockNumber
					err := bn.UnmarshalJSON([]byte(fmt.Sprintf("%v", data)))
					if err != nil {
						return nil, fmt.Errorf("invalid block number, data: %v, error: %v", data, err)
					}
					return bn, nil
				}

				if to == reflect.TypeOf(common.Address{}) {
					s, ok := data.(string)
					if !ok {
						return nil, fmt.Errorf("invalid address, data: %v", data)
					}
					return common.HexToAddress(s), nil
				}

				if to == reflect.TypeOf(common.Hash{}) {
					s, ok := data.(string)
					if !ok {
						return nil, fmt.Errorf("invalid hash, data: %v", data)
					}
					return common.HexToHash(s), nil
				}

				return data, nil
			},
		),
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(v.AllSettings()); err != nil {
		return nil, err
	}

	if err := v.Unmarshal(cfg, viper.DecodeHook(decoderConfig.DecodeHook)); err != nil {
		return nil, err
	}

	return cfg, nil
}
