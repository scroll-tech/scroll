package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/version"
)

func TestRunProver(t *testing.T) {
	prover := cmd.NewCmd("prover-test", "--version")
	defer prover.WaitExit()

	// wait result
	prover.ExpectWithTimeout(t, true, time.Second*3, fmt.Sprintf("prover version %s-", strings.Split(version.Version, "-")[0]))
	prover.RunApp(nil)
}
