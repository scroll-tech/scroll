package test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/version"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/proxy"
)

func testProxyClientCfg() *config.ProxyClient {

	return &config.ProxyClient{
		Secret:       "test-secret-key",
		ProxyName:    "test-proxy",
		ProxyVersion: version.Version,
	}
}

var testCompatibileMode bool

func testProxyUpStreamCfg(coordinatorURL string) *config.UpStream {

	return &config.UpStream{
		BaseUrl:              fmt.Sprintf("http://%s", coordinatorURL),
		RetryWaitTime:        3,
		ConnectionTimeoutSec: 30,
		CompatibileMode:      testCompatibileMode,
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
	client := clientManager.ClientAsProxy(ctx)

	// Client should not be nil if login succeeds
	// Note: This might be nil if the coordinator is not properly set up for proxy authentication
	// but the test validates that the Client method completes without panic
	assert.NotNil(t, client)
	token1 := client.Token()
	assert.NotEmpty(t, token1)
	t.Logf("Client token: %s (%v)", token1, client)

	if !upCfg.CompatibileMode {
		time.Sleep(time.Second * 2)
		client.Reset()
		client = clientManager.ClientAsProxy(ctx)
		assert.NotNil(t, client)
		token2 := client.Token()
		assert.NotEmpty(t, token2)
		t.Logf("Client token (sec): %s (%v)", token2, client)
		assert.NotEqual(t, token1, token2, "token should not be identical")
	}

}

func testProxyHandshake(t *testing.T) {
	// Setup proxy http server.
	proxyURL := randomURL()
	proxyHttpHandler := launchProxy(t, proxyURL, []string{}, false)
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
	proxyHttpHandler := launchProxy(t, proxyURL, []string{coordinatorURL}, false)
	defer func() {
		assert.NoError(t, proxyHttpHandler.Shutdown(context.Background()))
	}()

	chunkProver := newMockProver(t, "prover_chunk_test", proxyURL, message.ProofTypeChunk, version.Version)
	chunkProver.setUseCacheToken(true)
	code, _ := chunkProver.tryGetProverTask(t, message.ProofTypeChunk)
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
	setupCoordinatorDb(t)
	collector0, httpHandler0 := launchCoordinator(t, 3, coordinatorURL0)
	defer func() {
		collector0.Stop()
		httpHandler0.Shutdown(context.Background())
	}()
	coordinatorURL1 := urls[1]
	collector1, httpHandler1 := launchCoordinator(t, 3, coordinatorURL1)
	defer func() {
		collector1.Stop()
		httpHandler1.Shutdown(context.Background())
	}()
	coordinators := map[string]*http.Server{
		"coordinator_0": httpHandler0,
		"coordinator_1": httpHandler1,
	}

	proxyURL := urls[2]
	proxyHttpHandler := launchProxy(t, proxyURL, []string{coordinatorURL0, coordinatorURL1}, false)
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
	chunkProver.setUseCacheToken(true)
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

func testProxyPersistent(t *testing.T) {
	urls := randmURLBatch(4)
	coordinatorURL0 := urls[0]
	setupCoordinatorDb(t)
	collector0, httpHandler0 := launchCoordinator(t, 3, coordinatorURL0)
	defer func() {
		collector0.Stop()
		httpHandler0.Shutdown(context.Background())
	}()
	coordinatorURL1 := urls[1]
	collector1, httpHandler1 := launchCoordinator(t, 3, coordinatorURL1)
	defer func() {
		collector1.Stop()
		httpHandler1.Shutdown(context.Background())
	}()

	setupProxyDb(t)
	proxyURL1 := urls[2]
	proxyHttpHandler := launchProxy(t, proxyURL1, []string{coordinatorURL0, coordinatorURL1}, true)
	defer func() {
		assert.NoError(t, proxyHttpHandler.Shutdown(context.Background()))
	}()

	proxyURL2 := urls[3]
	proxyHttpHandler2 := launchProxy(t, proxyURL2, []string{coordinatorURL0, coordinatorURL1}, true)
	defer func() {
		assert.NoError(t, proxyHttpHandler2.Shutdown(context.Background()))
	}()

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)

	chunkProver := newMockProver(t, "prover_chunk_test", proxyURL1, message.ProofTypeChunk, version.Version)
	chunkProver.setUseCacheToken(true)
	task, _, _ := chunkProver.getProverTask(t, message.ProofTypeChunk)
	assert.NotNil(t, task)
	taskFrom, _, _ := strings.Cut(task.TaskID, ":")
	t.Log("get task from coordinator:", taskFrom)

	chunkProver.resetConnection(proxyURL2)
	task, _, _ = chunkProver.getProverTask(t, message.ProofTypeChunk)
	assert.NotNil(t, task)
	taskFrom2, _, _ := strings.Cut(task.TaskID, ":")
	assert.Equal(t, taskFrom, taskFrom2)
}

func TestProxyClient(t *testing.T) {
	testCompatibileMode = false
	// Set up the test environment.
	setEnv(t)
	t.Run("TestProxyClient", testProxyClient)
	t.Run("TestProxyHandshake", testProxyHandshake)
	t.Run("TestProxyGetTask", testProxyGetTask)
	t.Run("TestProxyValidProof", testProxyProof)
	t.Run("testProxyPersistent", testProxyPersistent)
}

func TestProxyClientCompatibleMode(t *testing.T) {
	testCompatibileMode = true
	// Set up the test environment.
	setEnv(t)
	t.Run("TestProxyClient", testProxyClient)
	t.Run("TestProxyHandshake", testProxyHandshake)
	t.Run("TestProxyGetTask", testProxyGetTask)
	t.Run("TestProxyValidProof", testProxyProof)
	t.Run("testProxyPersistent", testProxyPersistent)
}
