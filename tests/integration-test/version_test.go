package integration

import (
	"fmt"
	"testing"
	"time"

	"scroll-tech/common/version"
)

var (
	vs = version.Version
)

func TestVersion(t *testing.T) {
	// test cmd --version
	t.Run("testBridgeCmd", func(t *testing.T) {
		runDBCliApp(t, "--version", "")
	})
	t.Run("testCoordinatorCmd", testCoordinatorCmd)
	t.Run("testDatabaseCmd", testBridgeCmd)
	t.Run("testRollerCmd", testRollerCmd)
}

func testBridgeCmd(t *testing.T) {
	cmd := runBridgeApp(t, "--version")
	defer cmd.WaitExit()

	// wait result
	cmd.ExpectWithTimeout(true, time.Second*3, fmt.Sprintf("bridge version %s", vs))
	cmd.RunApp(false)
}

func testCoordinatorCmd(t *testing.T) {
	cmd := runCoordinatorApp(t, "--version")
	defer cmd.WaitExit()

	// Wait expect result
	cmd.ExpectWithTimeout(true, time.Second*3, fmt.Sprintf("coordinator version %s", vs))
	cmd.RunApp(false)
}

func testRollerCmd(t *testing.T) {
	cmd := runRollerApp(t, "--version")
	defer cmd.WaitExit()

	cmd.ExpectWithTimeout(true, time.Second*3, fmt.Sprintf("Roller version %s", vs))
	cmd.RunApp(false)
}
