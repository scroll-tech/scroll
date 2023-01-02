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
	t.Run("testStartProcess", testStartProcess)

	t.Cleanup(func() {
		free(t)
	})
}

func testStartProcess(t *testing.T) {
	// migrate db.
	runDBCliApp(t, "reset", "successful to reset")
	runDBCliApp(t, "migrate", "current version:")

	// Start bridge process.
	bridgeCmd := runBridgeApp(t)
	bridgeCmd.ExpectWithTimeout(true, time.Second*10, "Start bridge successfully")
	bridgeCmd.RunApp(true)

	// Start coordinator process.
	coordinatorCmd := runCoordinatorApp(t)
	coordinatorCmd.ExpectWithTimeout(true, time.Second*10, "Start coordinator successfully")
	coordinatorCmd.RunApp(true)

	// Start roller process.
	rollerCmd := runRollerApp(t)
	rollerCmd.ExpectWithTimeout(true, time.Second*20, "roller start successfully")
	rollerCmd.ExpectWithTimeout(true, time.Second*30, "register to coordinator successfully!")
	rollerCmd.RunApp(true)

	rollerCmd.WaitExit()
	bridgeCmd.WaitExit()
	coordinatorCmd.WaitExit()
}
