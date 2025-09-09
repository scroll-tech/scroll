package test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types/message"
	"scroll-tech/common/version"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/proxy"
	"scroll-tech/coordinator/internal/route"
)

func testProxyClientCfg() *config.ProxyClient {

	return &config.ProxyClient{
		Secret:       "test-secret-key",
		ProxyName:    "test-proxy",
		ProxyVersion: version.Version,
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
	assert.NotNil(t, client)
	assert.NotEmpty(t, client.Token())
	t.Logf("Client token: %s (%v)", client.Token(), client)
}

var (
	proxyConf *config.ProxyConfig
)

func setupProxy(t *testing.T, proxyURL string, coordinatorURL []string) *http.Server {
	var err error
	assert.NoError(t, err)

	coordinators := make(map[string]*config.UpStream)
	for i, n := range coordinatorURL {
		coordinators[fmt.Sprintf("coordinator_%d", i)] = testProxyUpStreamCfg(n)
	}

	tokenTimeout = 60
	proxyConf = &config.ProxyConfig{
		ProxyName: "test_proxy",
		ProxyManager: &config.ProxyManager{
			Verifier: &config.VerifierConfig{
				MinProverVersion: "v4.4.89",
				Verifiers: []config.AssetConfig{{
					AssetsPath: "",
					ForkName:   "euclidV2",
				}},
			},
			Client: testProxyClientCfg(),
			Auth: &config.Auth{
				Secret:                     "proxy",
				ChallengeExpireDurationSec: tokenTimeout,
				LoginExpireDurationSec:     tokenTimeout,
			},
		},
		Coordinators: coordinators,
	}

	router := gin.New()
	proxy.InitController(proxyConf, nil)
	route.ProxyRoute(router, proxyConf, nil)
	srv := &http.Server{
		Addr:    proxyURL,
		Handler: router,
	}
	go func() {
		runErr := srv.ListenAndServe()
		if runErr != nil && !errors.Is(runErr, http.ErrServerClosed) {
			assert.NoError(t, runErr)
		}
	}()
	time.Sleep(time.Second * 2)

	return srv
}

func testProxyHandshake(t *testing.T) {
	// Setup proxy http server.
	proxyURL := randomURL()
	proxyHttpHandler := setupProxy(t, proxyURL, []string{})
	defer func() {
		assert.NoError(t, proxyHttpHandler.Shutdown(context.Background()))
	}()

	chunkProver := newMockProver(t, "prover_chunk_test", proxyURL, message.ProofTypeChunk, version.Version)
	assert.True(t, chunkProver.healthCheckSuccess(t))
}

func testProxyGetTask(t *testing.T) {
	// Setup coordinator and http server.
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL)
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	proxyURL := randomURL()
	proxyHttpHandler := setupProxy(t, proxyURL, []string{coordinatorURL})
	defer func() {
		assert.NoError(t, proxyHttpHandler.Shutdown(context.Background()))
	}()

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)

	chunkProver := newMockProver(t, "prover_chunk_test", proxyURL, message.ProofTypeChunk, version.Version)
	code, _ := chunkProver.tryGetProverTask(t, message.ProofTypeChunk)
	assert.Empty(t, code)
}

func TestProxyClient(t *testing.T) {

	// Set up the test environment.
	setEnv(t)
	t.Run("TestProxyClient", testProxyClient)
	t.Run("TestProxyHandshake", testProxyHandshake)
	t.Run("TestProxyGetTask", testProxyGetTask)
}
