package utils

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

// LogConfig is for setup log types of the logger
type LogConfig struct {
	LogFile       string
	LogJSONFormat bool
	LogDebug      bool
	Verbosity     int
}

// LogSetup is for setup logger
func LogSetup(ctx *cli.Context) error {
	cfg := &LogConfig{
		LogFile:       ctx.String(LogFileFlag.Name),
		LogJSONFormat: ctx.Bool(LogJSONFormat.Name),
		LogDebug:      ctx.Bool(LogDebugFlag.Name),
		Verbosity:     ctx.Int(VerbosityFlag.Name),
	}

	var ostream log.Handler
	if logFile := cfg.LogFile; len(logFile) > 0 {
		fp, err := os.OpenFile(filepath.Clean(logFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			utils.Fatalf("Failed to open log file", "err", err)
		}
		if cfg.LogJSONFormat {
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
	log.PrintOrigins(cfg.LogDebug)
	glogger := log.NewGlogHandler(ostream)
	// Set log level
	glogger.Verbosity(log.Lvl(cfg.Verbosity))
	log.Root().SetHandler(glogger)
	return nil
}
