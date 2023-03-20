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

	"scroll-tech/common/docker"

	"github.com/stretchr/testify/assert"
)

func TestIntegration(t *testing.T) {
	base = docker.NewDockerApp()
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
	ewCmd := runEventWatcherApp(t)
	ewCmd.RunApp(func() bool { return ewCmd.WaitResult(t, time.Second*20, "Start event-watcher successfully") })

	goCmd := runGasOracleApp(t)
	goCmd.RunApp(func() bool { return goCmd.WaitResult(t, time.Second*20, "Start gas-oracle successfully") })

	mrCmd := runMsgRelayerApp(t)
	mrCmd.RunApp(func() bool { return mrCmd.WaitResult(t, time.Second*20, "Start message-relayer successfully") })

	rrCmd := runRollupRelayerApp(t)
	rrCmd.RunApp(func() bool { return rrCmd.WaitResult(t, time.Second*20, "Start rollup-relayer successfully") })

	// Start coordinator process.
	coordinatorCmd := runCoordinatorApp(t, "--ws", "--ws.port", "8391")
	coordinatorCmd.RunApp(func() bool { return coordinatorCmd.WaitResult(t, time.Second*20, "Start coordinator successfully") })

	// Start roller process.
	rollerCmd := runRollerApp(t)
	rollerCmd.ExpectWithTimeout(t, true, time.Second*60, "register to coordinator successfully!")
	rollerCmd.RunApp(func() bool { return rollerCmd.WaitResult(t, time.Second*40, "roller start successfully") })

	rollerCmd.WaitExit()
	ewCmd.WaitExit()
	goCmd.WaitExit()
	mrCmd.WaitExit()
	rrCmd.WaitExit()
	coordinatorCmd.WaitExit()
}

func testMonitorMetrics(t *testing.T) {
	// migrate db.
	runDBCliApp(t, "reset", "successful to reset")
	runDBCliApp(t, "migrate", "current version:")

	// Start bridge process with metrics server.
	port1, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort1 := strconv.FormatInt(port1.Int64()+50000, 10)
	ewCmd := runEventWatcherApp(t, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort1)
	ewCmd.RunApp(func() bool { return ewCmd.WaitResult(t, time.Second*20, "Start event-watcher successfully") })

	port2, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort2 := strconv.FormatInt(port2.Int64()+50000, 10)
	goCmd := runGasOracleApp(t, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort2)
	goCmd.RunApp(func() bool { return goCmd.WaitResult(t, time.Second*20, "Start gas-oracle successfully") })

	port3, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort3 := strconv.FormatInt(port3.Int64()+50000, 10)
	mrCmd := runMsgRelayerApp(t, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort3)
	mrCmd.RunApp(func() bool { return mrCmd.WaitResult(t, time.Second*20, "Start message-relayer successfully") })

	port4, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort4 := strconv.FormatInt(port4.Int64()+50000, 10)
	rrCmd := runRollupRelayerApp(t, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort4)
	rrCmd.RunApp(func() bool { return rrCmd.WaitResult(t, time.Second*20, "Start rollup-relayer successfully") })

	// Start coordinator process with metrics server.
	port5, _ := rand.Int(rand.Reader, big.NewInt(2000))
	svrPort5 := strconv.FormatInt(port5.Int64()+52000, 10)
	coordinatorCmd := runCoordinatorApp(t, "--metrics", "--metrics.addr", "localhost", "--metrics.port", svrPort5)
	coordinatorCmd.RunApp(func() bool { return coordinatorCmd.WaitResult(t, time.Second*20, "Start coordinator successfully") })

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
	ewCmd.WaitExit()
	goCmd.WaitExit()
	mrCmd.WaitExit()
	rrCmd.WaitExit()
	coordinatorCmd.WaitExit()
}
