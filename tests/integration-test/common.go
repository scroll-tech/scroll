package integration

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/errgroup"
	"math/big"
	"os"
	coordinatorConfig "scroll-tech/coordinator/config"
	"strconv"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	_ "scroll-tech/bridge/cmd/app"
	bridgeConfig "scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
	"scroll-tech/common/docker"
	_ "scroll-tech/coordinator/cmd/app"
	"scroll-tech/database"
	_ "scroll-tech/database/cmd/app"
	_ "scroll-tech/roller/cmd/app"
)

var (
	l1gethImg docker.ImgInstance
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance

	bridgeFile      string
	dbFile          string
	coordinatorFile string
	rollerFile      string
)

func setupEnv(t *testing.T) {
	l1gethImg = docker.NewTestL1Docker(t)
	l2gethImg = docker.NewTestL2Docker(t)
	dbImg = docker.NewTestDBDocker(t, "postgres")

	bridgeFile = mockBridgeConfig(t)
	dbFile = mockDatabaseConfig(t)
	coordinatorFile = mockCoordinatorConfig(t)
}

func free(t *testing.T) {
	assert.NoError(t, l1gethImg.Stop())
	assert.NoError(t, l2gethImg.Stop())
	assert.NoError(t, dbImg.Stop())

	assert.NoError(t, os.Remove(bridgeFile))
	assert.NoError(t, os.Remove(dbFile))
	assert.NoError(t, os.Remove(coordinatorFile))
}

type appAPI interface {
	RunApp(parallel bool)
	WaitExit()
	ExpectWithTimeout(parallel bool, timeout time.Duration, keyword string)
}

func runBridgeApp(t *testing.T, args ...string) appAPI {
	args = append(args, "--log.debug", "--config", bridgeFile)
	return docker.NewCmd(t, "bridge-test", args...)
}

func runCoordinatorApp(t *testing.T, args ...string) appAPI {
	args = append(args, "--log.debug", "--config", coordinatorFile)
	// start process
	return docker.NewCmd(t, "coordinator-test", args...)
}

func runDBCliApp(t *testing.T, option, keyword string) {
	args := []string{option, "--log.debug", "--config", dbFile}
	cmd := docker.NewCmd(t, "db_cli-test", args...)
	defer cmd.WaitExit()

	// Wait expect result.
	cmd.ExpectWithTimeout(true, time.Second*3, keyword)
	cmd.RunApp(false)
}

func runRollerApp(t *testing.T, args ...string) appAPI {
	args = append(args, "--log.debug", "--config", "../../roller/config.toml")
	return docker.NewCmd(t, "roller-test", args...)
}

func runSender(t *testing.T, cfg *bridgeConfig.SenderConfig, privs []*ecdsa.PrivateKey, to common.Address, data []byte) *sender.Sender {
	newSender, err := sender.NewSender(context.Background(), cfg, privs)
	assert.NoError(t, err)
	eg := errgroup.Group{}
	for i := 0; i < newSender.NumberOfAccounts(); i++ {
		idx := i
		eg.Go(func() error {
			_, err = newSender.SendTransaction(strconv.Itoa(idx), &to, big.NewInt(1), data)
			if err == nil {
				<-newSender.ConfirmChan()
			}
			return err
		})
	}
	assert.NoError(t, eg.Wait())
	return newSender
}

func mockBridgeConfig(t *testing.T) string {
	// Load origin bridge config file.
	cfg, err := bridgeConfig.NewConfig("../../bridge/config.json")
	assert.NoError(t, err)

	if l1gethImg != nil {
		cfg.L1Config.Endpoint = l1gethImg.Endpoint()
		cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1gethImg.Endpoint()
	}
	if l2gethImg != nil {
		cfg.L2Config.Endpoint = l2gethImg.Endpoint()
		cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = l2gethImg.Endpoint()
	}
	if dbImg != nil {
		cfg.DBConfig.DSN = dbImg.Endpoint()
	}

	// Store changed bridge config into a temp file.
	data, err := json.Marshal(cfg)
	assert.NoError(t, err)
	file := fmt.Sprintf("/tmp/%d_bridge-config.json", time.Now().Nanosecond())
	err = os.WriteFile(file, data, 0644)
	assert.NoError(t, err)

	return file
}

func mockCoordinatorConfig(t *testing.T) string {
	cfg, err := coordinatorConfig.NewConfig("../../coordinator/config.json")
	assert.NoError(t, err)

	cfg.RollerManagerConfig.VerifierEndpoint = ""
	if dbImg != nil {
		cfg.DBConfig.DSN = dbImg.Endpoint()
	}
	data, err := json.Marshal(cfg)
	assert.NoError(t, err)

	file := fmt.Sprintf("/tmp/%d_coordinator-config.json", time.Now().Nanosecond())
	err = os.WriteFile(file, data, 0644)
	assert.NoError(t, err)

	return file
}

func mockDatabaseConfig(t *testing.T) string {
	cfg, err := database.NewConfig("../../database/config.json")
	assert.NoError(t, err)
	if dbImg != nil {
		cfg.DSN = dbImg.Endpoint()
	}
	data, err := json.Marshal(cfg)
	assert.NoError(t, err)

	file := fmt.Sprintf("/tmp/%d_db-config.json", time.Now().Nanosecond())
	err = os.WriteFile(file, data, 0644)
	assert.NoError(t, err)

	return file
}
