package viper

import (
	"bytes"
	"fmt"
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

func (v *Viper) export() map[string]interface{} {
	c := make(map[string]interface{})
	v.data.Range(func(key, value any) bool {
		if nd, ok := value.(*Viper); ok {
			c[key.(string)] = nd.export()
		} else {
			c[key.(string)] = value
		}
		return true
	})
	return c
}

func (v *Viper) flush(m map[string]interface{}) {
	for key, val := range m {
		switch val.(type) {
		case map[interface{}]interface{}, map[string]interface{}:
			vp := v.Sub(key)
			if vp == nil {
				vp = &Viper{}
				v.data.Store(key, vp)
			}
			mp, ok := val.(map[string]interface{})
			if !ok {
				mp = make(map[string]interface{})
				for k, v := range val.(map[interface{}]interface{}) {
					mp[fmt.Sprintf("%v", k)] = v
				}
			}
			vp.flush(mp)
		default:
			v.data.Store(key, val)
		}
	}
}

func syncApolloRemoteConfig(remoteCfg string, vp *Viper) {
	agolloClient := config.MustInitApollo()

	for {
		cfgStr := agolloClient.GetStringValue(remoteCfg, "invalid")
		if cfgStr == "invalid" {
			log.Error("GetStringValue fail", "remoteCfg", remoteCfg)
			<-time.After(time.Second * 10)
			continue
		}
		if err := vp.ReadConfig(bytes.NewReader([]byte(cfgStr))); err != nil {
			log.Error("ReadConfig fail", "remoteCfg", remoteCfg, "config", cfgStr, "err", err)
			<-time.After(time.Second * 10)
			continue
		}
		<-time.After(time.Second * 10)
	}
}
