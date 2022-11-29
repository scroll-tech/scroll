package integration_test

import (
	"context"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
	"scroll-tech/database"
)

func testDatabaseOperation(t *testing.T) {
	resetCmd := runDBCliApp(t, "reset")
	resetCmd.OpenLog(true)
	// Wait reset result
	resetCmd.ExpectWithTimeout(true, time.Second*3, "successful to reset")
	resetCmd.WaitExit()

	migrateCmd := runDBCliApp(t, "migrate")
	// Wait migrate result
	migrateCmd.ExpectWithTimeout(true, time.Second*3, "current version: 5")
	migrateCmd.WaitExit()
}

func testBridgeOperation(t *testing.T) {
	cfg, err := config.NewConfig("../../bridge/config.json")
	assert.NoError(t, err)

	// reset db.
	dbCmd := runDBCliApp(t, "reset")
	dbCmd.WaitExit()

	// Start bridge process.
	bridgeCmd := runBridgeApp(t)
	defer bridgeCmd.WaitExit()

	// Create sender.
	relayerCfg := cfg.L1Config.RelayerConfig
	relayerCfg.SenderConfig.Endpoint = l2gethImg.Endpoint()
	relayerCfg.SenderConfig.Confirmations = 0
	newSender, err := sender.NewSender(context.Background(), relayerCfg.SenderConfig, relayerCfg.MessageSenderPrivateKeys)
	assert.NoError(t, err)
	// Send txs in parallel.
	var (
		eg errgroup.Group
		to = common.HexToAddress("0xFe94e28e4092A4Edc906D59b59623544B81b7F80")
	)
	// Send txs.
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

	// Create l2client and get the latest block number.
	l2Cli, err := ethclient.Dial(l2gethImg.Endpoint())
	assert.NoError(t, err)
	latest, err := l2Cli.BlockNumber(context.Background())
	assert.NoError(t, err)

	// get db handler.
	db, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        dbImg.Endpoint(),
	})
	assert.NoError(t, err)

	var (
		tick     = time.NewTicker(time.Second)
		tickStop = time.After(time.Second * 30)
		number   int64
	)
	for latest > uint64(number) {
		select {
		case <-tick.C:
			number, err = db.GetBlockTracesLatestHeight()
			if err == nil {
				t.Logf("current latest trace number is %d", number)
			}
		case <-tickStop:
			t.Errorf("has not receive the latest trace after %d seconds", 10)
			return
		}
	}

	// Release handler.
	newSender.Stop()
}
