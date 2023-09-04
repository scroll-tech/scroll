package app

import (
	"fmt"
	"testing"
	"time"

	"scroll-tech/common/cmd"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"
)

func TestRunChunkProver(t *testing.T) {
	prover := cmd.NewCmd(string(utils.ChunkProverApp), "--version")
	defer prover.WaitExit()

	// wait result
	prover.ExpectWithTimeout(t, true, time.Second*3, fmt.Sprintf("prover version %s", version.Version))
	prover.RunApp(nil)
}

func TestRunBatchProver(t *testing.T) {
	prover := cmd.NewCmd(string(utils.BatchProverApp), "--version")
	defer prover.WaitExit()

	// wait result
	prover.ExpectWithTimeout(t, true, time.Second*3, fmt.Sprintf("prover version %s", version.Version))
	prover.RunApp(nil)
}
