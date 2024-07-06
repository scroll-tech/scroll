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
	// GasOracleApp the name of mock gas-oracle app.
	GasOracleApp MockAppName = "gas-oracle-test"
	// RollupRelayerApp the name of mock rollup-relayer app.
	RollupRelayerApp MockAppName = "rollup-relayer-test"

	// DBCliApp the name of mock database app.
	DBCliApp MockAppName = "db_cli-test"

	// CoordinatorAPIApp the name of mock coordinator app.
	CoordinatorAPIApp MockAppName = "coordinator-api-test"
	// CoordinatorCronApp the name of mock coordinator cron app.
	CoordinatorCronApp MockAppName = "coordinator-cron-test"
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
