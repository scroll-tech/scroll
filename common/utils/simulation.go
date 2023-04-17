package utils

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/reexec"
	"github.com/urfave/cli/v2"
)

type MockAppName string

var (
	EventWatcherApp   MockAppName = "event-watcher-test"
	GasOracleApp      MockAppName = "gas-oracle-test"
	MessageRelayerApp MockAppName = "message-relayer-test"
	RollupRelayerApp  MockAppName = "rollup-relayer-test"

	CoordinatorApp MockAppName = "coordinator-test"
	DBCliApp       MockAppName = "db_cli-test"
	RollerApp      MockAppName = "roller-test"
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
