package integration

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
)

func TestIntegration(t *testing.T) {
	setupEnv(t)

	// test db_cli migrate cmd.
	t.Run("testDBClientMigrate", func(t *testing.T) {
		runDBCliApp(t, "migrate", "current version:")
	})

	// test bridge service
	t.Run("testStartProcess", testStartProcess)

	// test monitor metrics
	t.Run("testMonitorMetrics", testMonitorMetrics)

	t.Cleanup(func() {
		free(t)
	})
}

func testStartProcess(t *testing.T) {
	// migrate db.
	runDBCliApp(t, "reset", "successful to reset")
	runDBCliApp(t, "migrate", "current version:")

	// To do: recover this test
	// Start bridge process.
	// bridgeCmd := runBridgeApp(t)
	// bridgeCmd.RunApp(func() bool { return bridgeCmd.WaitResult(time.Second*20, "Start bridge successfully") })

	// Start coordinator process.
	coordinatorCmd := runCoordinatorApp(t, "--ws", "--ws.port", "8391")
	coordinatorCmd.RunApp(func() bool { return coordinatorCmd.WaitResult(time.Second*20, "Start coordinator successfully") })

	// Start roller process.
	rollerCmd := runRollerApp(t)
	rollerCmd.ExpectWithTimeout(true, time.Second*60, "register to coordinator successfully!")
	rollerCmd.RunApp(func() bool { return rollerCmd.WaitResult(time.Second*40, "roller start successfully") })

	rollerCmd.WaitExit()
	// bridgeCmd.WaitExit()
	coordinatorCmd.WaitExit()
}

func testMonitorMetrics(t *testing.T) {
	// migrate db.
	runDBCliApp(t, "reset", "successful to reset")
	runDBCliApp(t, "migrate", "current version:")

	// Start bridge process with metrics server.
	port, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort := strconv.FormatInt(port.Int64()+50000, 10)
	bridgeCmd := runBridgeApp(t, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort)
	bridgeCmd.RunApp(func() bool { return bridgeCmd.WaitResult(time.Second*20, "Start bridge successfully") })

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

	bridgeCmd.WaitExit()
}
