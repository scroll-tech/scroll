package coordinator_test

import (
	"compress/flate"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"scroll-tech/database/migrate"

	"scroll-tech/coordinator"
	client2 "scroll-tech/coordinator/client"
	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	"scroll-tech/coordinator/internal/utils"
	"scroll-tech/coordinator/verifier"

	"scroll-tech/common/docker"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	cutils "scroll-tech/common/utils"
)

var (
	dbCfg *config.DBConfig

	base *docker.App

	db         *gorm.DB
	l2BlockOrm *orm.L2Block
	chunkOrm   *orm.Chunk
	batchOrm   *orm.Batch

	wrappedBlock1 *types.WrappedBlock
	wrappedBlock2 *types.WrappedBlock
	chunk         *types.Chunk
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()
	m.Run()
	base.Free()
}

func randomURL() string {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000-1))
	return fmt.Sprintf("localhost:%d", 10000+2000+id.Int64())
}

func setEnv(t *testing.T) {
	base = docker.NewDockerApp()
	base.RunDBImage(t)

	dbCfg = &config.DBConfig{
		DSN:        base.DBConfig.DSN,
		DriverName: base.DBConfig.DriverName,
		MaxOpenNum: base.DBConfig.MaxOpenNum,
		MaxIdleNum: base.DBConfig.MaxIdleNum,
	}

	var err error
	db, err = utils.InitDB(dbCfg)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	batchOrm = orm.NewBatch(db)
	chunkOrm = orm.NewChunk(db)
	l2BlockOrm = orm.NewL2Block(db)

	templateBlockTrace, err := os.ReadFile("../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)
	wrappedBlock1 = &types.WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace, wrappedBlock1)
	assert.NoError(t, err)

	templateBlockTrace, err = os.ReadFile("../common/testdata/blockTrace_03.json")
	assert.NoError(t, err)
	wrappedBlock2 = &types.WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace, wrappedBlock2)
	assert.NoError(t, err)

	chunk = &types.Chunk{Blocks: []*types.WrappedBlock{wrappedBlock1, wrappedBlock2}}
	assert.NoError(t, err)
}

func TestApis(t *testing.T) {
	// Set up the test environment.
	base = docker.NewDockerApp()
	setEnv(t)

	t.Run("TestHandshake", testHandshake)
	t.Run("TestFailedHandshake", testFailedHandshake)
	t.Run("TestSeveralConnections", testSeveralConnections)
	t.Run("TestValidProof", testValidProof)
	t.Run("TestInvalidProof", testInvalidProof)
	t.Run("TestProofGeneratedFailed", testProofGeneratedFailed)
	t.Run("TestTimedoutProof", testTimedoutProof)
	t.Run("TestIdleRollerSelection", testIdleRollerSelection)
	t.Run("TestGracefulRestart", testGracefulRestart)
	t.Run("TestListRollers", testListRollers)

	// Teardown
	t.Cleanup(func() {
		base.Free()
	})
}

func testHandshake(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	rollerManager, handler := setupCoordinator(t, 1, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		rollerManager.Stop()
	}()

	roller := newMockRoller(t, "roller_test", wsURL)
	defer roller.close()

	assert.Equal(t, 1, rollerManager.GetNumberOfIdleRollers(message.ChunkProof))
}

func testFailedHandshake(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	rollerManager, handler := setupCoordinator(t, 1, wsURL, true)
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
			Timestamp: uint32(time.Now().Unix()),
		},
	}
	assert.NoError(t, authMsg.SignWithKey(privkey))
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
			Timestamp: uint32(time.Now().Unix()),
		},
	}
	assert.NoError(t, authMsg.SignWithKey(privkey))
	token, err := client.RequestToken(ctx, authMsg)
	assert.NoError(t, err)

	authMsg.Identity.Token = token
	assert.NoError(t, authMsg.SignWithKey(privkey))

	<-time.After(6 * time.Second)
	_, err = client.RegisterAndSubscribe(ctx, make(chan *message.TaskMsg, 4), authMsg)
	assert.Error(t, err)

	assert.Equal(t, 0, rollerManager.GetNumberOfIdleRollers(message.ChunkProof))
}

