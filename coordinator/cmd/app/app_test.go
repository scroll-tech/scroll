package app

import (
	"fmt"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/version"
)

func TestRunCoordinator(t *testing.T) {
	bridge := cmd.NewCmd(t, "coordinator-test", "--version")
	defer bridge.WaitExit()

	// wait result
	bridge.ExpectWithTimeout(true, time.Second*3, fmt.Sprintf("coordinator version %s", version.Version))
	bridge.RunApp(false)
}
