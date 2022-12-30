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
	"github.com/stretchr/testify/assert"

	_ "scroll-tech/database/cmd/app"

	_ "scroll-tech/roller/cmd/app"

	_ "scroll-tech/bridge/cmd/app"
	"scroll-tech/bridge/sender"

	"scroll-tech/common/cmd"
	"scroll-tech/common/docker"
	"scroll-tech/common/viper"

	_ "scroll-tech/coordinator/cmd/app"

	rollerConfig "scroll-tech/roller/config"
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
	RunApp(parallel bool)
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
	app.RunApp(false)
}

func runRollerApp(t *testing.T, args ...string) appAPI {
	args = append(args, "--log.debug", "--config", rollerFile)
	return cmd.NewCmd(t, "roller-test", args...)
}

func runSender(t *testing.T, endpoint string) *sender.Sender {
	priv, err := crypto.HexToECDSA("1212121212121212121212121212121212121212121212121212121212121212")
	assert.NoError(t, err)
	vp := viper.NewEmptyViper()
	vp.Set("endpoint", endpoint)
	vp.Set("check_pending_time", 3)
	vp.Set("escalate_blocks", 100)
	vp.Set("confirmations", 0)
	vp.Set("escalate_multiple_num", 11)
	vp.Set("escalate_multiple_den", 10)
	vp.Set("tx_type", "DynamicFeeTx")
	newSender, err := sender.NewSender(context.Background(), vp, []*ecdsa.PrivateKey{priv})
	assert.NoError(t, err)
	return newSender
}

func mockBridgeConfig(t *testing.T) string {
	// Load origin bridge config file.
	vp, err := viper.NewViper("../../bridge/config.json", true)
	assert.NoError(t, err)

	if l1gethImg != nil {
		vp.Set("l1_config.endpoint", l1gethImg.Endpoint())
		vp.Set("l2_config.relayer_config.sender_config.endpoint", l1gethImg.Endpoint())
	}
	if l2gethImg != nil {
		vp.Set("l2_config.endpoint", l2gethImg.Endpoint())
		vp.Set("l1_config.relayer_config.sender_config.endpoint", l2gethImg.Endpoint())
	}
	if dbImg != nil {
		vp.Set("db_config.dsn", dbImg.Endpoint())
	}

	// Store changed bridge config into a temp file.
	file := fmt.Sprintf("/tmp/%d_bridge-config.json", timestamp)
	assert.NoError(t, vp.WriteConfigAs(file))

	return file
}

func mockCoordinatorConfig(t *testing.T) string {
	vp, err := viper.NewViper("../../coordinator/config.json", true)
	assert.NoError(t, err)

	vp.Set("roller_manager_config.verifier.mock_mode", true)
	if dbImg != nil {
		vp.Set("db_config.dsn", dbImg.Endpoint())
	}

	if l2gethImg != nil {
		vp.Set("l2_config.endpoint", l2gethImg.Endpoint())
	}

	file := fmt.Sprintf("/tmp/%d_coordinator-config.json", timestamp)
	assert.NoError(t, vp.WriteConfigAs(file))

	return file
}

func mockDatabaseConfig(t *testing.T) string {
	vp, err := viper.NewViper("../../database/config.json", true)
	assert.NoError(t, err)
	if dbImg != nil {
		vp.Set("dsn", dbImg.Endpoint())
	}

	file := fmt.Sprintf("/tmp/%d_db-config.json", timestamp)
	assert.NoError(t, vp.WriteConfigAs(file))

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
