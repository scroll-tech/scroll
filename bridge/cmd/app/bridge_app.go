package app

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/modern-go/reflect2"

	bridgeConfig "scroll-tech/bridge/config"
	"scroll-tech/common/cmd"
	"scroll-tech/common/docker"
)

type BridgeApp struct {
	base *docker.DockerApp

	Config     *bridgeConfig.Config
	originFile string
	bridgeFile string

	name string
	args []string
	docker.AppAPI
}

func NewBridgeApp(base *docker.DockerApp, file string) *BridgeApp {
	bridgeFile := fmt.Sprintf("/tmp/%d_bridge-config.json", base.Timestamp)
	bridgeApp := &BridgeApp{
		base:       base,
		name:       "bridge-test",
		originFile: file,
		bridgeFile: bridgeFile,
		args:       []string{"--log.debug", "--config", bridgeFile},
	}
	if err := bridgeApp.MockBridgeConfig(); err != nil {
		panic(err)
	}
	return bridgeApp
}

func (b *BridgeApp) RunApp(t *testing.T, args ...string) {
	b.AppAPI = cmd.NewCmd("bridge-test", append(b.args, args...)...)
	b.AppAPI.RunApp(func() bool { return b.AppAPI.WaitResult(t, time.Second*20, "Start bridge successfully") })
}

func (b *BridgeApp) Free() {
	if !reflect2.IsNil(b.AppAPI) {
		b.AppAPI.WaitExit()
		_ = os.Remove(b.bridgeFile)
	}
}

func (b *BridgeApp) MockBridgeConfig() error {
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

	// Store changed bridge config into a temp file.
	data, err := json.Marshal(b.Config)
	if err != nil {
		return err
	}
	return os.WriteFile(b.bridgeFile, data, 0644)
}
