package integration

import (
	"context"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/config"
	"scroll-tech/database"
)

func TestIntegration(t *testing.T) {
	setupEnv(t)

	// test db_cli
	t.Run("testDatabaseOperation", testDatabaseOperation)
	// test bridge service
	t.Run("testBridgeOperation", testBridgeOperation)

	t.Cleanup(func() {
		free(t)
	})
}

func testDatabaseOperation(t *testing.T) {
	resetCmd := runDBCliApp(t, "reset")
	// Wait reset result
	resetCmd.ExpectWithTimeout(false, time.Second*3, "successful to reset")
	resetCmd.WaitExit()

	migrateCmd := runDBCliApp(t, "migrate")
	// Wait migrate result
	migrateCmd.ExpectWithTimeout(false, time.Second*3, "current version:")
	migrateCmd.WaitExit()
}

func testBridgeOperation(t *testing.T) {
	cfg, err := config.NewConfig("../../bridge/config.json")
	assert.NoError(t, err)

	// reset db.
	dbCmd := runDBCliApp(t, "migrate")
	dbCmd.WaitExit()

	// Start bridge process.
	bridgeCmd := runBridgeApp(t)
	defer bridgeCmd.WaitExit()

	// Start coordinator process.
	coordinatorCmd := runCoordinatorApp(t, "--ws.port", "8391")
	defer coordinatorCmd.WaitExit()

	// Start roller process.
	rollerCmd := runRollerApp(t)
	defer rollerCmd.WaitExit()

	// Create sender.
	relayerCfg := cfg.L1Config.RelayerConfig
	relayerCfg.SenderConfig.Endpoint = l2gethImg.Endpoint()
	relayerCfg.SenderConfig.Confirmations = 0
	// Send txs in parallel.
	newSender := runSender(t, relayerCfg.SenderConfig, relayerCfg.MessageSenderPrivateKeys, common.HexToAddress("0xFe94e28e4092A4Edc906D59b59623544B81b7F80"), nil)

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
			batch, err := db.GetLatestFinalizedBatch()
			if err == nil && batch.StartBlockNumber < latest && batch.EndBlockNumber >= latest {
				t.Logf("get latest finalized batch, ID: %s", batch.ID)
				return
			}
			/*number, err = db.GetBlockTracesLatestHeight()
			if err == nil {
				t.Logf("current latest trace number is %d", number)
			}*/
		case <-tickStop:
			t.Errorf("has not receive the latest trace after %d seconds", 10)
			return
		}
	}

	// Release handler.
	newSender.Stop()
}
