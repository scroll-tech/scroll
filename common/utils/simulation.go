package utils

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/reexec"
	"github.com/urfave/cli/v2"
)

// MockAppName a new type mock app.
type MockAppName string

var (
	// EventWatcherApp the name of mock event-watcher app.
	EventWatcherApp MockAppName = "event-watcher-test"
	// GasOracleApp the name of mock gas-oracle app.
	GasOracleApp MockAppName = "gas-oracle-test"
	// MessageRelayerApp the name of mock message-relayer app.
	MessageRelayerApp MockAppName = "message-relayer-test"
	// RollupRelayerApp the name of mock rollup-relayer app.
	RollupRelayerApp MockAppName = "rollup-relayer-test"
	// CoordinatorApp the name of mock coordinator app.
	CoordinatorApp MockAppName = "coordinator-test"
	// DBCliApp the name of mock database app.
	DBCliApp MockAppName = "db_cli-test"
	// RollerApp the name of mock prover app.
	RollerApp MockAppName = "prover-test"
)

// RegisterSimulation register initializer function for integration-test.
func RegisterSimulation(app *cli.App, name MockAppName) {
	// Run the app for integration-test
	reexec.Register(string(name), func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	reexec.Init()
}
