package app

import (
	"fmt"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/version"
)

func TestRunRoller(t *testing.T) {
	bridge := cmd.NewCmd(t, "roller-test", "--version")
	defer bridge.WaitExit()

	// wait result
	bridge.ExpectWithTimeout(true, time.Second*3, fmt.Sprintf("roller version %s", version.Version))
	bridge.RunApp(false)
}
