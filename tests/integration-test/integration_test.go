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
)

func TestMain(m *testing.M) {
	base = newDockerApp()
	bridge = newBridgeApp(base)
	coordinator = newCoordinatorApp(base)
	rollers = append(rollers, newRollerApp(base))

	m.Run()

	base.free()
	bridge.free()
	coordinator.free()
	rollers.free()
}

func TestStartProcess(t *testing.T) {
	// Start l1geth l2geth postgres.
	base.runImages(t)

	// migrate db.
	base.runDBApp(t, "reset", "successful to reset")
	base.runDBApp(t, "migrate", "current version:")

	// Start bridge process.
	bridge.runApp(t)

	// Start coordinator process.
	coordinator.runApp(t)

	// Start rollers processes.
	rollers.runApps(t)

	bridge.WaitExit()
	rollers.WaitExit()
	coordinator.WaitExit()
}

func TestMonitorMetrics(t *testing.T) {
	// Start l1geth l2geth postgres.
	base.runImages(t)

	// migrate db.
	// migrate db.
	base.runDBApp(t, "reset", "successful to reset")
	base.runDBApp(t, "migrate", "current version:")

	// Start bridge process with metrics server.
	port, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort := strconv.FormatInt(port.Int64()+50000, 10)
	bridge.runApp(t, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort)

	// Get monitor metrics.
	resp, err := http.Get("http://localhost:" + svrPort)
	assert.NoError(t, err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	bodyStr := string(body)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, true, strings.Contains(bodyStr, "bridge_l1_msg_sync_height"))
	assert.Equal(t, true, strings.Contains(bodyStr, "bridge_l2_msg_sync_height"))

	bridge.WaitExit()
}
