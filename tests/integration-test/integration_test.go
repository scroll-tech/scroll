package integration

import (
	"crypto/rand"
	"io/ioutil"
	"math/big"
	"net/http"

	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"

	_ "scroll-tech/database/cmd/app"
	"scroll-tech/database/migrate"

	rApp "scroll-tech/roller/cmd/app"

	cApp "scroll-tech/coordinator/cmd/app"

	bApp "scroll-tech/bridge/cmd/app"
)

var (
	base        *docker.App
	bridge      *bApp.BridgeApp
	coordinator *cApp.CoordinatorApp
	rollers     rApp.RollerApps
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()
	bridge = bApp.NewBridgeApp(base, "../../bridge/config.json")
	coordinator = cApp.NewCoordinatorApp(base, "../../coordinator/config.json")
	rollers = append(rollers, rApp.NewRollerApp(base, "../../roller/config.json", coordinator.WSEndpoint()))

	m.Run()

	base.Free()
	bridge.Free()
	coordinator.Free()
	rollers.Free()
}

func TestStartProcess(t *testing.T) {
	// Start l1geth l2geth and postgres docker containers.
	base.RunImages(t)
	// reset db.
	assert.NoError(t, migrate.ResetDB(base.DBClient(t)))

	// Start bridge.
	bridge.RunApp(t)

	// Start coordinator, ws is enabled by default.
	coordinator.RunApp(t)

	// Start rollers.
	rollers.RunApps(t)

	rollers.WaitExit()
	coordinator.WaitExit()
	bridge.WaitExit()
}

func TestMonitorMetrics(t *testing.T) {
	// Start l1geth l2geth and postgres docker containers.
	base.RunImages(t)
	// reset db.
	assert.NoError(t, migrate.ResetDB(base.DBClient(t)))

	// Start bridge process with metrics server.
	port1, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort1 := strconv.FormatInt(port1.Int64()+50000, 10)
	// Start bridge and open metrics flag.
	bridge.RunApp(t, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort1)

	// Start coordinator process with metrics server.
	port, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort2 := strconv.FormatInt(port.Int64()+52000, 10)
	// Start coordinator.
	coordinator.RunApp(t, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort2)

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
	resp, err = http.Get("http://localhost:" + svrPort2)
	assert.NoError(t, err)
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	bodyStr = string(body)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, true, strings.Contains(bodyStr, "coordinator_sessions_timeout_total"))
	assert.Equal(t, true, strings.Contains(bodyStr, "coordinator_rollers_disconnects_total"))

	// Exit.
	coordinator.WaitExit()
	bridge.WaitExit()
}
