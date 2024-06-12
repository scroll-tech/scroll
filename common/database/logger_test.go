package database

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/stretchr/testify/assert"
)

func TestGormLogger(t *testing.T) {
	var buf bytes.Buffer
	output := io.MultiWriter(os.Stderr, &buf)

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

	gl.Error(context.Background(), "test error: %s, %v", "testError", errors.New("test error"))
	gl.Warn(context.Background(), "test warn: %s, %v", "testWarn", errors.New("test warn"))
	gl.Info(context.Background(), "test info: %s, %v", "testInfo", errors.New("test info"))
	gl.Trace(context.Background(), time.Now(), func() (string, int64) { return "test trace", 1 }, nil)

	logOutput := buf.String()

	errorPattern := `ERROR\[\d{2}-\d{2}\|\d{2}:\d{2}:\d{2}\.\d{3}\] gorm\s+err message="test error: testError, test error"`
	warnPattern := `WARN \[\d{2}-\d{2}\|\d{2}:\d{2}:\d{2}\.\d{3}\] gorm\s+warn message="test warn: testWarn, test warn"`
	infoPattern := `INFO \[\d{2}-\d{2}\|\d{2}:\d{2}:\d{2}\.\d{3}\] gorm\s+info message="test info: testInfo, test info"`
	tracePattern := `DEBUG\[\d{2}-\d{2}\|\d{2}:\d{2}:\d{2}\.\d{3}\] gorm\s+line=.* cost=.* sql="test trace" rowsAffected=1 err=nil`

	assert.Regexp(t, regexp.MustCompile(errorPattern), logOutput, "Error log does not match expected pattern")
	assert.Regexp(t, regexp.MustCompile(warnPattern), logOutput, "Warn log does not match expected pattern")
	assert.Regexp(t, regexp.MustCompile(infoPattern), logOutput, "Info log does not match expected pattern")
	assert.Regexp(t, regexp.MustCompile(tracePattern), logOutput, "Trace log does not match expected pattern")
}
