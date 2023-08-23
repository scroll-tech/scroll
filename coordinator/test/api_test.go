package test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/database/migrate"

	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/version"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/controller/cron"
	"scroll-tech/coordinator/internal/orm"
	"scroll-tech/coordinator/internal/route"
)

var (
	dbCfg *database.Config
	conf  *config.Config

	base *docker.App

	db            *gorm.DB
	l2BlockOrm    *orm.L2Block
	chunkOrm      *orm.Chunk
	batchOrm      *orm.Batch
	proverTaskOrm *orm.ProverTask

	wrappedBlock1 *types.WrappedBlock
	wrappedBlock2 *types.WrappedBlock
	chunk         *types.Chunk

	tokenTimeout int
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

func setupCoordinator(t *testing.T, proversPerSession uint8, coordinatorURL string) (*cron.Collector, *http.Server) {
	var err error
	db, err = database.InitDB(dbCfg)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	tokenTimeout = 6
	conf = &config.Config{
		L2: &config.L2{
			ChainID: 111,
		},
		ProverManager: &config.ProverManager{
			ProversPerSession:      proversPerSession,
			Verifier:               &config.VerifierConfig{MockMode: true},
			BatchCollectionTimeSec: 10,
			ChunkCollectionTimeSec: 10,
			MaxVerifierWorkers:     10,
			SessionAttempts:        5,
		},
		Auth: &config.Auth{
			ChallengeExpireDurationSec: tokenTimeout,
			LoginExpireDurationSec:     tokenTimeout,
		},
	}

	proofCollector := cron.NewCollector(context.Background(), db, conf, nil)

	router := gin.New()
	api.InitController(conf, db, nil)
	route.Route(router, conf, nil)
	srv := &http.Server{
		Addr:    coordinatorURL,
		Handler: router,
	}
	go func() {
		runErr := srv.ListenAndServe()
		if runErr != nil && !errors.Is(runErr, http.ErrServerClosed) {
			assert.NoError(t, runErr)
		}
	}()
	time.Sleep(time.Second * 2)

	return proofCollector, srv
}

func setEnv(t *testing.T) {
	version.Version = "v4.1.97-aaa-bbb-ccc"

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
	proverTaskOrm = orm.NewProverTask(db)

	templateBlockTrace, err := os.ReadFile("../../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)
	wrappedBlock1 = &types.WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace, wrappedBlock1)
	assert.NoError(t, err)

	templateBlockTrace, err = os.ReadFile("../../common/testdata/blockTrace_03.json")
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
	t.Run("TestValidProof", testValidProof)
	t.Run("TestInvalidProof", testInvalidProof)
	t.Run("TestProofGeneratedFailed", testProofGeneratedFailed)
	t.Run("TestTimeoutProof", testTimeoutProof)

	// Teardown
	t.Cleanup(func() {
		base.Free()
	})
}

func testHandshake(t *testing.T) {
	// Setup coordinator and http server.
	coordinatorURL := randomURL()
	proofCollector, httpHandler := setupCoordinator(t, 1, coordinatorURL)
	defer func() {
		proofCollector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	chunkProver := newMockProver(t, "prover_chunk_test", coordinatorURL, message.ProofTypeChunk)
	assert.True(t, chunkProver.healthCheckSuccess(t))
}

func testFailedHandshake(t *testing.T) {
	// Setup coordinator and http server.
	coordinatorURL := randomURL()
	proofCollector, httpHandler := setupCoordinator(t, 1, coordinatorURL)
	defer func() {
		proofCollector.Stop()
	}()

	// Try to perform handshake without token
	chunkProver := newMockProver(t, "prover_chunk_test", coordinatorURL, message.ProofTypeChunk)
	assert.True(t, chunkProver.healthCheckSuccess(t))

	// Try to perform handshake with server shutdown
	assert.NoError(t, httpHandler.Shutdown(context.Background()))
	time.Sleep(time.Second)
	batchProver := newMockProver(t, "prover_batch_test", coordinatorURL, message.ProofTypeBatch)
	assert.True(t, batchProver.healthCheckFailure(t))
}

func testValidProof(t *testing.T) {
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL)
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), 0, 0, dbChunk.Hash, dbChunk.Hash, []*types.Chunk{chunk})
	assert.NoError(t, err)
	err = chunkOrm.UpdateBatchHashInRange(context.Background(), 0, 0, batch.Hash)
	assert.NoError(t, err)

	// create mock provers.
	provers := make([]*mockProver, 2)
	for i := 0; i < len(provers); i++ {
		var proofType message.ProofType
		if i%2 == 0 {
			proofType = message.ProofTypeChunk
		} else {
			proofType = message.ProofTypeBatch
		}
		provers[i] = newMockProver(t, "prover_test"+strconv.Itoa(i), coordinatorURL, proofType)

		// only prover 0 & 1 submit valid proofs.
		proofStatus := generatedFailed
		if i <= 1 {
			proofStatus = verifiedSuccess
		}
		proverTask := provers[i].getProverTask(t, proofType)
		assert.NotNil(t, proverTask)
		provers[i].submitProof(t, proverTask, proofStatus, types.Success)
	}

	// verify proof status
	var (
		tick     = time.Tick(1500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)

	var chunkProofStatus types.ProvingStatus
	var batchProofStatus types.ProvingStatus

	for {
		select {
		case <-tick:
			chunkProofStatus, err = chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			batchProofStatus, err = batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
			assert.NoError(t, err)
			if chunkProofStatus == types.ProvingTaskVerified && batchProofStatus == types.ProvingTaskVerified {
				return
			}
		case <-tickStop:
			t.Error("failed to check proof status", "chunkProofStatus", chunkProofStatus.String(), "batchProofStatus", batchProofStatus.String())
			return
		}
	}
}

