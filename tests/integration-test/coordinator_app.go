package integration

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"

	coordinatorConfig "scroll-tech/coordinator/config"

	"scroll-tech/common/cmd"
)

type coordinatorApp struct {
	base *dockerApp

	cfg             *coordinatorConfig.Config
	coordinatorFile string

	name string
	args []string
	appAPI
}

func newCoordinatorApp(base *dockerApp) *coordinatorApp {
	file := fmt.Sprintf("/tmp/%d_coordinator-config.json", base.timestamp)
	return &coordinatorApp{
		base:            base,
		coordinatorFile: file,
		name:            "coordinator-test",
		args:            []string{"--log.debug", "--config", file},
	}
}

func (c *coordinatorApp) runApp(t *testing.T, args ...string) {
	if err := c.mockCoordinatorConfig(); err != nil {
		t.Fatal(err)
	}
	port, _ := rand.Int(rand.Reader, big.NewInt(2000))
	c.base.wsPort = port.Int64() + 22000
	args = append(args, []string{"--ws", "--ws.port", strconv.Itoa(int(c.base.wsPort))}...)
	args = append(args, c.args...)
	c.appAPI = cmd.NewCmd(t, c.name, args...)
	c.appAPI.RunApp(func() bool { return c.appAPI.WaitResult(time.Second*20, "Start coordinator successfully") })
}

func (c *coordinatorApp) free() {
	_ = os.Remove(c.coordinatorFile)
}

func (c *coordinatorApp) mockCoordinatorConfig() error {
	if c.cfg == nil {
		cfg, err := coordinatorConfig.NewConfig("../../coordinator/config.json")
		if err != nil {
			return err
		}
		cfg.RollerManagerConfig.Verifier.MockMode = true
		c.cfg = cfg
	}

	base := c.base
	if base.dbImg != nil {
		c.cfg.DBConfig.DSN = base.dbImg.Endpoint()
	}

	if base.l2gethImg != nil {
		c.cfg.L2Config.Endpoint = base.l2gethImg.Endpoint()
	}

	data, err := json.Marshal(c.cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(c.coordinatorFile, data, 0644)
}
