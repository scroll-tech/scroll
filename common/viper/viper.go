package viper

import (
	"sync"
)

type Viper struct {
	isRoot bool

	configType string
	configFile string

	data sync.Map
}

func New() *Viper {
	return &Viper{
		isRoot: true,
	}
}
