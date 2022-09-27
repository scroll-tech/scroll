package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/scroll-tech/go-ethereum/cmd/utils"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

func setup(ctx *cli.Context) error {
	var ostream log.Handler
	if logFile := ctx.String(logFileFlag.Name); len(logFile) > 0 {
		fp, err := os.OpenFile(filepath.Clean(logFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			utils.Fatalf("Failed to open log file", "err", err)
		}
		ostream = log.StreamHandler(io.Writer(fp), log.TerminalFormat(true))
	} else {
		output := io.Writer(os.Stderr)
		usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
		if usecolor {
			output = colorable.NewColorableStderr()
		}
		ostream = log.StreamHandler(output, log.TerminalFormat(usecolor))
	}
	glogger := log.NewGlogHandler(ostream)
	// Set log level
	glogger.Verbosity(log.Lvl(ctx.Int(verbosityFlag.Name)))
	log.Root().SetHandler(glogger)
	return nil
}
