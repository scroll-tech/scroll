package integration_test

import (
	"testing"
	"time"

	"scroll-tech/common/docker"
)

func testBridgeCmd(t *testing.T) {
	cmd := docker.NewCmd(t)
	// wait result
	cmd.ExpectWithTimeout(time.Second*3, "bridge version prealpha-v4.1-")
	cmd.Run("bridge-test", "--version")
	cmd.WaitExit()
}

func testCoordinatorCmd(t *testing.T) {
	cmd := docker.NewCmd(t)
	// Wait expect result
	cmd.ExpectWithTimeout(time.Second*3, "coordinator version prealpha-v4.1-")
	cmd.Run("coordinator-test", "--version")
	cmd.WaitExit()
}

func testDatabaseCmd(t *testing.T) {
	cmd := docker.NewCmd(t)
	// Wait expect result
	cmd.ExpectWithTimeout(time.Second*3, "database version prealpha-v4.1-")
	cmd.Run("database-test", "--log.debug", "--version")
	cmd.WaitExit()
}

func testDatabaseOperation(t *testing.T) {
	cmd := docker.NewCmd(t)

	// Wait reset result
	cmd.ExpectWithTimeout(time.Second*3, "successful to reset")
	cmd.Run("database-test", "--log.debug", "reset", "--config", "../../database/config.json", "--db.dsn", dbImg.Endpoint())

	// Wait migrate result
	cmd.ExpectWithTimeout(time.Second*3, "current version: 5")
	cmd.Run("database-test", "--log.debug", "migrate", "--config", "../../database/config.json", "--db.dsn", dbImg.Endpoint())

	cmd.WaitExit()
}

/*func TestROllerCmd(t *testing.T) {
	cmd := &exec.Cmd{
		Path: reexec.Self(),
		Args: []string{"roller-test", "--help"},
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Error(err)
		fmt.Printf("Error running the reexec.Command - %s\n", err)
		os.Exit(1)
	}
}*/
