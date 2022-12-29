package viper

import (
	"strings"

	"github.com/spf13/viper"
)

// Flush deep copy all values from vp to root.
func (v *Viper) Flush(vp *viper.Viper) {
	subs := make(map[string]*Viper)
	for _, str := range vp.AllKeys() {
		idx := strings.LastIndex(str, ".")
		if idx < 0 {
			continue
		}
		path := str[:idx]
		// If don't exist get it.
		if _, exist := subs[path]; !exist {
			subs[path] = v.root.Sub(path)
		}
		if subs[path] != nil {
			subs[path].Set(str[idx+1:], vp.Get(str))
		}
	}
}
