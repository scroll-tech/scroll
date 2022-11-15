package integration_test

import (
	"scroll-tech/common/docker"
	"testing"
	"time"
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
	cmd.Run("database-test", "--version")
	cmd.WaitExit()
}

func testDatabase(t *testing.T) {

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
