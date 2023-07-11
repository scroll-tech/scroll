package app

import (
	"fmt"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/version"
)

func TestRunCoordinator(t *testing.T) {
	coordinator := cmd.NewCmd("coordinator-test", "--version")
	defer coordinator.WaitExit()

	// wait result
	coordinator.ExpectWithTimeout(t, true, time.Second*3, fmt.Sprintf("coordinator version %s", version.Version))
	coordinator.RunApp(nil)
}
