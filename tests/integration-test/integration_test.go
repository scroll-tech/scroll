package integration_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	rapp "scroll-tech/prover/cmd/app"

	"scroll-tech/database/migrate"

	capp "scroll-tech/coordinator/cmd/app"

	"scroll-tech/common/docker"
	"scroll-tech/common/types/message"

	bcmd "scroll-tech/bridge/cmd"
)

var (
	base           *docker.App
	bridgeApp      *bcmd.MockApp
	coordinatorApp *capp.CoordinatorApp
	chunkProverApp *rapp.ProverApp
	batchProverApp *rapp.ProverApp
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()
	bridgeApp = bcmd.NewBridgeApp(base, "../../bridge/conf/config.json")
	coordinatorApp = capp.NewCoordinatorApp(base, "../../coordinator/conf/config.json")
	chunkProverApp = rapp.NewProverApp(base, "../../prover/config.json", coordinatorApp.HTTPEndpoint(), message.ProofTypeChunk)
	batchProverApp = rapp.NewProverApp(base, "../../prover/config.json", coordinatorApp.HTTPEndpoint(), message.ProofTypeBatch)
	m.Run()
	bridgeApp.Free()
	coordinatorApp.Free()
	chunkProverApp.Free()
	batchProverApp.Free()
	base.Free()
}

func TestCoordinatorProverInteractionWithoutData(t *testing.T) {
	// Start postgres docker containers.
	base.RunL2Geth(t)
	base.RunDBImage(t)
	// Reset db.
	assert.NoError(t, migrate.ResetDB(base.DBClient(t)))

	// Run coordinator app.
	coordinatorApp.RunApp(t)

	// Run prover app.
	chunkProverApp.RunApp(t) // chunk prover login.
	batchProverApp.RunApp(t) // batch prover login.

	chunkProverApp.ExpectWithTimeout(t, false, 60*time.Second, "get empty prover task") // get prover task without data.
	batchProverApp.ExpectWithTimeout(t, false, 60*time.Second, "get empty prover task") // get prover task without data.

	// Free apps.
	chunkProverApp.WaitExit()
	batchProverApp.WaitExit()
	coordinatorApp.WaitExit()
}
