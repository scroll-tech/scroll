package integration_test

import (
	"testing"

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

func TestCoordinatorProverInteraction(t *testing.T) {
	// Start postgres docker containers.
	base.RunDBImage(t)
	// Reset db.
	assert.NoError(t, migrate.ResetDB(base.DBClient(t)))

	// Run coordinator app.
	coordinatorApp.RunApp(t)
	// Run prover app.
	proverApp.RunApp(t) // login success.

	// Free apps.
	proverApp.WaitExit()
	coordinatorApp.WaitExit()
}
