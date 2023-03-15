package app

import (
	"fmt"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/version"
)

func TestRunBridge(t *testing.T) {
	bridge := cmd.NewCmd("bridge-test", "--version")
	defer bridge.WaitExit()

	// wait result
	bridge.ExpectWithTimeout(t, true, time.Second*3, fmt.Sprintf("bridge version %s", version.Version))
	bridge.RunApp(nil)
}
