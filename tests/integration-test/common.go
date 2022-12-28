package integration

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"

	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	_ "scroll-tech/database/cmd/app"

	_ "scroll-tech/roller/cmd/app"
	rollerConfig "scroll-tech/roller/config"

	_ "scroll-tech/bridge/cmd/app"
	bridgeConfig "scroll-tech/bridge/config"

	"scroll-tech/common/cmd"
	"scroll-tech/common/docker"

	_ "scroll-tech/coordinator/cmd/app"
	coordinatorConfig "scroll-tech/coordinator/config"
)

var (
	l1gethImg docker.ImgInstance
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance

	l2Client *ethclient.Client

	timestamp int
	wsPort    int64

	bridgeFile      string
	dbFile          string
	coordinatorFile string
	rollerFile      string
	bboltDB         string

	privkey *ecdsa.PrivateKey
	l2Root  *bind.TransactOpts
)

type appAPI interface {
	RunApp(parallel bool)
	WaitExit()
	ExpectWithTimeout(parallel bool, timeout time.Duration, keyword string)
	RegistFunc(key string, check func(buf string))
	UnRegistFunc(key string)
}

func setupEnv(t *testing.T) {
	// Start l1geth l2geth and postgres.
	l1gethImg = docker.NewTestL1Docker(t)
	l2gethImg = docker.NewTestL2Docker(t)
	dbImg = docker.NewTestDBDocker(t, "postgres")

	var err error
	l2Client, err = ethclient.Dial(l2gethImg.Endpoint())
	assert.NoError(t, err)

	privkey, _ = crypto.HexToECDSA("1212121212121212121212121212121212121212121212121212121212121212")
	l2Root, err = bind.NewKeyedTransactorWithChainID(privkey, big.NewInt(53077))
	assert.NoError(t, err)

	// Create configs.
	mockConfig(t)
}

func free(t *testing.T) {
	assert.NoError(t, l1gethImg.Stop())
	assert.NoError(t, l2gethImg.Stop())
	assert.NoError(t, dbImg.Stop())
	// Delete configs.
	freeConfig()
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
	okCh := make(chan struct{}, 1)
	app.RegistFunc(keyword, func(buf string) {
		if strings.Contains(buf, keyword) {
			select {
			case okCh <- struct{}{}:
			default:
				return
			}
		}
	})
	defer app.UnRegistFunc(keyword)
	app.RunApp(true)

	select {
	case <-okCh:
		return
	case <-time.After(time.Second * 3):
		assert.Fail(t, fmt.Sprintf("didn't get the desired result before timeout, keyword: %s", keyword))
	}
}

func runRollerApp(t *testing.T, args ...string) appAPI {
	args = append(args, "--log.debug", "--config", rollerFile)
	return cmd.NewCmd(t, "roller-test", args...)
}

func mockConfig(t *testing.T) {
	freeConfig()
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

func freeConfig() {
	// Delete temporary files.
	_ = os.Remove(bridgeFile)
	_ = os.Remove(dbFile)
	_ = os.Remove(coordinatorFile)
	_ = os.Remove(rollerFile)
	_ = os.Remove(bboltDB)
}

func mockBridgeConfig(t *testing.T) string {
	// Load origin bridge config file.
	cfg, err := bridgeConfig.NewConfig("../../bridge/config.json")
	assert.NoError(t, err)
	cfg.L2Config.BatchProposerConfig.BatchTimeSec = 0
	cfg.L2Config.Confirmations = 0

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
	data, err := json.MarshalIndent(cfg, "", "    ")
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

	data, err := json.MarshalIndent(cfg, "", "    ")
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
	data, err := json.MarshalIndent(cfg, "", "    ")
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

	data, err := json.MarshalIndent(cfg, "", "    ")
	assert.NoError(t, err)

	file := fmt.Sprintf("/tmp/%d_roller-config.json", timestamp)
	err = os.WriteFile(file, data, 0644)
	assert.NoError(t, err)

	return file
}
