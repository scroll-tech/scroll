package integration

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	_ "scroll-tech/database/cmd/app"

	bridgeConfig "scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
	_ "scroll-tech/roller/cmd/app"
	rollerConfig "scroll-tech/roller/config"

	"scroll-tech/common/cmd"
	"scroll-tech/common/docker"

	_ "scroll-tech/coordinator/cmd/app"
	coordinatorConfig "scroll-tech/coordinator/config"
)

var (
	l1gethImg docker.ImgInstance
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance

	timestamp int
	wsPort    int64

	bridgeFile      string
	dbFile          string
	coordinatorFile string

	bboltDB    string
	rollerFile string
)

func setupEnv(t *testing.T) {
	// Start l1geth l2geth and postgres.
	l1gethImg = docker.NewTestL1Docker(t)
	l2gethImg = docker.NewTestL2Docker(t)
	dbImg = docker.NewTestDBDocker(t, "postgres")

	// Create a random ws port.
	port, _ := rand.Int(rand.Reader, big.NewInt(2000))
	wsPort = port.Int64() + 22000
	timestamp = time.Now().Nanosecond()

	// Load reset and store config into a random file.
	bridgeFile = mockBridgeConfig(t)
	dbFile = mockDatabaseConfig(t)
	coordinatorFile = mockCoordinatorConfig(t)
	rollerFile = mockRollerConfig(t)
}

func free(t *testing.T) {
	assert.NoError(t, l1gethImg.Stop())
	assert.NoError(t, l2gethImg.Stop())
	assert.NoError(t, dbImg.Stop())

	// Delete temporary files.
	assert.NoError(t, os.Remove(bridgeFile))
	assert.NoError(t, os.Remove(dbFile))
	assert.NoError(t, os.Remove(coordinatorFile))
	assert.NoError(t, os.Remove(rollerFile))
	assert.NoError(t, os.Remove(bboltDB))
}

type appAPI interface {
	WaitResult(timeout time.Duration, keyword string) bool
	RunApp(waitResult func() bool)
	WaitExit()
	ExpectWithTimeout(parallel bool, timeout time.Duration, keyword string)
}

func runBridgeApp(t *testing.T, args ...string) appAPI {
	args = append(args, "--log.debug", "--config", bridgeFile)
	return cmd.NewCmd(t, "bridge-test", args...)
}

func runCoordinatorApp(t *testing.T, args ...string) appAPI {
	args = append(args, "--log.debug", "--config", coordinatorFile, "--ws", "--ws.port", strconv.Itoa(int(wsPort)))
	// start process
	return cmd.NewCmd(t, "coordinator-test", args...)
}

func runDBCliApp(t *testing.T, option, keyword string) {
	args := []string{option, "--config", dbFile}
	app := cmd.NewCmd(t, "db_cli-test", args...)
	defer app.WaitExit()

	// Wait expect result.
	app.ExpectWithTimeout(true, time.Second*3, keyword)
	app.RunApp(nil)
}

func runRollerApp(t *testing.T, args ...string) appAPI {
	args = append(args, "--log.debug", "--config", rollerFile)
	return cmd.NewCmd(t, "roller-test", args...)
}

func runSender(t *testing.T, endpoint string) *sender.Sender {
	priv, err := crypto.HexToECDSA("1212121212121212121212121212121212121212121212121212121212121212")
	assert.NoError(t, err)
	newSender, err := sender.NewSender(context.Background(), &bridgeConfig.SenderConfig{
		Endpoint:            endpoint,
		CheckPendingTime:    3,
		EscalateBlocks:      100,
		Confirmations:       rpc.LatestBlockNumber,
		EscalateMultipleNum: 11,
		EscalateMultipleDen: 10,
		TxType:              "LegacyTx",
	}, []*ecdsa.PrivateKey{priv})
	assert.NoError(t, err)
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
	file := fmt.Sprintf("/tmp/%d_bridge-config.json", timestamp)
	err = os.WriteFile(file, data, 0644)
	assert.NoError(t, err)

	return file
}

func mockCoordinatorConfig(t *testing.T) string {
	cfg, err := coordinatorConfig.NewConfig("../../coordinator/config.json")
	assert.NoError(t, err)

	cfg.RollerManagerConfig.Verifier.MockMode = true
	if dbImg != nil {
		cfg.DBConfig.DSN = dbImg.Endpoint()
	}

	if l2gethImg != nil {
		cfg.L2Config.Endpoint = l2gethImg.Endpoint()
	}

	data, err := json.Marshal(cfg)
	assert.NoError(t, err)

	file := fmt.Sprintf("/tmp/%d_coordinator-config.json", timestamp)
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

	file := fmt.Sprintf("/tmp/%d_db-config.json", timestamp)
	err = os.WriteFile(file, data, 0644)
	assert.NoError(t, err)

	return file
}

func mockRollerConfig(t *testing.T) string {
	cfg, err := rollerConfig.NewConfig("../../roller/config.json")
	assert.NoError(t, err)
	cfg.CoordinatorURL = fmt.Sprintf("ws://localhost:%d", wsPort)

	// Reuse l1geth's keystore file
	cfg.KeystorePath = "../../common/docker/l1geth/genesis-keystore"
	cfg.KeystorePassword = "scrolltest"

	bboltDB = fmt.Sprintf("/tmp/%d_bbolt_db", timestamp)
	cfg.DBPath = bboltDB

	data, err := json.Marshal(cfg)
	assert.NoError(t, err)

	file := fmt.Sprintf("/tmp/%d_roller-config.json", timestamp)
	err = os.WriteFile(file, data, 0644)
	assert.NoError(t, err)

	return file
}