func testInvalidProof(t *testing.T) {
	// Setup coordinator and ws server.
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL)
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), 0, 0, dbChunk.Hash, dbChunk.Hash, []*types.Chunk{chunk})
	assert.NoError(t, err)
	err = batchOrm.UpdateChunkProofsStatusByBatchHash(context.Background(), batch.Hash, types.ChunkProofsStatusReady)
	assert.NoError(t, err)

	// create mock provers.
	provers := make([]*mockProver, 2)
	for i := 0; i < len(provers); i++ {
		var proofType message.ProofType
		if i%2 == 0 {
			proofType = message.ProofTypeChunk
		} else {
			proofType = message.ProofTypeBatch
		}
		provers[i] = newMockProver(t, "prover_test"+strconv.Itoa(i), coordinatorURL, proofType)
		proverTask := provers[i].getProverTask(t, proofType)
		assert.NotNil(t, proverTask)
		provers[i].submitProof(t, proverTask, verifiedFailed, types.ErrCoordinatorHandleZkProofFailure)
	}

	// verify proof status
	var (
		tick     = time.Tick(1500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)

	var chunkProofStatus types.ProvingStatus
	var batchProofStatus types.ProvingStatus

	for {
		select {
		case <-tick:
			chunkProofStatus, err = chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			batchProofStatus, err = batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
			assert.NoError(t, err)
			if chunkProofStatus == types.ProvingTaskUnassigned && batchProofStatus == types.ProvingTaskUnassigned {
				return
			}
		case <-tickStop:
			t.Error("failed to check proof status", "chunkProofStatus", chunkProofStatus.String(), "batchProofStatus", batchProofStatus.String())
			return
		}
	}
}

