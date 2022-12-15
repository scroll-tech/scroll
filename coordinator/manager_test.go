package coordinator_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum"

	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	"scroll-tech/coordinator"
	client2 "scroll-tech/coordinator/client"
	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"

	"scroll-tech/common/docker"
	"scroll-tech/common/message"
	"scroll-tech/common/utils"

	bridge_config "scroll-tech/bridge/config"

	coordinator_config "scroll-tech/coordinator/config"
)

var (
	cfg   *bridge_config.Config
	dbImg docker.ImgInstance
)

func randomUrl() string {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000-1))
	return fmt.Sprintf("localhost:%d", 10000+2000+id.Int64())
}

func setEnv(t *testing.T) (err error) {
	// Load config.
	cfg, err = bridge_config.NewConfig("../bridge/config.json")
	assert.NoError(t, err)

	// Create db container.
	dbImg = docker.NewTestDBDocker(t, cfg.DBConfig.DriverName)
	cfg.DBConfig.DSN = dbImg.Endpoint()

	return
}

func TestApis(t *testing.T) {
	// Set up the test environment.
	assert.True(t, assert.NoError(t, setEnv(t)), "failed to setup the test environment.")

	t.Run("TestHandshake", testHandshake)
	t.Run("TestFailedHandshake", testFailedHandshake)
	t.Run("TestSeveralConnections", testSeveralConnections)
	t.Run("TestIdleRollerSelection", testIdleRollerSelection)
	t.Run("TestRollerReconnect", testRollerReconnect)
	t.Run("TestGracefulRestart", testGracefulRestart)

	// Teardown
	t.Cleanup(func() {
		dbImg.Stop()
	})
}

func testHandshake(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	// Setup coordinator and ws server.
	wsURL := "ws://" + randomUrl()
	rollerManager, handler := setupCoordinator(t, cfg.DBConfig, wsURL)
	defer func() {
		handler.Shutdown(context.Background())
		rollerManager.Stop()
	}()

	roller := newMockRoller(t, "roller_test", wsURL)
	defer roller.close()

	assert.Equal(t, 1, rollerManager.GetNumberOfIdleRollers())
}

func testFailedHandshake(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	// Setup coordinator and ws server.
	wsURL := "ws://" + randomUrl()
	rollerManager, handler := setupCoordinator(t, cfg.DBConfig, wsURL)
	defer func() {
		handler.Shutdown(context.Background())
		rollerManager.Stop()
	}()

	// prepare
	name := "roller_test"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Try to perform handshake without token
	// create a new ws connection
	client, err := client2.DialContext(ctx, wsURL)
	assert.NoError(t, err)
	// create private key
	privkey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	authMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name:      name,
			Timestamp: time.Now().UnixNano(),
		},
	}
	assert.NoError(t, authMsg.Sign(privkey))
	_, err = client.RegisterAndSubscribe(ctx, make(chan *message.TaskMsg, 4), authMsg)
	assert.Error(t, err)

	// Try to perform handshake with timeouted token
	// create a new ws connection
	client, err = client2.DialContext(ctx, wsURL)
	assert.NoError(t, err)
	// create private key
	privkey, err = crypto.GenerateKey()
	assert.NoError(t, err)

	authMsg = &message.AuthMsg{
		Identity: &message.Identity{
			Name:      name,
			Timestamp: time.Now().UnixNano(),
		},
	}
	assert.NoError(t, authMsg.Sign(privkey))
	token, err := client.RequestToken(ctx, authMsg)
	assert.NoError(t, err)

	authMsg.Identity.Token = token
	assert.NoError(t, authMsg.Sign(privkey))

	<-time.After(6 * time.Second)
	_, err = client.RegisterAndSubscribe(ctx, make(chan *message.TaskMsg, 4), authMsg)
	assert.Error(t, err)

	assert.Equal(t, 0, rollerManager.GetNumberOfIdleRollers())
}

func testSeveralConnections(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	// Setup coordinator and ws server.
	wsURL := "ws://" + randomUrl()
	rollerManager, handler := setupCoordinator(t, cfg.DBConfig, wsURL)
	defer func() {
		handler.Shutdown(context.Background())
		rollerManager.Stop()
	}()

	var (
		batch   = 100
		eg      = errgroup.Group{}
		rollers = make([]*mockRoller, batch)
	)
	for i := 0; i < batch; i++ {
		idx := i
		eg.Go(func() error {
			rollers[idx] = newMockRoller(t, "roller_test_"+strconv.Itoa(idx), wsURL)
			return nil
		})
	}
	assert.NoError(t, eg.Wait())

	// check roller's idle connections
	assert.Equal(t, batch, rollerManager.GetNumberOfIdleRollers())

	// close connection
	for _, roller := range rollers {
		roller.close()
	}

	var (
		tick     = time.Tick(time.Second)
		tickStop = time.Tick(time.Second * 15)
	)
	for {
		select {
		case <-tick:
			if rollerManager.GetNumberOfIdleRollers() == 0 {
				return
			}
		case <-tickStop:
			t.Error("roller connect is blocked")
			return
		}
	}
}

