package viper

import (
	"bytes"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/log"

	config "scroll-tech/common/apollo"
)

// Viper : viper config.
type Viper struct {
	isRoot bool

	configType string
	configFile string

	data sync.Map
}

// New : new a empty viper config.
func New() *Viper {
	return &Viper{
		isRoot: true,
	}
}

// NewViper : new a viper config with local and remote config path.
func NewViper(file string, remoteCfg string) (*Viper, error) {
	vp := New()
	vp.SetConfigFile(file)
	err := vp.ReadInFile()
	if err != nil {
		return nil, err
	}

	if remoteCfg != "" {
		// use apollo.
		log.Info("Apollo remote config", "config name", remoteCfg)
		go syncApolloRemoteConfig(remoteCfg, vp)
	}

	return vp, nil
}

func syncApolloRemoteConfig(remoteCfg string, vp *Viper) {
	agolloClient := config.MustInitApollo()

	for {
		config := make(map[string]interface{})
		cfgStr := agolloClient.GetStringValue(remoteCfg, "")
		if err := vp.unmarshal(bytes.NewReader([]byte(cfgStr)), config); err != nil {
			log.Error("Unmarshal apollo config fail", "err", err, "config", cfgStr)
			<-time.After(time.Second * 3)
			continue
		}
		vp.flush(config)
		<-time.After(time.Second * 3)
	}
}

// WriteConfigAs : writes current configuration to a given filename.
// TODO: implement WriteConfigAs
func (v *Viper) WriteConfigAs(filename string) error {
	return nil
}
