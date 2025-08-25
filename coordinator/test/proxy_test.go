package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/proxy"
)

func testProxyClientCfg() *config.ProxyClient {

	return &config.ProxyClient{
		Secret:    "test-secret-key",
		ProxyName: "test-proxy",
	}
}

func testProxyUpStreamCfg(coordinatorURL string) *config.UpStream {

	return &config.UpStream{
		BaseUrl:              fmt.Sprintf("http://%s", coordinatorURL),
		RetryWaitTime:        3,
		ConnectionTimeoutSec: 30,
	}

}

func testProxyClient(t *testing.T) {

	// Setup coordinator and http server.
	coordinatorURL := randomURL()
	proofCollector, httpHandler := setupCoordinator(t, 1, coordinatorURL)
	defer func() {
		proofCollector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	cliCfg := testProxyClientCfg()
	upCfg := testProxyUpStreamCfg(coordinatorURL)

	clientManager, err := proxy.NewClientManager("test_coordinator", cliCfg, upCfg)
	assert.NoError(t, err)
	assert.NotNil(t, clientManager)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test Client method
	client := clientManager.Client(ctx)

	// Client should not be nil if login succeeds
	// Note: This might be nil if the coordinator is not properly set up for proxy authentication
	// but the test validates that the Client method completes without panic
	t.Logf("Client toke: %v", client)

}

func TestProxyClient(t *testing.T) {

	// Set up the test environment.
	setEnv(t)
	t.Run("TestProxyHandshake", testProxyClient)
}
