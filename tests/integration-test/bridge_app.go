package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	bridgeConfig "scroll-tech/bridge/config"

	"scroll-tech/common/cmd"
)

type bridgeApp struct {
	base *dockerApp

	cfg        *bridgeConfig.Config
	bridgeFile string

	name string
	args []string
	appAPI
}

func newBridgeApp(base *dockerApp) *bridgeApp {
	bridgeFile := fmt.Sprintf("/tmp/%d_bridge-config.json", base.timestamp)
	return &bridgeApp{
		base:       base,
		name:       "bridge-test",
		bridgeFile: bridgeFile,
		args:       []string{"--log.debug", "--config", bridgeFile},
	}
}

func (b *bridgeApp) runApp(t *testing.T, args ...string) {
	if err := b.mockBridgeConfig(); err != nil {
		t.Fatal(err)
	}
	b.appAPI = cmd.NewCmd(t, b.name, append(b.args, args...)...)
	b.appAPI.RunApp(func() bool { return b.appAPI.WaitResult(time.Second*20, "Start bridge successfully") })
}

func (b *bridgeApp) free() {
	_ = os.Remove(b.bridgeFile)
}

func (b *bridgeApp) mockBridgeConfig() error {
	// Load origin bridge config file.
	if b.cfg == nil {
		cfg, err := bridgeConfig.NewConfig("../../bridge/config.json")
		if err != nil {
			return err
		}
		b.cfg = cfg
	}

	var base = b.base
	if base.l1gethImg != nil {
		b.cfg.L1Config.Endpoint = base.l1gethImg.Endpoint()
		b.cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = base.l1gethImg.Endpoint()
	}
	if base.l2gethImg != nil {
		b.cfg.L2Config.Endpoint = base.l2gethImg.Endpoint()
		b.cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = base.l2gethImg.Endpoint()
	}
	if base.dbImg != nil {
		b.cfg.DBConfig.DSN = base.dbImg.Endpoint()
	}

	// Store changed bridge config into a temp file.
	data, err := json.Marshal(b.cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(b.bridgeFile, data, 0644)
}
