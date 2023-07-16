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
	wsStartPort int64 = 40000
)

// CoordinatorApp coordinator-test client manager.
type CoordinatorApp struct {
	Config *coordinatorConfig.Config

	base *docker.App

	originFile      string
	coordinatorFile string
	WSPort          int64

	args []string
	docker.AppAPI
}

// NewCoordinatorApp return a new coordinatorApp manager.
func NewCoordinatorApp(base *docker.App, file string) *CoordinatorApp {
	coordinatorFile := fmt.Sprintf("/tmp/%d_coordinator-config.json", base.Timestamp)
	port, _ := rand.Int(rand.Reader, big.NewInt(2000))
	wsPort := port.Int64() + wsStartPort
	coordinatorApp := &CoordinatorApp{
		base:            base,
		originFile:      file,
		coordinatorFile: coordinatorFile,
		WSPort:          wsPort,
		args:            []string{"--log.debug", "--config", coordinatorFile, "--ws", "--ws.port", strconv.Itoa(int(wsPort))},
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

// WSEndpoint returns ws endpoint.
func (c *CoordinatorApp) WSEndpoint() string {
	return fmt.Sprintf("ws://localhost:%d", c.WSPort)
}

// MockConfig creates a new coordinator config.
func (c *CoordinatorApp) MockConfig(store bool) error {
	base := c.base
	cfg, err := coordinatorConfig.NewConfig(c.originFile)
	if err != nil {
		return err
	}
	// Reset roller manager config for manager test cases.
	cfg.RollerManagerConfig = &coordinatorConfig.RollerManagerConfig{
		RollersPerSession: 1,
		Verifier:          &coordinatorConfig.VerifierConfig{MockMode: true},
		CollectionTime:    1,
		TokenTimeToLive:   1,
	}
	cfg.DBConfig.DSN = base.DBImg.Endpoint()
	cfg.L2Config.Endpoint = base.L2gethImg.Endpoint()
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
