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
	bridgeCmd.RunApp(func() bool { return bridgeCmd.WaitResult(time.Second*20, "Start bridge successfully") })

	// Start coordinator process.
	coordinatorCmd := runCoordinatorApp(t, "--ws", "--ws.port", "8391")
	coordinatorCmd.ExpectWithTimeout(true, time.Second*20, "Start coordinator successfully")
	coordinatorCmd.RunApp(true)

	// Start roller process.
	rollerCmd := runRollerApp(t)
	rollerCmd.ExpectWithTimeout(true, time.Second*40, "roller start successfully")
	rollerCmd.ExpectWithTimeout(true, time.Second*60, "register to coordinator successfully!")
	rollerCmd.RunApp(true)

	rollerCmd.WaitExit()
	bridgeCmd.WaitExit()
	coordinatorCmd.WaitExit()
}
