package viper

import (
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/spf13/viper"
	"strings"
	"sync"
)

var (
	root *Viper
)

func init() {
	root = &Viper{
		path: "",
		// Get the root viper.
		Viper:  viper.GetViper(),
		subVps: make(map[string]*Viper),
	}
}

type Viper struct {
	path string

	subVps map[string]*Viper

	mu sync.RWMutex
	*viper.Viper
}

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
			vip := root.Viper.Sub(path)
			if vip == nil {
				return nil
			}
			sub.subVps[path] = &Viper{
				path:   path,
				Viper:  vip,
				subVps: make(map[string]*Viper),
			}
			sub = sub.subVps[path]
		}
	}
	return sub
}

func (v *Viper) Set(key string, val interface{}) {
	var sub = v
	if idx := strings.LastIndex(key, "."); idx >= 0 {
		path := absolutePath(v.path, key[:idx])
		sub = root.Sub(path)
		if sub == nil {
			log.Error("don't exist the sub viper", "path", path)
			return
		}
		key = key[idx+1:]
	}
	sub.mu.Lock()
	sub.Viper.Set(key, val)
	sub.mu.Unlock()
}

// Unmarshal unmarshals the config into a Struct. Make sure that the tags
// on the fields of the structure are properly set.
func (v *Viper) Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return v.Viper.Unmarshal(rawVal, opts...)
}

func absolutePath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}
