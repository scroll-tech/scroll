package app

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"

	coordinatorConfig "scroll-tech/coordinator/internal/config"

	"scroll-tech/common/cmd"
	"scroll-tech/common/docker"
	"scroll-tech/common/utils"
)

var (
	httpStartPort int64 = 40000
)

// CoordinatorApp coordinator-test client manager.
type CoordinatorApp struct {
	Config *coordinatorConfig.Config

	base *docker.App

	originFile      string
	coordinatorFile string
	HTTPPort        int64

	args []string
	docker.AppAPI
}

// NewCoordinatorApp return a new coordinatorApp manager.
func NewCoordinatorApp(base *docker.App, file string) *CoordinatorApp {
	coordinatorFile := fmt.Sprintf("/tmp/%d_coordinator-config.json", base.Timestamp)
	port, _ := rand.Int(rand.Reader, big.NewInt(2000))
	httpPort := port.Int64() + httpStartPort
	coordinatorApp := &CoordinatorApp{
		base:            base,
		originFile:      file,
		coordinatorFile: coordinatorFile,
		HTTPPort:        httpPort,
		args:            []string{"--log.debug", "--config", coordinatorFile, "--http", "--http.port", strconv.Itoa(int(httpPort))},
	}
	if err := coordinatorApp.MockConfig(true); err != nil {
		panic(err)
	}
	return coordinatorApp
}

// RunApp run coordinator-test child process by multi parameters.
func (c *CoordinatorApp) RunApp(t *testing.T, args ...string) {
	c.AppAPI = cmd.NewCmd(string(utils.CoordinatorApp), append(c.args, args...)...)
	c.AppAPI.RunApp(func() bool { return c.AppAPI.WaitResult(t, time.Second*20, "Start coordinator successfully") })
}

// Free stop and release coordinator-test.
func (c *CoordinatorApp) Free() {
	if !utils.IsNil(c.AppAPI) {
		c.AppAPI.WaitExit()
	}
	_ = os.Remove(c.coordinatorFile)
}

// HTTPEndpoint returns ws endpoint.
func (c *CoordinatorApp) HTTPEndpoint() string {
	return fmt.Sprintf("http://localhost:%d", c.HTTPPort)
}

// MockConfig creates a new coordinator config.
func (c *CoordinatorApp) MockConfig(store bool) error {
	base := c.base
	cfg, err := coordinatorConfig.NewConfig(c.originFile)
	if err != nil {
		return err
	}
	// Reset prover manager config for manager test cases.
	cfg.ProverManagerConfig = &coordinatorConfig.ProverManagerConfig{
		ProversPerSession: 1,
		Verifier:          &coordinatorConfig.VerifierConfig{MockMode: true},
		CollectionTimeSec: 60,
		TokenTimeToLive:   1,
	}
	cfg.DBConfig.DSN = base.DBImg.Endpoint()
	cfg.L2Config.ChainID = 111
	c.Config = cfg

	if !store {
		return nil
	}

	data, err := json.Marshal(c.Config)
	if err != nil {
		return err
	}

	return os.WriteFile(c.coordinatorFile, data, 0600)
}
