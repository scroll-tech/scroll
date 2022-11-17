package integration_test

import (
	"testing"
	"time"

	"scroll-tech/common/docker"
)

func testBridgeCmd(t *testing.T) {
	cmd := docker.NewCmd(t)
	defer cmd.WaitExit()

	// wait result
	cmd.ExpectWithTimeout(time.Second*3, "bridge version prealpha-v4.1-")
	cmd.Run("bridge-test", "--version")
}

func testCoordinatorCmd(t *testing.T) {
	cmd := docker.NewCmd(t)
	defer cmd.WaitExit()

	// Wait expect result
	cmd.ExpectWithTimeout(time.Second*3, "coordinator version prealpha-v4.1-")
	cmd.Run("coordinator-test", "--version")
}

func testDatabaseCmd(t *testing.T) {
	cmd := docker.NewCmd(t)
	defer cmd.WaitExit()

	cmd.OpenLog(true)
	// Wait expect result
	cmd.ExpectWithTimeout(time.Second*3, "database version prealpha-v4.1-")
	cmd.Run("db_cli-test", "--log.debug", "--version")
}

func testDatabaseOperation(t *testing.T) {
	cmd := docker.NewCmd(t)
	defer cmd.WaitExit()

	cmd.OpenLog(true)

	// Wait reset result
	cmd.ExpectWithTimeout(time.Second*3, "successful to reset")
	cmd.Run("db_cli-test", "--log.debug", "reset", "--config", "../../database/config.json", "--db.dsn", dbImg.Endpoint())

	// Wait migrate result
	cmd.ExpectWithTimeout(time.Second*3, "current version: 5")
	cmd.Run("db_cli-test", "--log.debug", "migrate", "--config", "../../database/config.json", "--db.dsn", dbImg.Endpoint())
}

func testRollerCmd(t *testing.T) {
	cmd := docker.NewCmd(t)
	defer cmd.WaitExit()

	cmd.OpenLog(true)
	cmd.ExpectWithTimeout(time.Second*3, "Roller version prealpha-v4.1-")
	cmd.Run("roller-test", "--log.debug", "--version")
}
