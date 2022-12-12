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
	bridgeCmd.ExpectWithTimeout(false, time.Second*10, "Start bridge successfully")

	// Start coordinator process.
	coordinatorCmd := runCoordinatorApp(t, "--ws", "--ws.port", "8391")
	coordinatorCmd.RunApp(true)
	coordinatorCmd.ExpectWithTimeout(false, time.Second*10, "Start coordinator successfully")

	// Start roller process.
	rollerCmd := runRollerApp(t)
	rollerCmd.RunApp(true)
	rollerCmd.ExpectWithTimeout(false, time.Second*20, "roller start successfully")
	rollerCmd.ExpectWithTimeout(false, time.Second*10, "register to coordinator successfully!")

	rollerCmd.WaitExit()
	bridgeCmd.WaitExit()
	coordinatorCmd.WaitExit()
}
