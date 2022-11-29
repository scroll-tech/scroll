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
	cmd := docker.NewCmd(t)
	defer cmd.WaitExit()

	// wait result
	cmd.ExpectWithTimeout(true, time.Second*3, "bridge version prealpha-v4.1-")
	cmd.Run("bridge-test", "--version")
}

func testCoordinatorCmd(t *testing.T) {
	cmd := docker.NewCmd(t)
	defer cmd.WaitExit()

	// Wait expect result
	cmd.ExpectWithTimeout(true, time.Second*3, "coordinator version prealpha-v4.1-")
	cmd.Run("coordinator-test", "--version")
}

func testDatabaseCmd(t *testing.T) {
	cmd := docker.NewCmd(t)
	defer cmd.WaitExit()

	cmd.OpenLog(true)
	// Wait expect result
	cmd.ExpectWithTimeout(true, time.Second*3, "database version prealpha-v4.1-")
	cmd.Run("db_cli-test", "--log.debug", "--version")
}

func testRollerCmd(t *testing.T) {
	cmd := docker.NewCmd(t)
	defer cmd.WaitExit()

	cmd.OpenLog(true)
	cmd.ExpectWithTimeout(true, time.Second*3, "Roller version prealpha-v4.1-")
	cmd.Run("roller-test", "--log.debug", "--version")
}
