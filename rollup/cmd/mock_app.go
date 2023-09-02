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

	"scroll-tech/bridge/internal/config"
)

// MockApp mockApp-test client manager.
type MockApp struct {
	Config *config.Config
	base   *docker.App

	mockApps map[utils.MockAppName]docker.AppAPI

	originFile string
	bridgeFile string

	args []string
}

// NewBridgeApp return a new bridgeApp manager, name mush be one them.
func NewBridgeApp(base *docker.App, file string) *MockApp {

	bridgeFile := fmt.Sprintf("/tmp/%d_bridge-config.json", base.Timestamp)
	bridgeApp := &MockApp{
		base:       base,
		mockApps:   make(map[utils.MockAppName]docker.AppAPI),
		originFile: file,
		bridgeFile: bridgeFile,
		args:       []string{"--log.debug", "--config", bridgeFile},
	}
	if err := bridgeApp.MockConfig(true); err != nil {
		panic(err)
	}
	return bridgeApp
}

// RunApp run bridge-test child process by multi parameters.
func (b *MockApp) RunApp(t *testing.T, name utils.MockAppName, args ...string) {
	if !(name == utils.EventWatcherApp ||
		name == utils.GasOracleApp ||
		name == utils.MessageRelayerApp ||
		name == utils.RollupRelayerApp) {
		t.Errorf(fmt.Sprintf("Don't support the mock app, name: %s", name))
		return
	}

	if app, ok := b.mockApps[name]; ok {
		t.Logf(fmt.Sprintf("%s already exist, free the current and recreate again", string(name)))
		app.WaitExit()
	}
	appAPI := cmd.NewCmd(string(name), append(b.args, args...)...)
	keyword := fmt.Sprintf("Start %s successfully", string(name)[:len(string(name))-len("-test")])
	appAPI.RunApp(func() bool { return appAPI.WaitResult(t, time.Second*20, keyword) })
	b.mockApps[name] = appAPI
}

// WaitExit wait util all processes exit.
func (b *MockApp) WaitExit() {
	for _, app := range b.mockApps {
		app.WaitExit()
	}
	b.mockApps = make(map[utils.MockAppName]docker.AppAPI)
}

// Free stop and release bridge mocked apps.
func (b *MockApp) Free() {
	b.WaitExit()
	_ = os.Remove(b.bridgeFile)
}

// MockConfig creates a new bridge config.
func (b *MockApp) MockConfig(store bool) error {
	base := b.base
	// Load origin bridge config file.
	cfg, err := config.NewConfig(b.originFile)
	if err != nil {
		return err
	}

	cfg.L1Config.Endpoint = base.L1gethImg.Endpoint()
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = base.L1gethImg.Endpoint()
	cfg.L2Config.Endpoint = base.L2gethImg.Endpoint()
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = base.L2gethImg.Endpoint()
	cfg.DBConfig.DSN = base.DBImg.Endpoint()
	b.Config = cfg

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
