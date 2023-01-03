package viper

import (
	"path/filepath"
)

func getConfigType(file string, tp string) string {
	ext := filepath.Ext(file)
	if len(ext) > 1 {
		return ext[1:]
	}
	return tp
}