func testProofGeneratedFailed(t *testing.T) {
	// Setup coordinator and ws server.
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL)
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), 0, 0, dbChunk.Hash, dbChunk.Hash, []*types.Chunk{chunk})
	assert.NoError(t, err)
	err = batchOrm.UpdateChunkProofsStatusByBatchHash(context.Background(), batch.Hash, types.ChunkProofsStatusReady)
	assert.NoError(t, err)

	// create mock provers.
	provers := make([]*mockProver, 2)
	for i := 0; i < len(provers); i++ {
		var proofType message.ProofType
		if i%2 == 0 {
			proofType = message.ProofTypeChunk
		} else {
			proofType = message.ProofTypeBatch
		}
		provers[i] = newMockProver(t, "prover_test"+strconv.Itoa(i), coordinatorURL, proofType)
		proverTask := provers[i].getProverTask(t, proofType)
		assert.NotNil(t, proverTask)
		provers[i].submitProof(t, proverTask, generatedFailed, types.ErrCoordinatorHandleZkProofFailure)
	}

	// verify proof status
	var (
		tick     = time.Tick(1500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)

	var (
		chunkProofStatus             types.ProvingStatus
		batchProofStatus             types.ProvingStatus
		chunkProverTaskProvingStatus types.ProverProveStatus
		batchProverTaskProvingStatus types.ProverProveStatus
	)

	for {
		select {
		case <-tick:
			chunkProofStatus, err = chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			batchProofStatus, err = batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
			assert.NoError(t, err)
			if chunkProofStatus == types.ProvingTaskAssigned && batchProofStatus == types.ProvingTaskAssigned {
				return
			}

			chunkProverTaskProvingStatus, err = proverTaskOrm.GetProvingStatusByTaskID(context.Background(), message.ProofTypeChunk, dbChunk.Hash)
			assert.NoError(t, err)
			batchProverTaskProvingStatus, err = proverTaskOrm.GetProvingStatusByTaskID(context.Background(), message.ProofTypeBatch, batch.Hash)
			assert.NoError(t, err)
			if chunkProverTaskProvingStatus == types.ProverProofInvalid && batchProverTaskProvingStatus == types.ProverProofInvalid {
				return
			}
		case <-tickStop:
			t.Error("failed to check proof status", "chunkProofStatus", chunkProofStatus.String(), "batchProofStatus", batchProofStatus.String())
			return
		}
	}
}

func testTimeoutProof(t *testing.T) {
	// Setup coordinator and ws server.
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 1, coordinatorURL)
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), 0, 0, dbChunk.Hash, dbChunk.Hash, []*types.Chunk{chunk})
	assert.NoError(t, err)
	err = batchOrm.UpdateChunkProofsStatusByBatchHash(context.Background(), batch.Hash, types.ChunkProofsStatusReady)
	assert.NoError(t, err)

	// create first chunk & batch mock prover, that will not send any proof.
	chunkProver1 := newMockProver(t, "prover_test"+strconv.Itoa(0), coordinatorURL, message.ProofTypeChunk)
	proverChunkTask := chunkProver1.getProverTask(t, message.ProofTypeChunk)
	assert.NotNil(t, proverChunkTask)

	batchProver1 := newMockProver(t, "prover_test"+strconv.Itoa(1), coordinatorURL, message.ProofTypeBatch)
	proverBatchTask := batchProver1.getProverTask(t, message.ProofTypeBatch)
	assert.NotNil(t, proverBatchTask)

	// verify proof status, it should be assigned, because prover didn't send any proof
	chunkProofStatus, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, chunkProofStatus, types.ProvingTaskAssigned)

	batchProofStatus, err := batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
	assert.NoError(t, err)
	assert.Equal(t, batchProofStatus, types.ProvingTaskAssigned)

	// wait coordinator to reset the prover task proving status
	time.Sleep(time.Duration(conf.ProverManager.BatchCollectionTimeSec*2) * time.Second)

	// create second mock prover, that will send valid proof.
	chunkProver2 := newMockProver(t, "prover_test"+strconv.Itoa(2), coordinatorURL, message.ProofTypeChunk)
	proverChunkTask2 := chunkProver2.getProverTask(t, message.ProofTypeChunk)
	assert.NotNil(t, proverChunkTask2)
	chunkProver2.submitProof(t, proverChunkTask2, verifiedSuccess, types.Success)

	batchProver2 := newMockProver(t, "prover_test"+strconv.Itoa(3), coordinatorURL, message.ProofTypeBatch)
	proverBatchTask2 := batchProver2.getProverTask(t, message.ProofTypeBatch)
	assert.NotNil(t, proverBatchTask2)
	batchProver2.submitProof(t, proverBatchTask2, verifiedSuccess, types.Success)

	// verify proof status, it should be verified now, because second prover sent valid proof
	chunkProofStatus2, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, chunkProofStatus2, types.ProvingTaskVerified)

	batchProofStatus2, err := batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
	assert.NoError(t, err)
	assert.Equal(t, batchProofStatus2, types.ProvingTaskVerified)
}
