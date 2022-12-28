package viper

import (
	"github.com/scroll-tech/go-ethereum/log"
	"strings"

	"github.com/spf13/viper"
)

// GetViper gets the global Viper instance.
func GetViper() *Viper {
	return root
}

// SetConfigFile explicitly defines the absolutePath, name and extension of the config file.
// Viper will use this and not check any of the config paths.
func SetConfigFile(in string) { root.SetConfigFile(in) }

// ReadInConfig will discover and load the configuration file from disk
// and key/value stores, searching in one of the defined paths.
func ReadInConfig() error { return root.ReadInConfig() }

// Sub returns new Viper instance representing a sub tree of this instance.
// Sub is case-insensitive for a key.
func Sub(key string) *Viper { return root.Sub(key) }

// Set sets the value for the key in the override register.
// Set is case-insensitive for a key.
// Will be used instead of values obtained via
// flags, config file, ENV, default, or key/value store.
func Set(key string, value interface{}) {
	idx := strings.LastIndex(key, ".")
	if idx > 0 {
		if sub := root.Sub(key[:idx]); sub != nil {
			sub.Set(key[idx+1:], value)
		} else {
			log.Error("don't exist the sub viper", "path", key[:idx])
		}
	} else {
		root.Set(key, value)
	}
}

// Flush deep copy all values from vp to root.
func Flush(vp *viper.Viper) {
	subs := make(map[string]*Viper)
	for _, str := range vp.AllKeys() {
		idx := strings.LastIndex(str, ".")
		if idx < 0 {
			continue
		}
		path := str[:idx]
		// If don't exist get it.
		if _, exist := subs[path]; !exist {
			subs[path] = root.Sub(path)
		}
		if subs[path] != nil {
			subs[path].Set(str[idx+1:], vp.Get(str))
		}
	}
}
