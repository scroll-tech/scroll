package test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types"
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
	urls := randmURLBatch(2)
	coordinatorURL := urls[0]
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL)
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	proxyURL := urls[1]
	proxyHttpHandler := setupProxy(t, proxyURL, []string{coordinatorURL})
	defer func() {
		assert.NoError(t, proxyHttpHandler.Shutdown(context.Background()))
	}()

	chunkProver := newMockProver(t, "prover_chunk_test", proxyURL, message.ProofTypeChunk, version.Version)
	code, msg := chunkProver.tryGetProverTask(t, message.ProofTypeChunk)
	assert.Equal(t, int(types.ErrCoordinatorEmptyProofData), code)

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)

	task, code, msg := chunkProver.getProverTask(t, message.ProofTypeChunk)
	assert.Empty(t, code)
	if code == 0 {
		t.Log("get task id", task.TaskID)
	} else {
		t.Log("get task error msg", msg)
	}

}

func testProxyProof(t *testing.T) {
	urls := randmURLBatch(3)
	coordinatorURL0 := urls[0]
	collector0, httpHandler0 := setupCoordinator(t, 3, coordinatorURL0)
	defer func() {
		collector0.Stop()
		httpHandler0.Shutdown(context.Background())
	}()
	coordinatorURL1 := urls[1]
	collector1, httpHandler1 := setupCoordinator(t, 3, coordinatorURL1)
	defer func() {
		collector1.Stop()
		httpHandler1.Shutdown(context.Background())
	}()
	coordinators := map[string]*http.Server{
		"coordinator_0": httpHandler0,
		"coordinator_1": httpHandler1,
	}

	proxyURL := urls[2]
	proxyHttpHandler := setupProxy(t, proxyURL, []string{coordinatorURL0, coordinatorURL1})
	defer func() {
		fmt.Println("px end start")
		assert.NoError(t, proxyHttpHandler.Shutdown(context.Background()))
		fmt.Println("px end")
	}()

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)

	chunkProver := newMockProver(t, "prover_chunk_test", proxyURL, message.ProofTypeChunk, version.Version)
	task, code, msg := chunkProver.getProverTask(t, message.ProofTypeChunk)
	assert.Empty(t, code)
	if code == 0 {
		t.Log("get task", task)
		parts, _, _ := strings.Cut(task.TaskID, ":")
		// close the coordinator which do not dispatch task first, so if we submit to wrong target,
		// there would be a chance the submit failed (to the closed coordinator)
		for n, srv := range coordinators {
			if n != parts {
				t.Log("close coordinator", n)
				assert.NoError(t, srv.Shutdown(context.Background()))
			}
		}
		exceptProofStatus := verifiedSuccess
		chunkProver.submitProof(t, task, exceptProofStatus, types.Success)

	} else {
		t.Log("get task error msg", msg)
	}

	// verify proof status
	var (
		tick     = time.Tick(1500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)

	var (
		chunkProofStatus    types.ProvingStatus
		chunkActiveAttempts int16
		chunkMaxAttempts    int16
	)

	for {
		select {
		case <-tick:
			chunkProofStatus, err = chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			if chunkProofStatus == types.ProvingTaskVerified {
				return
			}

			chunkActiveAttempts, chunkMaxAttempts, err = chunkOrm.GetAttemptsByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			assert.Equal(t, 1, int(chunkMaxAttempts))
			assert.Equal(t, 0, int(chunkActiveAttempts))

		case <-tickStop:
			t.Error("failed to check proof status", "chunkProofStatus", chunkProofStatus.String())
			return
		}
	}
}

func TestProxyClient(t *testing.T) {

	// Set up the test environment.
	setEnv(t)
	t.Run("TestProxyClient", testProxyClient)
	t.Run("TestProxyHandshake", testProxyHandshake)
	t.Run("TestProxyGetTask", testProxyGetTask)
	t.Run("TestProxyValidProof", testProxyProof)
}
