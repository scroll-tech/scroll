package app

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/modern-go/reflect2"

	"scroll-tech/common/cmd"
	"scroll-tech/common/docker"
	coordinatorConfig "scroll-tech/coordinator/config"
)

type CoordinatorApp struct {
	base *docker.DockerApp

	Config          *coordinatorConfig.Config
	originFile      string
	coordinatorFile string
	wsPort          int64

	args []string
	docker.AppAPI
}

func NewCoordinatorApp(base *docker.DockerApp, file string) *CoordinatorApp {
	coordinatorFile := fmt.Sprintf("/tmp/%d_coordinator-config.json", base.Timestamp)
	port, _ := rand.Int(rand.Reader, big.NewInt(2000))
	coordinatorApp := &CoordinatorApp{
		base:            base,
		originFile:      file,
		coordinatorFile: coordinatorFile,
		wsPort:          port.Int64(),
		args:            []string{"--log.debug", "--config", coordinatorFile, "--ws", "--ws.port", port.String()},
	}
	if err := coordinatorApp.MockCoordinatorConfig(); err != nil {
		panic(err)
	}
	return coordinatorApp
}

func (c *CoordinatorApp) RunApp(t *testing.T, args ...string) {
	c.AppAPI = cmd.NewCmd("coordinator-test", append(c.args, args...)...)
	c.AppAPI.RunApp(func() bool { return c.AppAPI.WaitResult(t, time.Second*20, "Start coordinator successfully") })
}

func (c *CoordinatorApp) Free() {
	if !reflect2.IsNil(c.AppAPI) {
		c.AppAPI.WaitExit()
		_ = os.Remove(c.coordinatorFile)
	}
}

func (c *CoordinatorApp) WSEndpoint() string {
	return fmt.Sprintf("ws://localhost:%d", c.wsPort)
}

func (c *CoordinatorApp) MockCoordinatorConfig() error {
	base := c.base
	if c.Config == nil {
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
		c.Config = cfg
	}

	c.Config.DBConfig.DSN = base.DBImg.Endpoint()
	c.Config.L2Config.Endpoint = base.L2gethImg.Endpoint()

	data, err := json.Marshal(c.Config)
	if err != nil {
		return err
	}

	return os.WriteFile(c.coordinatorFile, data, 0644)
}
