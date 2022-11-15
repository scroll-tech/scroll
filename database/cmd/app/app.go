package app

import (
	"fmt"
	"os"
)

// RunDatabase run database cmd instance.
func RunDatabase() {
	// Run the sequencer.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
