package utils

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/reexec"
	"github.com/urfave/cli/v2"
)

// RegisterSimulation register initializer function for integration-test.
func RegisterSimulation(app *cli.App, name string) {
	// Run the app for integration-test
	reexec.Register(name, func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	reexec.Init()
}
