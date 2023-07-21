package test

import (
	"compress/flate"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"scroll-tech/database/migrate"

	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	"scroll-tech/coordinator/client"
	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/controller/cron"
	"scroll-tech/coordinator/internal/logic/rollermanager"
	"scroll-tech/coordinator/internal/orm"
)

var (
	dbCfg *database.Config

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

func setupCoordinator(t *testing.T, rollersPerSession uint8, wsURL string, resetDB bool) (*http.Server, *cron.Collector) {
	var err error
	db, err = database.InitDB(dbCfg)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	if resetDB {
		assert.NoError(t, migrate.ResetDB(sqlDB))
	}

	conf := config.Config{
		RollerManagerConfig: &config.RollerManagerConfig{
			RollersPerSession:  rollersPerSession,
			Verifier:           &config.VerifierConfig{MockMode: true},
			CollectionTime:     1,
			TokenTimeToLive:    5,
			MaxVerifierWorkers: 10,
			SessionAttempts:    2,
		},
	}
	proofCollector := cron.NewCollector(context.Background(), db, &conf)
	tmpAPI := api.RegisterAPIs(&conf, db)
	handler, _, err := utils.StartWSEndpoint(strings.Split(wsURL, "//")[1], tmpAPI, flate.NoCompression)
	assert.NoError(t, err)
	rollermanager.InitRollerManager(db)
	return handler, proofCollector
}

func setEnv(t *testing.T) {
	base = docker.NewDockerApp()
	base.RunDBImage(t)

	dbCfg = &database.Config{
		DSN:        base.DBConfig.DSN,
		DriverName: base.DBConfig.DriverName,
		MaxOpenNum: base.DBConfig.MaxOpenNum,
		MaxIdleNum: base.DBConfig.MaxIdleNum,
	}

	var err error
	db, err = database.InitDB(dbCfg)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	batchOrm = orm.NewBatch(db)
	chunkOrm = orm.NewChunk(db)
	l2BlockOrm = orm.NewL2Block(db)

	templateBlockTrace, err := os.ReadFile("../testdata/blockTrace_02.json")
	assert.NoError(t, err)
	wrappedBlock1 = &types.WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace, wrappedBlock1)
	assert.NoError(t, err)

	templateBlockTrace, err = os.ReadFile("../testdata/blockTrace_03.json")
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
	t.Run("TestTimeoutProof", testTimeoutProof)
	t.Run("TestIdleRollerSelection", testIdleRollerSelection)
	t.Run("TestGracefulRestart", testGracefulRestart)

	// Teardown
	t.Cleanup(func() {
		base.Free()
	})
}

func testHandshake(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	handler, proofCollector := setupCoordinator(t, 1, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		proofCollector.Stop()
	}()

	roller1 := newMockRoller(t, "roller_test", wsURL, message.ProofTypeChunk)
	defer roller1.close()

	roller2 := newMockRoller(t, "roller_test", wsURL, message.ProofTypeBatch)
	defer roller2.close()

	assert.Equal(t, 1, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeChunk))
	assert.Equal(t, 1, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeBatch))
}

func testFailedHandshake(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	handler, proofCollector := setupCoordinator(t, 1, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		proofCollector.Stop()
	}()

	// prepare
	name := "roller_test"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Try to perform handshake without token
	// create a new ws connection
	c, err := client.DialContext(ctx, wsURL)
	assert.NoError(t, err)
	// create private key
	privkey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	authMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name: name,
		},
	}
	assert.NoError(t, authMsg.SignWithKey(privkey))
	_, err = c.RegisterAndSubscribe(ctx, make(chan *message.TaskMsg, 4), authMsg)
	assert.Error(t, err)

	// Try to perform handshake with timeouted token
	// create a new ws connection
	c, err = client.DialContext(ctx, wsURL)
	assert.NoError(t, err)
	// create private key
	privkey, err = crypto.GenerateKey()
	assert.NoError(t, err)

	authMsg = &message.AuthMsg{
		Identity: &message.Identity{
			Name: name,
		},
	}
	assert.NoError(t, authMsg.SignWithKey(privkey))
	token, err := c.RequestToken(ctx, authMsg)
	assert.NoError(t, err)

	authMsg.Identity.Token = token
	assert.NoError(t, authMsg.SignWithKey(privkey))

	<-time.After(6 * time.Second)
	_, err = c.RegisterAndSubscribe(ctx, make(chan *message.TaskMsg, 4), authMsg)
	assert.Error(t, err)

	assert.Equal(t, 0, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeChunk))
}

