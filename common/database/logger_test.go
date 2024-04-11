package database

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/scroll-tech/go-ethereum/log"
)

func TestGormLogger(t *testing.T) {
	output := io.Writer(os.Stderr)
	usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
	if usecolor {
		output = colorable.NewColorableStderr()
	}
	ostream := log.StreamHandler(output, log.TerminalFormat(usecolor))
	glogger := log.NewGlogHandler(ostream)
	// Set log level
	glogger.Verbosity(log.LvlTrace)
	log.Root().SetHandler(glogger)

	var gl gormLogger
	gl.gethLogger = log.Root()

	gl.Error(context.Background(), "test %s error:%v", "testError", errors.New("test error"))
	gl.Warn(context.Background(), "test %s warn:%v", "testWarn", errors.New("test warn"))
	gl.Info(context.Background(), "test %s warn:%v", "testInfo", errors.New("test info"))
	gl.Trace(context.Background(), time.Now(), func() (string, int64) { return "test trace", 1 }, nil)
}
