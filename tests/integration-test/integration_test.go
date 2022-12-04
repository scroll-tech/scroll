package integration

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"scroll-tech/bridge/config"
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	setupEnv(t)

	// test db_cli migrate cmd.
	t.Run("testDBClientMigrate", func(t *testing.T) {
		runDBCliApp(t, "migrate", "current version:")
	})

	// test bridge service
	t.Run("testBridgeOperation", testBridgeOperation)

	t.Cleanup(func() {
		free(t)
	})
}

func testBridgeOperation(t *testing.T) {
	cfg, err := config.NewConfig("../../bridge/config.json")
	assert.NoError(t, err)

	// migrate db.
	runDBCliApp(t, "migrate", "current version:")

	// Start bridge process.
	bridgeCmd := runBridgeApp(t)
	bridgeCmd.RunApp(true)

	// Start coordinator process.
	coordinatorCmd := runCoordinatorApp(t, "--ws", "--ws.port", "8391")
	coordinatorCmd.RunApp(true)

	// Start roller process.
	rollerCmd := runRollerApp(t)
	rollerCmd.RunApp(true)

	// Create sender.
	relayerCfg := cfg.L1Config.RelayerConfig
	relayerCfg.SenderConfig.Endpoint = l2gethImg.Endpoint()
	relayerCfg.SenderConfig.Confirmations = 0
	// Send txs in parallel.
	newSender := runSender(t, relayerCfg.SenderConfig, relayerCfg.MessageSenderPrivateKeys, common.HexToAddress("0xFe94e28e4092A4Edc906D59b59623544B81b7F80"), nil)

	coordinatorCmd.ExpectWithTimeout(false, 60*time.Second, "Verify zk proof successfully")

	newSender.Stop()
	rollerCmd.WaitExit()
	bridgeCmd.WaitExit()
	coordinatorCmd.WaitExit()
}
