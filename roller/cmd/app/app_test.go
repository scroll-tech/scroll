package app

import (
	"fmt"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/version"
)

func TestRunRoller(t *testing.T) {
	roller := cmd.NewCmd(t, "roller-test", "--version")
	defer roller.WaitExit()

	// wait result
	roller.ExpectWithTimeout(true, time.Second*3, fmt.Sprintf("roller version %s", version.Version))
	roller.RunApp(false)
}
