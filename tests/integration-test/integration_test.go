package integration

import (
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
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
	// migrate db.
	runDBCliApp(t, "reset", "successful to reset")
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

	// Send txs in parallel.
	newSender := runSender(t, l2gethImg.Endpoint(), common.HexToAddress("0xFe94e28e4092A4Edc906D59b59623544B81b7F80"), nil)

	// Expect verify result.
	coordinatorCmd.ExpectWithTimeout(false, 60*time.Second, "Verify zk proof successfully")

	newSender.Stop()
	rollerCmd.WaitExit()
	bridgeCmd.WaitExit()
	coordinatorCmd.WaitExit()
}
