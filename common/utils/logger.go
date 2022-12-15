package utils

import (
	"io"
	"os"
	"path/filepath"

	apollo_config "scroll-tech/common/apollo"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/scroll-tech/go-ethereum/cmd/utils"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

func getLogVerbose(defaultLevel int, use_apollo bool) log.Lvl {
	if !use_apollo {
		return log.Lvl(defaultLevel)
	}
	return log.Lvl(apollo_config.AgolloClient.GetIntValue("logVerbosity", defaultLevel))
}

// LogSetup is for setup logger
func LogSetup(ctx *cli.Context, use_apollo bool) error {
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
	glogger.Verbosity(getLogVerbose(ctx.Int(VerbosityFlag.Name), use_apollo))
	log.Root().SetHandler(glogger)
	return nil
}
