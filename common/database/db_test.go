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
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"
	"scroll-tech/common/version"
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

func TestDB(t *testing.T) {
	version.Version = "v4.1.98-aaa-bbb-ccc"
	base := docker.NewDockerApp()
	base.RunDBImage(t)

	dbCfg := &Config{
		DSN:         base.DBConfig.DSN,
		DriverName:  base.DBConfig.DriverName,
		MaxOpenNum:  base.DBConfig.MaxOpenNum,
		MaxIdleNum:  base.DBConfig.MaxIdleNum,
		MaxLifetime: base.DBConfig.MaxLifetime,
		MaxIdleTime: base.DBConfig.MaxIdleTime,
	}

	var err error
	db, err := InitDB(dbCfg)
	assert.NoError(t, err)

	sqlDB, err := Ping(db)
	assert.NoError(t, err)
	assert.NotNil(t, sqlDB)

	assert.NoError(t, CloseDB(db))
}
