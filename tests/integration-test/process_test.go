package integration_test

import (
	"context"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"golang.org/x/sync/errgroup"
	"math/big"
	"scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"

	"scroll-tech/common/docker"
)

func testDatabaseOperation(t *testing.T) {
	cmd := docker.NewCmd(t)

	cmd.OpenLog(true)
	// Wait reset result
	cmd.ExpectWithTimeout(true, time.Second*3, "successful to reset")
	cmd.Run("db_cli-test", "--log.debug", "reset", "--config", "../../database/config.json", "--db.dsn", dbImg.Endpoint())
	cmd.WaitExit()

	// Wait migrate result
	cmd.ExpectWithTimeout(true, time.Second*3, "current version: 5")
	cmd.Run("db_cli-test", "--log.debug", "migrate", "--config", "../../database/config.json", "--db.dsn", dbImg.Endpoint())
	cmd.WaitExit()
}

func testBridgeOperation(t *testing.T) {
	cfg, err := config.NewConfig("../../bridge/config.json")
	assert.NoError(t, err)
	cfg.L2Config.Endpoint = l2gethImg.Endpoint()
	cfg.L1Config.Endpoint = l1gethImg.Endpoint()
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = l2gethImg.Endpoint()
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1gethImg.Endpoint()

	db, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        dbImg.Endpoint(),
	})
	assert.NoError(t, err)
	// Reset db.
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))

	cmd := docker.NewCmd(t)
	cmd.OpenLog(true)
	defer cmd.WaitExit()
	// Start bridge service.
	go cmd.Run("bridge-test", "--log.debug",
		"--config", "../../bridge/config.json",
		"--l1.endpoint", l1gethImg.Endpoint(),
		"--l2.endpoint", l2gethImg.Endpoint(),
		"--db.dsn", dbImg.Endpoint())

	relayerCfg := cfg.L1Config.RelayerConfig
	relayerCfg.SenderConfig.Confirmations = 0
	newSender, err := sender.NewSender(context.Background(), relayerCfg.SenderConfig, relayerCfg.MessageSenderPrivateKeys)
	assert.NoError(t, err)

	// Send txs in parallel.
	var (
		eg errgroup.Group
		to = common.HexToAddress("0xFe94e28e4092A4Edc906D59b59623544B81b7F80")
	)
	for i := 0; i < newSender.NumberOfAccounts(); i++ {
		idx := i
		eg.Go(func() error {
			_, err = newSender.SendTransaction(strconv.Itoa(idx), &to, big.NewInt(1), nil)
			if err == nil {
				<-newSender.ConfirmChan()
			}
			return err
		})
	}
	assert.NoError(t, eg.Wait())

	l2Cli, err := ethclient.Dial(l2gethImg.Endpoint())
	assert.NoError(t, err)
	latest, err := l2Cli.BlockNumber(context.Background())
	assert.NoError(t, err)

	var (
		tick     = time.NewTicker(time.Second)
		tickStop = time.After(time.Second * 30)
		number   int64
	)
	for latest > uint64(number) {
		select {
		case <-tick.C:
			number, _ = db.GetBlockTracesLatestHeight()
		case <-tickStop:
			t.Errorf("has not receive the latest trace after %d seconds", 10)
			return
		}
	}
}
