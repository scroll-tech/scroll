package app

import (
	"fmt"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/version"
)

func TestRunDatabase(t *testing.T) {
	bridge := cmd.NewCmd("db_cli-test", "--version")
	defer bridge.WaitExit()

	// wait result
	bridge.ExpectWithTimeout(t, true, time.Second*3, fmt.Sprintf("db_cli version %s", version.Version))
	bridge.RunApp(nil)
}