func testSeveralConnections(t *testing.T) {
	wsURL := "ws://" + randomURL()
	handler, proofCollector := setupCoordinator(t, 1, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		proofCollector.Stop()
	}()

	var (
		batch   = 200
		eg      = errgroup.Group{}
		rollers = make([]*mockRoller, batch)
	)
	for i := 0; i < batch; i += 2 {
		idx := i
		eg.Go(func() error {
			rollers[idx] = newMockRoller(t, "roller_test_"+strconv.Itoa(idx), wsURL, message.ProofTypeChunk)
			rollers[idx+1] = newMockRoller(t, "roller_test_"+strconv.Itoa(idx+1), wsURL, message.ProofTypeBatch)
			return nil
		})
	}
	assert.NoError(t, eg.Wait())

	// check roller's idle connections
	assert.Equal(t, batch/2, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeChunk))
	assert.Equal(t, batch/2, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeBatch))

	// close connection
	for _, roller := range rollers {
		roller.close()
	}

	var (
		tick     = time.Tick(time.Second)
		tickStop = time.Tick(time.Minute)
	)
	for {
		select {
		case <-tick:
			if rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeChunk) == 0 {
				return
			}
		case <-tickStop:
			t.Error("roller connect is blocked")
			return
		}
	}
}

func testValidProof(t *testing.T) {
	wsURL := "ws://" + randomURL()
	handler, collector := setupCoordinator(t, 3, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		collector.Stop()
	}()

	// create mock rollers.
	rollers := make([]*mockRoller, 6)
	for i := 0; i < len(rollers); i++ {
		var proofType message.ProofType
		if i%2 == 0 {
			proofType = message.ProofTypeChunk
		} else {
			proofType = message.ProofTypeBatch
		}
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i), wsURL, proofType)

		// only roller 0 & 1 submit valid proofs.
		proofStatus := generatedFailed
		if i <= 1 {
			proofStatus = verifiedSuccess
		}
		rollers[i].waitTaskAndSendProof(t, time.Second, false, proofStatus)
	}

	defer func() {
		// close connection
		for _, roller := range rollers {
			roller.close()
		}
	}()
	assert.Equal(t, 3, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeChunk))
	assert.Equal(t, 3, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeBatch))

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), 0, 0, dbChunk.Hash, dbChunk.Hash, []*types.Chunk{chunk})
	assert.NoError(t, err)
	err = chunkOrm.UpdateBatchHashInRange(context.Background(), 0, 0, batch.Hash)
	assert.NoError(t, err)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)
	for {
		select {
		case <-tick:
			chunkProofStatus, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			batchProofStatus, err := batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
			assert.NoError(t, err)
			if chunkProofStatus == types.ProvingTaskVerified && batchProofStatus == types.ProvingTaskVerified {
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
	handler, collector := setupCoordinator(t, 3, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		collector.Stop()
	}()

	// create mock rollers.
	rollers := make([]*mockRoller, 6)
	for i := 0; i < len(rollers); i++ {
		var proofType message.ProofType
		if i%2 == 0 {
			proofType = message.ProofTypeChunk
		} else {
			proofType = message.ProofTypeBatch
		}
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i), wsURL, proofType)
		rollers[i].waitTaskAndSendProof(t, time.Second, false, verifiedFailed)
	}
	defer func() {
		// close connection
		for _, roller := range rollers {
			roller.close()
		}
	}()
	assert.Equal(t, 3, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeChunk))
	assert.Equal(t, 3, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeBatch))

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), 0, 0, dbChunk.Hash, dbChunk.Hash, []*types.Chunk{chunk})
	assert.NoError(t, err)
	err = chunkOrm.UpdateChunkProofsStatusByBatchHash(context.Background(), batch.Hash, types.ChunkProofsStatusReady)
	assert.NoError(t, err)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)
	for {
		select {
		case <-tick:
			chunkProofStatus, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			batchProofStatus, err := batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
			assert.NoError(t, err)
			if chunkProofStatus == types.ProvingTaskFailed && batchProofStatus == types.ProvingTaskFailed {
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
	handler, collector := setupCoordinator(t, 3, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		collector.Stop()
	}()

	// create mock rollers.
	rollers := make([]*mockRoller, 6)
	for i := 0; i < len(rollers); i++ {
		var proofType message.ProofType
		if i%2 == 0 {
			proofType = message.ProofTypeChunk
		} else {
			proofType = message.ProofTypeBatch
		}
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i), wsURL, proofType)
		rollers[i].waitTaskAndSendProof(t, time.Second, false, generatedFailed)
	}
	defer func() {
		// close connection
		for _, roller := range rollers {
			roller.close()
		}
	}()
	assert.Equal(t, 3, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeChunk))
	assert.Equal(t, 3, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeBatch))

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), 0, 0, dbChunk.Hash, dbChunk.Hash, []*types.Chunk{chunk})
	assert.NoError(t, err)
	err = chunkOrm.UpdateChunkProofsStatusByBatchHash(context.Background(), batch.Hash, types.ChunkProofsStatusReady)
	assert.NoError(t, err)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)
	for {
		select {
		case <-tick:
			chunkProofStatus, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			batchProofStatus, err := batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
			assert.NoError(t, err)
			if chunkProofStatus == types.ProvingTaskFailed && batchProofStatus == types.ProvingTaskFailed {
				return
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testTimeoutProof(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	handler, collector := setupCoordinator(t, 1, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		collector.Stop()
	}()

	// create first chunk & batch mock roller, that will not send any proof.
	chunkRoller1 := newMockRoller(t, "roller_test"+strconv.Itoa(0), wsURL, message.ProofTypeChunk)
	batchRoller1 := newMockRoller(t, "roller_test"+strconv.Itoa(1), wsURL, message.ProofTypeBatch)
	defer func() {
		// close connection
		chunkRoller1.close()
		batchRoller1.close()
	}()
	assert.Equal(t, 1, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeChunk))
	assert.Equal(t, 1, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeBatch))

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), 0, 0, dbChunk.Hash, dbChunk.Hash, []*types.Chunk{chunk})
	assert.NoError(t, err)
	err = chunkOrm.UpdateChunkProofsStatusByBatchHash(context.Background(), batch.Hash, types.ChunkProofsStatusReady)
	assert.NoError(t, err)

	// verify proof status, it should be assigned, because roller didn't send any proof
	ok := utils.TryTimes(30, func() bool {
		chunkProofStatus, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
		if err != nil {
			return false
		}
		batchProofStatus, err := batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
		if err != nil {
			return false
		}
		return chunkProofStatus == types.ProvingTaskAssigned && batchProofStatus == types.ProvingTaskAssigned
	})
	assert.Falsef(t, !ok, "failed to check proof status")

	// create second mock roller, that will send valid proof.
	chunkRoller2 := newMockRoller(t, "roller_test"+strconv.Itoa(2), wsURL, message.ProofTypeChunk)
	chunkRoller2.waitTaskAndSendProof(t, time.Second, false, verifiedSuccess)
	batchRoller2 := newMockRoller(t, "roller_test"+strconv.Itoa(3), wsURL, message.ProofTypeBatch)
	batchRoller2.waitTaskAndSendProof(t, time.Second, false, verifiedSuccess)
	defer func() {
		// close connection
		chunkRoller2.close()
		batchRoller2.close()
	}()
	assert.Equal(t, 1, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeChunk))
	assert.Equal(t, 1, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeBatch))

	// verify proof status, it should be verified now, because second roller sent valid proof
	ok = utils.TryTimes(200, func() bool {
		chunkProofStatus, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
		if err != nil {
			return false
		}
		batchProofStatus, err := batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
		if err != nil {
			return false
		}
		return chunkProofStatus == types.ProvingTaskVerified && batchProofStatus == types.ProvingTaskVerified
	})
	assert.Falsef(t, !ok, "failed to check proof status")
}

