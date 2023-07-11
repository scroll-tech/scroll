package tests

import (
	"crypto/rand"
	"math/big"
	"strconv"
	"testing"

	_ "scroll-tech/bridge/cmd/event_watcher/app"
	_ "scroll-tech/bridge/cmd/gas_oracle/app"
	_ "scroll-tech/bridge/cmd/msg_relayer/app"
	_ "scroll-tech/bridge/cmd/rollup_relayer/app"

	"scroll-tech/common/database"
	cutils "scroll-tech/common/utils"

	"github.com/stretchr/testify/assert"
)

func testProcessStart(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	bridgeApp.RunApp(t, cutils.EventWatcherApp)
	bridgeApp.RunApp(t, cutils.GasOracleApp)
	bridgeApp.RunApp(t, cutils.MessageRelayerApp)
	bridgeApp.RunApp(t, cutils.RollupRelayerApp)

	bridgeApp.WaitExit()
}

func testProcessStartEnableMetrics(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	port, err := rand.Int(rand.Reader, big.NewInt(2000))
	assert.NoError(t, err)
	svrPort := strconv.FormatInt(port.Int64()+50000, 10)
	bridgeApp.RunApp(t, cutils.EventWatcherApp, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort)

	port, err = rand.Int(rand.Reader, big.NewInt(2000))
	assert.NoError(t, err)
	svrPort = strconv.FormatInt(port.Int64()+50000, 10)
	bridgeApp.RunApp(t, cutils.GasOracleApp, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort)

	port, err = rand.Int(rand.Reader, big.NewInt(2000))
	assert.NoError(t, err)
	svrPort = strconv.FormatInt(port.Int64()+50000, 10)
	bridgeApp.RunApp(t, cutils.MessageRelayerApp, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort)

	port, err = rand.Int(rand.Reader, big.NewInt(2000))
	assert.NoError(t, err)
	svrPort = strconv.FormatInt(port.Int64()+50000, 10)
	bridgeApp.RunApp(t, cutils.RollupRelayerApp, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort)

	bridgeApp.WaitExit()
}
