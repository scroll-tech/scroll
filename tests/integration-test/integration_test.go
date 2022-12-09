package integration

import (
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
	// migrate db.
	runDBCliApp(t, "reset", "successful to reset")
	runDBCliApp(t, "migrate", "current version:")

	// Start bridge process.
	bridgeCmd := runBridgeApp(t)
	bridgeCmd.RunApp(true)
	bridgeCmd.ExpectWithTimeout(false, time.Second*3, "Start bridge successfully")

	// Start coordinator process.
	coordinatorCmd := runCoordinatorApp(t, "--ws", "--ws.port", "8391")
	coordinatorCmd.RunApp(true)
	coordinatorCmd.ExpectWithTimeout(false, time.Second*10, "Start coordinator successfully")

	// Start roller process.
	rollerCmd := runRollerApp(t)
	rollerCmd.RunApp(true)
	rollerCmd.ExpectWithTimeout(false, time.Second*10, "roller start successfully")
	rollerCmd.ExpectWithTimeout(false, time.Second*3, "register to coordinator successfully!")

	// Send txs in parallel.
	//newSender := runSender(t, l2gethImg.Endpoint(), common.HexToAddress("0xFe94e28e4092A4Edc906D59b59623544B81b7F80"), nil)
	//defer newSender.Stop()

	rollerCmd.WaitExit()
	bridgeCmd.WaitExit()
	coordinatorCmd.WaitExit()
}