func testIdleRollerSelection(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	handler, collector := setupCoordinator(t, 1, wsURL, true)
	defer func() {
		handler.Shutdown(context.Background())
		collector.Stop()
	}()

	// create mock rollers.
	rollers := make([]*mockRoller, 20)
	for i := 0; i < len(rollers); i++ {
		var proofType message.ProofType
		if i%2 == 0 {
			proofType = message.ProofTypeChunk
		} else {
			proofType = message.ProofTypeBatch
		}
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i), wsURL, proofType)
		rollers[i].waitTaskAndSendProof(t, time.Second, false, verifiedSuccess)
	}
	defer func() {
		// close connection
		for _, roller := range rollers {
			roller.close()
		}
	}()

	assert.Equal(t, len(rollers)/2, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeChunk))
	assert.Equal(t, len(rollers)/2, rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeBatch))

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), 0, 0, dbChunk.Hash, dbChunk.Hash, []*types.Chunk{chunk})
	assert.NoError(t, err)
	err = chunkOrm.UpdateBatchHashInRange(context.Background(), 0, 0, batch.Hash)
	assert.NoError(t, err)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(10 * time.Second)
	)
	for {
		select {
		case <-tick:
			chunkProofStatus, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			batchProofStatus, err := batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
			assert.NoError(t, err)
			if chunkProofStatus == types.ProvingTaskVerified && batchProofStatus == types.ProvingTaskVerified {
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
	handler, collector := setupCoordinator(t, 1, wsURL, true)

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), 0, 0, dbChunk.Hash, dbChunk.Hash, []*types.Chunk{chunk})
	assert.NoError(t, err)
	err = chunkOrm.UpdateBatchHashInRange(context.Background(), 0, 0, batch.Hash)
	assert.NoError(t, err)

	// create mock roller
	chunkRoller := newMockRoller(t, "roller_test", wsURL, message.ProofTypeChunk)
	batchRoller := newMockRoller(t, "roller_test", wsURL, message.ProofTypeBatch)
	// wait 10 seconds, coordinator restarts before roller submits proof
	chunkRoller.waitTaskAndSendProof(t, 10*time.Second, false, verifiedSuccess)
	batchRoller.waitTaskAndSendProof(t, 10*time.Second, false, verifiedSuccess)

	// wait for coordinator to dispatch task
	<-time.After(5 * time.Second)
	// the coordinator will delete the roller if the subscription is closed.
	chunkRoller.close()
	batchRoller.close()

	provingStatus, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskAssigned, provingStatus)

	// Close rollerManager and ws handler.
	handler.Shutdown(context.Background())
	collector.Stop()

	// Setup new coordinator and ws server.
	newHandler, newCollector := setupCoordinator(t, 1, wsURL, false)
	defer func() {
		newHandler.Shutdown(context.Background())
		newCollector.Stop()
	}()

	// at this point, roller haven't submitted
	status, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskAssigned, status)
	status, err = batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskUnassigned, status) // chunk proofs not ready yet

	// will overwrite the roller client for `SubmitProof`
	chunkRoller.waitTaskAndSendProof(t, time.Second, true, verifiedSuccess)
	batchRoller.waitTaskAndSendProof(t, time.Second, true, verifiedSuccess)
	defer func() {
		chunkRoller.close()
		batchRoller.close()
	}()

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
			chunkProofStatus, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			batchProofStatus, err := batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
			assert.NoError(t, err)
			if chunkProofStatus == types.ProvingTaskVerified && batchProofStatus == types.ProvingTaskVerified {
				return
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}
