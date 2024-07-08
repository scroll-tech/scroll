package tests

import (
	"crypto/rand"
	"math/big"
	"strconv"
	"testing"

	_ "scroll-tech/rollup/cmd/gas_oracle/app"
	_ "scroll-tech/rollup/cmd/rollup_relayer/app"

	"scroll-tech/common/database"
	cutils "scroll-tech/common/utils"

	"github.com/stretchr/testify/assert"
)

func testProcessStart(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)
	rollupApp.RunApp(t, cutils.GasOracleApp, "--genesis", "../conf/genesis.json")
	rollupApp.RunApp(t, cutils.RollupRelayerApp, "--genesis", "../conf/genesis.json")

	rollupApp.WaitExit()
}

func testProcessStartEnableMetrics(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	port, err := rand.Int(rand.Reader, big.NewInt(10000))
	assert.NoError(t, err)
	svrPort := strconv.FormatInt(port.Int64()+20000, 10)
	rollupApp.RunApp(t, cutils.GasOracleApp, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort, "--genesis", "../conf/genesis.json")

	port, err = rand.Int(rand.Reader, big.NewInt(10000))
	assert.NoError(t, err)
	svrPort = strconv.FormatInt(port.Int64()+30000, 10)
	rollupApp.RunApp(t, cutils.RollupRelayerApp, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort, "--genesis", "../conf/genesis.json")

	rollupApp.WaitExit()
}
