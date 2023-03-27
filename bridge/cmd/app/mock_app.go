package app

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/docker"
	"scroll-tech/common/utils"

	bridgeConfig "scroll-tech/bridge/config"
)

// BridgeApp bridge-test client manager.
type BridgeApp struct {
	Config *bridgeConfig.Config

	base *docker.App

	originFile string
	bridgeFile string

	name string
	args []string
	docker.AppAPI
}

// NewBridgeApp return a new bridgeApp manager.
func NewBridgeApp(base *docker.App, file string) *BridgeApp {
	bridgeFile := fmt.Sprintf("/tmp/%d_bridge-config.json", base.Timestamp)
	bridgeApp := &BridgeApp{
		base:       base,
		name:       "bridge-test",
		originFile: file,
		bridgeFile: bridgeFile,
		args:       []string{"--log.debug", "--config", bridgeFile},
	}
	return bridgeApp
}

// RunApp run bridge-test child process by multi parameters.
func (b *BridgeApp) RunApp(t *testing.T, args ...string) {
	b.AppAPI = cmd.NewCmd("bridge-test", append(b.args, args...)...)
	b.AppAPI.RunApp(func() bool { return b.AppAPI.WaitResult(t, time.Second*20, "Start bridge successfully") })
}

// Free stop and release bridge-test.
func (b *BridgeApp) Free() {
	if !utils.IsNil(b.AppAPI) {
		b.AppAPI.WaitExit()
		_ = os.Remove(b.bridgeFile)
	}
}

// MockBridgeConfig creates a new bridge config.
func (b *BridgeApp) MockBridgeConfig(store bool) error {
	base := b.base
	// Load origin bridge config file.
	if b.Config == nil {
		cfg, err := bridgeConfig.NewConfig(b.originFile)
		if err != nil {
			return err
		}
		b.Config = cfg
	}

	b.Config.L1Config.Endpoint = base.L1gethImg.Endpoint()
	b.Config.L2Config.RelayerConfig.SenderConfig.Endpoint = base.L1gethImg.Endpoint()
	b.Config.L2Config.Endpoint = base.L2gethImg.Endpoint()
	b.Config.L1Config.RelayerConfig.SenderConfig.Endpoint = base.L2gethImg.Endpoint()
	b.Config.DBConfig.DSN = base.DBImg.Endpoint()

	if !store {
		return nil
	}
	// Store changed bridge config into a temp file.
	data, err := json.Marshal(b.Config)
	if err != nil {
		return err
	}
	return os.WriteFile(b.bridgeFile, data, 0600)
}
