package integration_test

import (
	"crypto/rand"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	bcmd "scroll-tech/bridge/cmd"

	"scroll-tech/common/docker"

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

	// Run coordinator app.
	coordinatorApp.RunApp(t)
	// Run roller app.
	rollerApp.RunApp(t)

	// Free apps.
	rollerApp.WaitExit()
	coordinatorApp.WaitExit()
}

func TestMonitorMetrics(t *testing.T) {
	// Start l1geth l2geth and postgres docker containers.
	base.RunImages(t)
	// Reset db.
	assert.NoError(t, migrate.ResetDB(base.DBClient(t)))

	// Start coordinator process with metrics server.
	port, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort := strconv.FormatInt(port.Int64()+52000, 10)
	coordinatorApp.RunApp(t, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort)

	time.Sleep(time.Second)

	// Get coordinator monitor metrics.
	resp, err := http.Get("http://localhost:" + svrPort)
	assert.NoError(t, err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	bodyStr := string(body)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, true, strings.Contains(bodyStr, "coordinator_sessions_timeout_total"))
	assert.Equal(t, true, strings.Contains(bodyStr, "coordinator_rollers_disconnects_total"))

	// Exit.
	coordinatorApp.WaitExit()
}