func testSeveralConnections(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	rollerManager, handler := setupCoordinator(t, 1, wsURL, true)
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
	assert.Equal(t, batch, rollerManager.GetNumberOfIdleRollers(message.ChunkProof))

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
			if rollerManager.GetNumberOfIdleRollers(message.ChunkProof) == 0 {
				return
			}
		case <-tickStop:
			t.Error("roller connect is blocked")
			return
		}
	}
}

func testValidProof(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	rollerManager, handler := setupCoordinator(t, 3, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		rollerManager.Stop()
	}()

	// create mock rollers.
	rollers := make([]*mockRoller, 3)
	for i := 0; i < len(rollers); i++ {
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i), wsURL)
		// only roller 0 submits valid proof.
		proofStatus := verifiedSuccess
		if i > 0 {
			proofStatus = generatedFailed
		}
		rollers[i].waitTaskAndSendProof(t, time.Second, false, proofStatus)
	}
	defer func() {
		// close connection
		for _, roller := range rollers {
			roller.close()
		}
	}()
	assert.Equal(t, 3, rollerManager.GetNumberOfIdleRollers(message.ChunkProof))

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(10 * time.Second)
	)
	for {
		select {
		case <-tick:
			status, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			if status == types.ProvingTaskVerified {
				return
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testInvalidProof(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	rollerManager, handler := setupCoordinator(t, 3, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		rollerManager.Stop()
	}()

	// create mock rollers.
	rollers := make([]*mockRoller, 3)
	for i := 0; i < len(rollers); i++ {
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i), wsURL)
		rollers[i].waitTaskAndSendProof(t, time.Second, false, verifiedFailed)
	}
	defer func() {
		// close connection
		for _, roller := range rollers {
			roller.close()
		}
	}()
	assert.Equal(t, 3, rollerManager.GetNumberOfIdleRollers(message.ChunkProof))

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(10 * time.Second)
	)
	for {
		select {
		case <-tick:
			status, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			if status == types.ProvingTaskFailed {
				return
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testProofGeneratedFailed(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	rollerManager, handler := setupCoordinator(t, 3, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		rollerManager.Stop()
	}()

	// create mock rollers.
	rollers := make([]*mockRoller, 3)
	for i := 0; i < len(rollers); i++ {
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i), wsURL)
		rollers[i].waitTaskAndSendProof(t, time.Second, false, generatedFailed)
	}
	defer func() {
		// close connection
		for _, roller := range rollers {
			roller.close()
		}
	}()
	assert.Equal(t, 3, rollerManager.GetNumberOfIdleRollers(message.ChunkProof))

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(10 * time.Second)
	)
	for {
		select {
		case <-tick:
			status, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			if status == types.ProvingTaskFailed {
				return
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testTimedoutProof(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	rollerManager, handler := setupCoordinator(t, 1, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		rollerManager.Stop()
	}()

	// create first mock roller, that will not send any proof.
	roller1 := newMockRoller(t, "roller_test"+strconv.Itoa(0), wsURL)
	defer func() {
		// close connection
		roller1.close()
	}()
	assert.Equal(t, 1, rollerManager.GetNumberOfIdleRollers(message.ChunkProof))

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)

	// verify proof status, it should be assigned, because roller didn't send any proof
	ok := cutils.TryTimes(30, func() bool {
		status, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
		if err != nil {
			return false
		}
		return status == types.ProvingTaskAssigned
	})
	assert.Falsef(t, !ok, "failed to check proof status")

	// create second mock roller, that will send valid proof.
	roller2 := newMockRoller(t, "roller_test"+strconv.Itoa(1), wsURL)
	roller2.waitTaskAndSendProof(t, time.Second, false, verifiedSuccess)
	defer func() {
		// close connection
		roller2.close()
	}()
	assert.Equal(t, 1, rollerManager.GetNumberOfIdleRollers(message.ChunkProof))

	// verify proof status, it should be verified now, because second roller sent valid proof
	ok = cutils.TryTimes(200, func() bool {
		status, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
		if err != nil {
			return false
		}
		return status == types.ProvingTaskVerified
	})
	assert.Falsef(t, !ok, "failed to check proof status")
}

func testIdleRollerSelection(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	rollerManager, handler := setupCoordinator(t, 1, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		rollerManager.Stop()
	}()

	// create mock rollers.
	rollers := make([]*mockRoller, 20)
	for i := 0; i < len(rollers); i++ {
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i), wsURL)
		rollers[i].waitTaskAndSendProof(t, time.Second, false, verifiedSuccess)
	}
	defer func() {
		// close connection
		for _, roller := range rollers {
			roller.close()
		}
	}()

	assert.Equal(t, len(rollers), rollerManager.GetNumberOfIdleRollers(message.ChunkProof))

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(10 * time.Second)
	)
	for {
		select {
		case <-tick:
			status, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			if status == types.ProvingTaskVerified {
				return
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testGracefulRestart(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	rollerManager, handler := setupCoordinator(t, 1, wsURL, true)

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)

	// create mock roller
	roller := newMockRoller(t, "roller_test", wsURL)
	// wait 10 seconds, coordinator restarts before roller submits proof
	roller.waitTaskAndSendProof(t, 10*time.Second, false, verifiedSuccess)

	// wait for coordinator to dispatch task
	<-time.After(5 * time.Second)
	// the coordinator will delete the roller if the subscription is closed.
	roller.close()

	info, err := rollerManager.GetSessionInfo(dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskAssigned.String(), info.Status)

	// Close rollerManager and ws handler.
	handler.Shutdown(context.Background())
	rollerManager.Stop()

	// Setup new coordinator and ws server.
	newRollerManager, newHandler := setupCoordinator(t, 1, wsURL, false)
	defer func() {
		newHandler.Shutdown(context.Background())
		newRollerManager.Stop()
	}()

	info, err = newRollerManager.GetSessionInfo(dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskAssigned.String(), info.Status)

	// at this point, roller haven't submitted
	status, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskAssigned, status)

	// will overwrite the roller client for `SubmitProof`
	roller.waitTaskAndSendProof(t, time.Second, true, verifiedSuccess)
	defer roller.close()

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(15 * time.Second)
	)
	for {
		select {
		case <-tick:
			// this proves that the roller submits to the new coordinator,
			// because the roller client for `submitProof` has been overwritten
			status, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			if status == types.ProvingTaskVerified {
				return
			}

		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testListRollers(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	rollerManager, handler := setupCoordinator(t, 1, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		rollerManager.Stop()
	}()

	var names = []string{
		"roller_test_1",
		"roller_test_2",
		"roller_test_3",
	}

	roller1 := newMockRoller(t, names[0], wsURL)
	roller2 := newMockRoller(t, names[1], wsURL)
	roller3 := newMockRoller(t, names[2], wsURL)
	defer func() {
		roller1.close()
		roller2.close()
	}()

	// test ListRollers API
	rollers, err := rollerManager.ListRollers()
	assert.NoError(t, err)
	var rollersName []string
	for _, roller := range rollers {
		rollersName = append(rollersName, roller.Name)
	}
	sort.Strings(rollersName)
	assert.True(t, reflect.DeepEqual(names, rollersName))

	// test ListRollers if one roller closed.
	roller3.close()
	// wait coordinator free completely
	time.Sleep(time.Second * 5)

	rollers, err = rollerManager.ListRollers()
	assert.NoError(t, err)
	var newRollersName []string
	for _, roller := range rollers {
		newRollersName = append(newRollersName, roller.Name)
	}
	sort.Strings(newRollersName)
	assert.True(t, reflect.DeepEqual(names[:2], newRollersName))
}

func setupCoordinator(t *testing.T, rollersPerSession uint8, wsURL string, resetDB bool) (rollerManager *coordinator.Manager, handler *http.Server) {
	db, err := utils.InitDB(dbCfg)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	if resetDB {
		assert.NoError(t, migrate.ResetDB(sqlDB))
	}

	rollerManager, err = coordinator.New(context.Background(), &config.RollerManagerConfig{
		RollersPerSession:  rollersPerSession,
		Verifier:           &config.VerifierConfig{MockMode: true},
		CollectionTime:     1,
		TokenTimeToLive:    5,
		MaxVerifierWorkers: 10,
		SessionAttempts:    2,
	}, db)
	assert.NoError(t, err)
	assert.NoError(t, rollerManager.Start())

	// start ws service
	handler, _, err = cutils.StartWSEndpoint(strings.Split(wsURL, "//")[1], rollerManager.APIs(), flate.NoCompression)
	assert.NoError(t, err)

	return rollerManager, handler
}

type mockRoller struct {
	rollerName string
	privKey    *ecdsa.PrivateKey

	wsURL  string
	client *client2.Client

	taskCh    chan *message.TaskMsg
	taskCache sync.Map

	sub    ethereum.Subscription
	stopCh chan struct{}
}

func newMockRoller(t *testing.T, rollerName string, wsURL string) *mockRoller {
	privKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	roller := &mockRoller{
		rollerName: rollerName,
		privKey:    privKey,
		wsURL:      wsURL,
		taskCh:     make(chan *message.TaskMsg, 4),
		stopCh:     make(chan struct{}),
	}
	roller.client, roller.sub, err = roller.connectToCoordinator()
	assert.NoError(t, err)

	return roller
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
			Timestamp: uint32(time.Now().Unix()),
		},
	}
	_ = authMsg.SignWithKey(r.privKey)

	token, err := client.RequestToken(context.Background(), authMsg)
	if err != nil {
		return nil, nil, err
	}
	authMsg.Identity.Token = token
	_ = authMsg.SignWithKey(r.privKey)

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

type proofStatus uint32

const (
	verifiedSuccess proofStatus = iota
	verifiedFailed
	generatedFailed
)

// Wait for the proof task, after receiving the proof task, roller submits proof after proofTime secs.
func (r *mockRoller) waitTaskAndSendProof(t *testing.T, proofTime time.Duration, reconnect bool, proofStatus proofStatus) {
	// simulating the case that the roller first disconnects and then reconnects to the coordinator
	// the Subscription and its `Err()` channel will be closed, and the coordinator will `freeRoller()`
	if reconnect {
		var err error
		r.client, r.sub, err = r.connectToCoordinator()
		if err != nil {
			t.Fatal(err)
			return
		}
	}

	// Release cached tasks.
	r.releaseTasks()

	r.stopCh = make(chan struct{})
	go r.loop(t, r.client, proofTime, proofStatus, r.stopCh)
}

func (r *mockRoller) loop(t *testing.T, client *client2.Client, proofTime time.Duration, proofStatus proofStatus, stopCh chan struct{}) {
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
			if proofStatus == generatedFailed {
				proof.Status = message.StatusProofError
			} else if proofStatus == verifiedFailed {
				proof.ProofDetail.Proof.Proof = []byte(verifier.InvalidTestProof)
			}
			assert.NoError(t, proof.Sign(r.privKey))
			assert.NoError(t, client.SubmitProof(context.Background(), proof))
		case <-stopCh:
			return
		}
	}
}

func (r *mockRoller) close() {
	close(r.stopCh)
	r.sub.Unsubscribe()
}
