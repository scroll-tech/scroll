package utils

import (
	"io"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

// LogSetup is for setup logger
func LogSetup(ctx *cli.Context) error {
	var ostream log.Handler
	if logFile := ctx.String(LogFileFlag.Name); len(logFile) > 0 {
		fp, err := os.OpenFile(filepath.Clean(logFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			utils.Fatalf("Failed to open log file", "err", err)
		}
		if ctx.Bool(LogJSONFormat.Name) {
			ostream = log.StreamHandler(io.Writer(fp), log.JSONFormat())
		} else {
			ostream = log.StreamHandler(io.Writer(fp), log.TerminalFormat(true))
		}
	} else {
		output := io.Writer(os.Stderr)
		usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
		if usecolor {
			output = colorable.NewColorableStderr()
		}
		ostream = log.StreamHandler(output, log.TerminalFormat(usecolor))
	}
	// show the call file and line number
	log.PrintOrigins(ctx.Bool(LogDebugFlag.Name))
	glogger := log.NewGlogHandler(ostream)
	// Set log level
	glogger.Verbosity(log.Lvl(ctx.Int(VerbosityFlag.Name)))
	log.Root().SetHandler(glogger)
	return nil
}
