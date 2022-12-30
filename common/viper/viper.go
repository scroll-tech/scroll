package viper

import (
	"strings"
	"sync"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/spf13/viper"
)

type Viper struct {
	path string
	root *Viper

	subVps map[string]*Viper

	mu sync.RWMutex
	vp *viper.Viper
}

func NewViper(file string, use_apollo bool) (*Viper, error) {
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
	return root, nil
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

func (v *Viper) Set(key string, val interface{}) {
	var sub = v
	if idx := strings.LastIndex(key, "."); idx >= 0 {
		path := absolutePath(v.path, key[:idx])
		sub = v.root.Sub(path)
		if sub == nil {
			log.Error("don't exist the sub viper", "path", path)
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
