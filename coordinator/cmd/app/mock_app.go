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

	"scroll-tech/common/cmd"
	"scroll-tech/common/docker"
	"scroll-tech/common/utils"
	coordinatorConfig "scroll-tech/coordinator/config"
)

var (
	wsStartPort int64 = 40000
)

type CoordinatorApp struct {
	Config *coordinatorConfig.Config

	base *docker.DockerApp

	originFile      string
	coordinatorFile string
	WSPort          int64

	args []string
	docker.AppAPI
}

func NewCoordinatorApp(base *docker.DockerApp, file string) *CoordinatorApp {
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
	return coordinatorApp
}

func (c *CoordinatorApp) RunApp(t *testing.T, args ...string) {
	c.AppAPI = cmd.NewCmd("coordinator-test", append(c.args, args...)...)
	c.AppAPI.RunApp(func() bool { return c.AppAPI.WaitResult(t, time.Second*20, "Start coordinator successfully") })
}

func (c *CoordinatorApp) Free() {
	if !utils.IsNil(c.AppAPI) {
		c.AppAPI.WaitExit()
		_ = os.Remove(c.coordinatorFile)
	}
}

func (c *CoordinatorApp) WSEndpoint() string {
	return fmt.Sprintf("ws://localhost:%d", c.WSPort)
}

func (c *CoordinatorApp) MockCoordinatorConfig(store bool) error {
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
	c.Config = cfg

	c.Config.DBConfig.DSN = base.DBImg.Endpoint()
	c.Config.L2Config.Endpoint = base.L2gethImg.Endpoint()

	if !store {
		return nil
	}

	data, err := json.Marshal(c.Config)
	if err != nil {
		return err
	}

	return os.WriteFile(c.coordinatorFile, data, 0644)
}
