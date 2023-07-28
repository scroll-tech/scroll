package app

import (
	"fmt"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/version"
)

func TestRunRoller(t *testing.T) {
	prover := cmd.NewCmd("prover-test", "--version")
	defer prover.WaitExit()

	// wait result
	prover.ExpectWithTimeout(t, true, time.Second*3, fmt.Sprintf("prover version %s", version.Version))
	prover.RunApp(nil)
}
