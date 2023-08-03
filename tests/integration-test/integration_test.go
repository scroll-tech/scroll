package integration_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	rapp "scroll-tech/prover/cmd/app"

	"scroll-tech/database/migrate"

	capp "scroll-tech/coordinator/cmd/app"

	"scroll-tech/common/docker"

	bcmd "scroll-tech/bridge/cmd"
)

var (
	base           *docker.App
	bridgeApp      *bcmd.MockApp
	coordinatorApp *capp.CoordinatorApp
	proverApp      *rapp.ProverApp
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()
	bridgeApp = bcmd.NewBridgeApp(base, "../../bridge/conf/config.json")
	coordinatorApp = capp.NewCoordinatorApp(base, "../../coordinator/conf/config.json")
	proverApp = rapp.NewProverApp(base, "../../prover/config.json", coordinatorApp.HTTPEndpoint())
	m.Run()
	bridgeApp.Free()
	coordinatorApp.Free()
	proverApp.Free()
	base.Free()
}

func TestCoordinatorProverInteractionWithoutData(t *testing.T) {
	// Start postgres docker containers.
	base.RunDBImage(t)
	// Reset db.
	assert.NoError(t, migrate.ResetDB(base.DBClient(t)))

	base.RunL2Geth(t)

	// Run coordinator app.
	coordinatorApp.RunApp(t)
	// Run prover app.
	proverApp.RunApp(t) // login success.

	proverApp.ExpectWithTimeout(t, false, 60*time.Second, "get empty prover task") // get prover task without data.

	// Free apps.
	proverApp.WaitExit()
	coordinatorApp.WaitExit()
}
