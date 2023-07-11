package app

import (
	"fmt"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/version"
)

func TestRunRoller(t *testing.T) {
	roller := cmd.NewCmd("roller-test", "--version")
	defer roller.WaitExit()

	// wait result
	roller.ExpectWithTimeout(t, true, time.Second*3, fmt.Sprintf("roller version %s", version.Version))
	roller.RunApp(nil)
}
