package app

import (
	"fmt"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/version"
)

func TestRunDatabase(t *testing.T) {
	dbcli := cmd.NewCmd("db_cli-test", "--version")
	defer dbcli.WaitExit()

	// wait result
	dbcli.ExpectWithTimeout(t, true, time.Second*3, fmt.Sprintf("db_cli version %s", version.Version))
	dbcli.RunApp(nil)
}
