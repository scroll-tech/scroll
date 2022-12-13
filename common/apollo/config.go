package config

import (
	"fmt"

	"github.com/scroll-tech/go-ethereum/log"

	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
)

// AgolloClient is client to fetch Apollo config
var AgolloClient agollo.Client

// MustInitApollo init apollo client
func MustInitApollo() {
	agollo.SetLogger(&DefaultLogger{})
	var err error
	c := &config.AppConfig{
		AppID:          "scroll-apollo-server",
		Cluster:        "default",
		IP:             "http://monitor.scroll.tech:8080",
		NamespaceName:  "application",
		IsBackupConfig: true,
	}

	AgolloClient, err = agollo.StartWithConfig(func() (*config.AppConfig, error) {
		return c, nil
	})
	if err != nil {
		log.Crit("MustInitApollo fail", "error: ", err)
		panic(err)
	}
	PrintConfig(AgolloClient)
	log.Info("MustInitApollo success")
}

// PrintConfig print remote config
func PrintConfig(client agollo.Client) {
	cache := client.GetDefaultConfigCache()
	count := 0
	cache.Range(func(key, value interface{}) bool {
		log.Info("PrintConfig", "key : ", key, ", value :", value)
		count++
		return true
	})
}

// DefaultLogger is the logger of agollo
type DefaultLogger struct {
}

// Debugf is the Debugf logger of agollo
func (logger *DefaultLogger) Debugf(format string, params ...interface{}) {
	fmt.Printf(format+"\n", params...)
}

// Infof is the Infof logger of agollo
func (logger *DefaultLogger) Infof(format string, params ...interface{}) {
	fmt.Printf(format+"\n", params...)
}

// Warnf is the Warnf logger of agollo
func (logger *DefaultLogger) Warnf(format string, params ...interface{}) {
	fmt.Printf(format+"\n", params...)
}

// Errorf is the Errorf logger of agollo
func (logger *DefaultLogger) Errorf(format string, params ...interface{}) {
	fmt.Printf(format+"\n", params...)
}

// Debug is the Debug logger of agollo
func (logger *DefaultLogger) Debug(v ...interface{}) {}

// Info is the Info logger of agollo
func (logger *DefaultLogger) Info(v ...interface{}) {}

// Warn is the Warn logger of agollo
func (logger *DefaultLogger) Warn(v ...interface{}) {}

// Error is the Error logger of agollo
func (logger *DefaultLogger) Error(v ...interface{}) {}
