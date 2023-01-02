package viper

import (
	"bytes"
	"strings"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/spf13/viper"
	originVP "github.com/spf13/viper"

	config "scroll-tech/common/apollo"
)

// Viper : viper config.
type Viper struct {
	path string
	root *Viper

	subVps map[string]*Viper

	mu sync.RWMutex
	vp *viper.Viper
}

// NewViper : new a viper config instance.
func NewViper(file string, remoteCfg string) (*Viper, error) {
	vp := viper.New()
	vp.SetConfigFile(file)
	err := vp.ReadInConfig()
	if err != nil {
		return nil, err
	}
	root := &Viper{
		// Get the root viper.
		vp:     vp,
		subVps: make(map[string]*Viper),
	}
	root.root = root

	if remoteCfg != "" {
		// use apollo.
		go flushApolloRemoteConfig(remoteCfg, root)
	}

	return root, nil
}

func flushApolloRemoteConfig(remoteCfg string, vp *Viper) {
	agolloClient := config.MustInitApollo()

	for {
		origin := originVP.New()
		origin.SetConfigType("json")
		cfgStr := agolloClient.GetStringValue(remoteCfg, "")
		err := origin.ReadConfig(bytes.NewBuffer([]byte(cfgStr)))
		if err != nil {
			log.Error("ReadConfig from apollo fail", "err", err)
			<-time.After(time.Second * 3)
			continue
		}
		vp.Flush(origin)
		<-time.After(time.Second * 3)
	}
}

// NewEmptyViper : new a empty viper config instance.
func NewEmptyViper() *Viper {
	root := &Viper{
		// Get the root viper.
		vp:     viper.New(),
		subVps: make(map[string]*Viper),
	}
	root.root = root
	return root
}

// Sub : get a viper sub config.
func (v *Viper) Sub(key string) *Viper {
	var (
		path = v.path
		sub  = v
	)
	for _, s := range strings.Split(key, ".") {
		path = absolutePath(path, s)
		if vp := sub.subVps[path]; vp != nil {
			sub = vp
		} else {
			vip := v.root.vp.Sub(path)
			if vip == nil {
				return nil
			}
			sub.subVps[path] = &Viper{
				path:   path,
				vp:     vip,
				root:   v.root,
				subVps: make(map[string]*Viper),
			}
			sub = sub.subVps[path]
		}
	}
	return sub
}

// Set : set viper config recursively.
func (v *Viper) Set(key string, val interface{}) {
	var sub = v
	if idx := strings.LastIndex(key, "."); idx >= 0 {
		path := absolutePath(v.path, key[:idx])
		sub = v.root.Sub(path)
		if sub == nil {
			log.Error("Invalid path while updating viper configuration", "path", path)
			return
		}
		key = key[idx+1:]
	}
	sub.mu.Lock()
	defer sub.mu.Unlock()
	sub.vp.Set(key, val)
}

// Unmarshal unmarshals the config into a Struct. Make sure that the tags
// on the fields of the structure are properly set.
func (v *Viper) Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return v.vp.Unmarshal(rawVal, opts...)
}

func absolutePath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}
