package integration_test

import (
	"testing"
	"time"

	"scroll-tech/common/docker"
)

func TestVersion(t *testing.T) {
	// test cmd --version
	t.Run("testBridgeCmd", testBridgeCmd)
	t.Run("testCoordinatorCmd", testCoordinatorCmd)
	t.Run("testDatabaseCmd", testDatabaseCmd)
	t.Run("testRollerCmd", testRollerCmd)
}

func testBridgeCmd(t *testing.T) {
	cmd := runBridgeApp(t, "--version")

	// wait result
	cmd.ExpectWithTimeout(true, time.Second*3, "bridge version prealpha-v4.1-")
	cmd.WaitExit()
}

func testCoordinatorCmd(t *testing.T) {
	cmd := runCoordinatorApp(t, "--version")
	defer cmd.WaitExit()

	// Wait expect result
	cmd.ExpectWithTimeout(true, time.Second*3, "coordinator version prealpha-v4.1-")
}

func testDatabaseCmd(t *testing.T) {
	cmd := runDBCliApp(t, "--version")
	defer cmd.WaitExit()

	// Wait expect result
	cmd.ExpectWithTimeout(true, time.Second*3, "database version prealpha-v4.1-")
}

func testRollerCmd(t *testing.T) {
	cmd := docker.NewCmd(t)
	defer cmd.WaitExit()

	cmd.ExpectWithTimeout(true, time.Second*3, "Roller version prealpha-v4.1-")
	cmd.Run("roller-test", "--log.debug", "--version")
}
