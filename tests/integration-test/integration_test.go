package integration_test

import (
	"crypto/rand"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	bcmd "scroll-tech/bridge/cmd"
	_ "scroll-tech/bridge/cmd/event_watcher/app"
	_ "scroll-tech/bridge/cmd/gas_oracle/app"
	_ "scroll-tech/bridge/cmd/msg_relayer/app"
	_ "scroll-tech/bridge/cmd/rollup_relayer/app"

	"scroll-tech/common/docker"
	"scroll-tech/common/utils"

	rapp "scroll-tech/roller/cmd/app"

	"scroll-tech/database/migrate"

	capp "scroll-tech/coordinator/cmd/app"
)

var (
	base           *docker.App
	bridgeApp      *bcmd.MockApp
	coordinatorApp *capp.CoordinatorApp
	rollerApp      *rapp.RollerApp
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()
	bridgeApp = bcmd.NewBridgeApp(base, "../../bridge/conf/config.json")
	coordinatorApp = capp.NewCoordinatorApp(base, "../../coordinator/config.json")
	rollerApp = rapp.NewRollerApp(base, "../../roller/config.json", coordinatorApp.WSEndpoint())
	m.Run()
	bridgeApp.Free()
	coordinatorApp.Free()
	rollerApp.Free()
	base.Free()
}

func TestStartProcess(t *testing.T) {
	// Start l1geth l2geth and postgres docker containers.
	base.RunImages(t)
	// Reset db.
	assert.NoError(t, migrate.ResetDB(base.DBClient(t)))

	// Run bridge apps.
	bridgeApp.RunApp(t, utils.EventWatcherApp)
	bridgeApp.RunApp(t, utils.GasOracleApp)
	bridgeApp.RunApp(t, utils.MessageRelayerApp)
	bridgeApp.RunApp(t, utils.RollupRelayerApp)

	// Run coordinator app.
	coordinatorApp.RunApp(t)
	// Run roller app.
	rollerApp.RunApp(t)

	// Free apps.
	bridgeApp.WaitExit()
	rollerApp.WaitExit()
	coordinatorApp.WaitExit()
}

func TestMonitorMetrics(t *testing.T) {
	// Start l1geth l2geth and postgres docker containers.
	base.RunImages(t)
	// Reset db.
	assert.NoError(t, migrate.ResetDB(base.DBClient(t)))

	port1, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort1 := strconv.FormatInt(port1.Int64()+50000, 10)
	bridgeApp.RunApp(t, utils.EventWatcherApp, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort1)

	port2, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort2 := strconv.FormatInt(port2.Int64()+50000, 10)
	bridgeApp.RunApp(t, utils.GasOracleApp, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort2)

	port3, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort3 := strconv.FormatInt(port3.Int64()+50000, 10)
	bridgeApp.RunApp(t, utils.MessageRelayerApp, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort3)

	port4, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort4 := strconv.FormatInt(port4.Int64()+50000, 10)
	bridgeApp.RunApp(t, utils.RollupRelayerApp, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort4)

	// Start coordinator process with metrics server.
	port5, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort5 := strconv.FormatInt(port5.Int64()+52000, 10)
	coordinatorApp.RunApp(t, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort5)

	// Get bridge monitor metrics.
	resp, err := http.Get("http://localhost:" + svrPort1)
	assert.NoError(t, err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	bodyStr := string(body)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, true, strings.Contains(bodyStr, "bridge_l1_msgs_sync_height"))
	assert.Equal(t, true, strings.Contains(bodyStr, "bridge_l2_msgs_sync_height"))

	// Get coordinator monitor metrics.
	resp, err = http.Get("http://localhost:" + svrPort5)
	assert.NoError(t, err)
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	bodyStr = string(body)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, true, strings.Contains(bodyStr, "coordinator_sessions_timeout_total"))
	assert.Equal(t, true, strings.Contains(bodyStr, "coordinator_rollers_disconnects_total"))

	// Exit.
	bridgeApp.WaitExit()
	coordinatorApp.WaitExit()
}
