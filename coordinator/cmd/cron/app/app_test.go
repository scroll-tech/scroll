package app

import (
	"fmt"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/version"
)

func TestRunCoordinatorCron(t *testing.T) {
	coordinator := cmd.NewCmd("coordinator-cron-test", "--version")
	defer coordinator.WaitExit()

	// wait result
	coordinator.ExpectWithTimeout(t, true, time.Second*3, fmt.Sprintf("coordinator cron version %s", version.Version))
	coordinator.RunApp(nil)
}