func testIdleRollerSelection(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	// Setup coordinator and ws server.
	wsURL := "ws://" + randomUrl()
	rollerManager, handler := setupCoordinator(t, cfg.DBConfig, wsURL)
	defer func() {
		handler.Shutdown(context.Background())
		rollerManager.Stop()
	}()

	// create mock rollers.
	rollers := make([]*mockRoller, 20)
	for i := 0; i < len(rollers); i++ {
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i), wsURL)
		go rollers[i].waitTaskAndSendProof(t, time.Second)
	}
	defer func() {
		// close connection
		for _, roller := range rollers {
			roller.close()
		}
	}()

	assert.Equal(t, len(rollers), rollerManager.GetNumberOfIdleRollers())

	var ids = make([]string, 2)
	dbTx, err := l2db.Beginx()
	assert.NoError(t, err)
	for i := range ids {
		ID, err := l2db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(i)}, &orm.BlockInfo{Number: uint64(i)}, "0f", 1, 194676)
		assert.NoError(t, err)
		ids[i] = ID
	}
	assert.NoError(t, dbTx.Commit())

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(10 * time.Second)
	)
	for len(ids) > 0 {
		select {
		case <-tick:
			status, err := l2db.GetProvingStatusByID(ids[0])
			assert.NoError(t, err)
			if status == orm.ProvingTaskVerified {
				ids = ids[1:]
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testRollerReconnect(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	var ids = make([]string, 1)
	dbTx, err := l2db.Beginx()
	assert.NoError(t, err)
	for i := range ids {
		ID, err := l2db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(i)}, &orm.BlockInfo{Number: uint64(i)}, "0f", 1, 194676)
		assert.NoError(t, err)
		ids[i] = ID
	}
	assert.NoError(t, dbTx.Commit())

	// Setup coordinator and ws server.
	wsURL := "ws://" + randomUrl()
	rollerManager, handler := setupCoordinator(t, cfg.DBConfig, wsURL)
	defer func() {
		handler.Shutdown(context.Background())
		rollerManager.Stop()
	}()

	// create mock roller
	roller := newMockRoller(t, "roller_test", wsURL)
	defer roller.close()
	go roller.waitTaskAndSendProof(t, time.Second)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(15 * time.Second)
	)
	for len(ids) > 0 {
		select {
		case <-tick:
			status, err := l2db.GetProvingStatusByID(ids[0])
			assert.NoError(t, err)
			if status == orm.ProvingTaskVerified {
				ids = ids[1:]
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testGracefulRestart(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	var ids = make([]string, 1)
	dbTx, err := l2db.Beginx()
	assert.NoError(t, err)
	for i := range ids {
		ids[i], err = l2db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(i)}, &orm.BlockInfo{Number: uint64(i)}, "0f", 1, 194676)
		assert.NoError(t, err)
	}
	assert.NoError(t, dbTx.Commit())

	// Setup coordinator and ws server.
	wsURL := "ws://" + randomUrl()
	rollerManager, handler := setupCoordinator(t, cfg.DBConfig, wsURL)

	// create mock roller
	roller := newMockRoller(t, "roller_test", wsURL)
	// wait 5 seconds, coordinator restarts before roller submits proof
	go roller.waitTaskAndSendProof(t, 10*time.Second)

	// wait for coordinator to dispatch task
	<-time.After(5 * time.Second)
	// the coordinator will delete the roller if the subscription is closed.
	roller.close()

	// Close rollerManager and ws handler.
	handler.Shutdown(context.Background())
	rollerManager.Stop()

	// Setup new coordinator and ws server.
	newRollerManager, newHandler := setupCoordinator(t, cfg.DBConfig, wsURL)
	defer func() {
		newHandler.Shutdown(context.Background())
		newRollerManager.Stop()
	}()

	for i := range ids {
		info, err := newRollerManager.GetSessionInfo(ids[i])
		assert.Equal(t, orm.ProvingTaskAssigned.String(), info.Status)
		assert.NoError(t, err)

		// at this point, roller haven't submitted
		status, err := l2db.GetProvingStatusByID(ids[i])
		assert.NoError(t, err)
		assert.Equal(t, orm.ProvingTaskAssigned, status)
	}

	// will overwrite the roller client for `SubmitProof`
	go roller.waitTaskAndSendProof(t, time.Millisecond*500)
	defer roller.close()

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(15 * time.Second)
	)
	for len(ids) > 0 {
		select {
		case <-tick:
			// this proves that the roller submits to the new coordinator,
			// because the roller client for `submitProof` has been overwritten
			status, err := l2db.GetProvingStatusByID(ids[0])
			assert.NoError(t, err)
			if status == orm.ProvingTaskVerified {
				ids = ids[1:]
			}

		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func setupCoordinator(t *testing.T, dbCfg *database.DBConfig, wsURL string) (rollerManager *coordinator.Manager, handler *http.Server) {
	// Get db handler.
	db, err := database.NewOrmFactory(dbCfg)
	assert.True(t, assert.NoError(t, err), "failed to get db handler.")

	rollerManager, err = coordinator.New(context.Background(), &coordinator_config.RollerManagerConfig{
		RollersPerSession: 1,
		Verifier:          &coordinator_config.VerifierConfig{MockMode: true},
		CollectionTime:    1,
		TokenTimeToLive:   5,
	}, db, nil)
	assert.NoError(t, err)
	assert.NoError(t, rollerManager.Start())

	// start ws service
	handler, _, err = utils.StartWSEndpoint(strings.Split(wsURL, "//")[1], rollerManager.APIs())
	assert.NoError(t, err)

	return rollerManager, handler
}

type mockRoller struct {
	rollerName string
	privKey    *ecdsa.PrivateKey

	wsURL     string
	taskCh    chan *message.TaskMsg
	taskCache sync.Map

	mu     sync.Mutex
	stopCh chan struct{}
}

func newMockRoller(t *testing.T, rollerName string, wsURL string) *mockRoller {
	privKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	return &mockRoller{
		rollerName: rollerName,
		privKey:    privKey,
		wsURL:      wsURL,
		taskCh:     make(chan *message.TaskMsg, 4),
		stopCh:     make(chan struct{}),
	}
}

// connectToCoordinator sets up a websocket client to connect to the roller manager.
func (r *mockRoller) connectToCoordinator() (*client2.Client, ethereum.Subscription, error) {
	// Create connection.
	client, err := client2.Dial(r.wsURL)
	if err != nil {
		return nil, nil, err
	}

	// create a new ws connection
	authMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name:      r.rollerName,
			Timestamp: time.Now().UnixNano(),
		},
	}
	_ = authMsg.Sign(r.privKey)

	token, err := client.RequestToken(context.Background(), authMsg)
	if err != nil {
		return nil, nil, err
	}
	authMsg.Identity.Token = token
	_ = authMsg.Sign(r.privKey)

	sub, err := client.RegisterAndSubscribe(context.Background(), r.taskCh, authMsg)
	if err != nil {
		return nil, nil, err
	}

	return client, sub, nil
}

func (r *mockRoller) releaseTasks() {
	r.taskCache.Range(func(key, value any) bool {
		r.taskCh <- value.(*message.TaskMsg)
		r.taskCache.Delete(key)
		return true
	})
}

// Wait for the proof task, after receiving the proof task, roller submits proof after proofTime secs.
func (r *mockRoller) waitTaskAndSendProof(t *testing.T, proofTime time.Duration) {
	// simulating the case that the roller first disconnects and then reconnects to the coordinator
	// the Subscription and its `Err()` channel will be closed, and the coordinator will `freeRoller()`
	client, sub, err := r.connectToCoordinator()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer sub.Unsubscribe()

	// Release cached tasks.
	r.releaseTasks()

	r.mu.Lock()
	r.stopCh = make(chan struct{})
	stopCh := r.stopCh
	r.mu.Unlock()
	for {
		select {
		case task := <-r.taskCh:
			r.taskCache.Store(task.ID, task)
			// simulate proof time
			select {
			case <-time.After(proofTime):
			case <-stopCh:
				return
			}
			proof := &message.ProofMsg{
				ProofDetail: &message.ProofDetail{
					ID:     task.ID,
					Status: message.StatusOk,
					Proof:  &message.AggProof{},
				},
			}
			assert.NoError(t, proof.Sign(r.privKey))
			ok, err := client.SubmitProof(context.Background(), proof)
			assert.NoError(t, err)
			assert.Equal(t, true, ok)
		case <-stopCh:
			return
		}
	}
}

func (r *mockRoller) close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	close(r.stopCh)
}
