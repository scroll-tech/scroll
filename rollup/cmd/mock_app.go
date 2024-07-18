package app

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/testcontainers"
	"scroll-tech/common/utils"

	"scroll-tech/rollup/internal/config"
)

// MockApp mockApp-test client manager.
type MockApp struct {
	Config   *config.Config
	testApps *testcontainers.TestcontainerApps

	mockApps map[utils.MockAppName]*cmd.Cmd

	originFile string
	rollupFile string

	args []string
}

// NewRollupApp return a new rollupApp manager.
func NewRollupApp(testApps *testcontainers.TestcontainerApps, file string) *MockApp {
	rollupFile := fmt.Sprintf("/tmp/%d_rollup-config.json", testApps.Timestamp)
	rollupApp := &MockApp{
		testApps:   testApps,
		mockApps:   make(map[utils.MockAppName]*cmd.Cmd),
		originFile: file,
		rollupFile: rollupFile,
		args:       []string{"--log.debug", "--config", rollupFile},
	}
	if err := rollupApp.MockConfig(true); err != nil {
		panic(err)
	}
	return rollupApp
}

// RunApp run rollup-test child process by multi parameters.
func (b *MockApp) RunApp(t *testing.T, name utils.MockAppName, args ...string) {
	if !(name == utils.GasOracleApp || name == utils.RollupRelayerApp) {
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
	b.mockApps = make(map[utils.MockAppName]*cmd.Cmd)
}

// Free stop and release rollup mocked apps.
func (b *MockApp) Free() {
	b.WaitExit()
	_ = os.Remove(b.rollupFile)
}

// MockConfig creates a new rollup config.
func (b *MockApp) MockConfig(store bool) error {
	// Load origin rollup config file.
	cfg, err := config.NewConfig(b.originFile)
	if err != nil {
		return err
	}

	l1GethEndpoint, err := b.testApps.GetPoSL1EndPoint()
	if err != nil {
		return err
	}
	l2GethEndpoint, err := b.testApps.GetL2GethEndPoint()
	if err != nil {
		return err
	}
	dbEndpoint, err := b.testApps.GetDBEndPoint()
	if err != nil {
		return err
	}
	cfg.L1Config.Endpoint = l1GethEndpoint
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = l2GethEndpoint
	cfg.L2Config.Endpoint = l2GethEndpoint
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1GethEndpoint
	cfg.DBConfig.DSN = dbEndpoint
	b.Config = cfg

	if !store {
		return nil
	}
	// Store changed rollup config into a temp file.
	data, err := json.Marshal(b.Config)
	if err != nil {
		return err
	}
	return os.WriteFile(b.rollupFile, data, 0600)
}
