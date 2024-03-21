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

	"github.com/scroll-tech/go-ethereum/params"

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
	Config      *coordinatorConfig.Config
	ChainConfig *params.ChainConfig

	base *docker.App

	configOriginFile      string
	chainConfigOriginFile string
	coordinatorFile       string
	genesisFile           string
	HTTPPort              int64

	args []string
	docker.AppAPI
}

// NewCoordinatorApp return a new coordinatorApp manager.
func NewCoordinatorApp(base *docker.App, configFile string, chainConfigFile string) *CoordinatorApp {
	coordinatorFile := fmt.Sprintf("/tmp/%d_coordinator-config.json", base.Timestamp)
	genesisFile := fmt.Sprintf("/tmp/%d_genesis.json", base.Timestamp)
	port, _ := rand.Int(rand.Reader, big.NewInt(2000))
	httpPort := port.Int64() + httpStartPort
	coordinatorApp := &CoordinatorApp{
		base:                  base,
		configOriginFile:      configFile,
		chainConfigOriginFile: chainConfigFile,
		coordinatorFile:       coordinatorFile,
		genesisFile:           genesisFile,
		HTTPPort:              httpPort,
		args:                  []string{"--log.debug", "--config", coordinatorFile, "--genesis", genesisFile, "--http", "--http.port", strconv.Itoa(int(httpPort))},
	}
	if err := coordinatorApp.MockConfig(true); err != nil {
		panic(err)
	}
	return coordinatorApp
}

// RunApp run coordinator-test child process by multi parameters.
func (c *CoordinatorApp) RunApp(t *testing.T, args ...string) {
	c.AppAPI = cmd.NewCmd(string(utils.CoordinatorAPIApp), append(c.args, args...)...)
	c.AppAPI.RunApp(func() bool { return c.AppAPI.WaitResult(t, time.Second*20, "Start coordinator api successfully") })
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
	cfg, err := coordinatorConfig.NewConfig(c.configOriginFile)
	if err != nil {
		return err
	}
	// Reset prover manager config for manager test cases.
	cfg.ProverManager = &coordinatorConfig.ProverManager{
		ProversPerSession:      1,
		Verifier:               &coordinatorConfig.VerifierConfig{MockMode: true},
		BatchCollectionTimeSec: 60,
		ChunkCollectionTimeSec: 60,
		SessionAttempts:        10,
		MaxVerifierWorkers:     4,
		MinProverVersion:       "v1.0.0",
	}
	cfg.DB.DSN = base.DBImg.Endpoint()
	cfg.L2.ChainID = 111
	cfg.Auth.ChallengeExpireDurationSec = 1
	cfg.Auth.LoginExpireDurationSec = 1
	c.Config = cfg

	genesis, err := utils.ReadGenesis(c.chainConfigOriginFile)
	if err != nil {
		return err
	}
	chainConf := genesis.Config
	c.ChainConfig = chainConf

	if !store {
		return nil
	}

	coordinatorConfigData, err := json.Marshal(c.Config)
	if err != nil {
		return err
	}
	genesisConfigData, err := json.Marshal(genesis)
	if err != nil {
		return err
	}

	if writeErr := os.WriteFile(c.coordinatorFile, coordinatorConfigData, 0600); writeErr != nil {
		return writeErr
	}
	if writeErr := os.WriteFile(c.genesisFile, genesisConfigData, 0600); writeErr != nil {
		return writeErr
	}
	return nil
}
