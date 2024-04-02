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
	"scroll-tech/common/database"
	"scroll-tech/common/testcontainers"
	tc "scroll-tech/common/testcontainers"
	"scroll-tech/common/types"
	"scroll-tech/common/types/encoding"
	"scroll-tech/common/types/message"
	"scroll-tech/common/version"
	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/controller/cron"
	"scroll-tech/coordinator/internal/orm"
	"scroll-tech/coordinator/internal/route"
	"scroll-tech/database/migrate"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

const (
	forkNumberFour  = 4
	forkNumberThree = 3
	forkNumberTwo   = 2
	forkNumberOne   = 1
)

var (
	dbCfg *database.Config
	conf  *config.Config

	testApps *testcontainers.TestcontainerApps

	db                 *gorm.DB
	l2BlockOrm         *orm.L2Block
	chunkOrm           *orm.Chunk
	batchOrm           *orm.Batch
	proverTaskOrm      *orm.ProverTask
	proverBlockListOrm *orm.ProverBlockList

	block1 *encoding.Block
	block2 *encoding.Block

	chunk          *encoding.Chunk
	hardForkChunk1 *encoding.Chunk
	hardForkChunk2 *encoding.Chunk

	batch          *encoding.Batch
	hardForkBatch1 *encoding.Batch
	hardForkBatch2 *encoding.Batch

	tokenTimeout int
)

func TestMain(m *testing.M) {
	defer func() {
		if testApps != nil {
			testApps.Free()
		}
	}()
	m.Run()
}

func randomURL() string {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000-1))
	return fmt.Sprintf("localhost:%d", 10000+2000+id.Int64())
}

func setupCoordinator(t *testing.T, proversPerSession uint8, coordinatorURL string, nameForkMap map[string]int64) (*cron.Collector, *http.Server) {
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
			MinProverVersion:       version.Version,
		},
		Auth: &config.Auth{
			ChallengeExpireDurationSec: tokenTimeout,
			LoginExpireDurationSec:     tokenTimeout,
		},
	}

	var chainConf params.ChainConfig
	for forkName, forkNumber := range nameForkMap {
		switch forkName {
		case "bernoulli":
			chainConf.BernoulliBlock = big.NewInt(forkNumber)
		case "london":
			chainConf.LondonBlock = big.NewInt(forkNumber)
		case "istanbul":
			chainConf.IstanbulBlock = big.NewInt(forkNumber)
		case "homestead":
			chainConf.HomesteadBlock = big.NewInt(forkNumber)
		case "eip155":
			chainConf.EIP155Block = big.NewInt(forkNumber)
		}
	}

	proofCollector := cron.NewCollector(context.Background(), db, conf, nil)

	router := gin.New()
	api.InitController(conf, &chainConf, db, nil)
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
	var err error

	version.Version = "v4.1.98"

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	testApps = tc.NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())

	dsn, err := testApps.GetDBEndPoint()
	assert.NoError(t, err)
	dbCfg = &database.Config{
		DSN:        dsn,
		DriverName: "postgres",
		MaxOpenNum: 200,
		MaxIdleNum: 20,
	}

	db, err = database.InitDB(dbCfg)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	batchOrm = orm.NewBatch(db)
	chunkOrm = orm.NewChunk(db)
	l2BlockOrm = orm.NewL2Block(db)
	proverTaskOrm = orm.NewProverTask(db)
	proverBlockListOrm = orm.NewProverBlockList(db)

	templateBlockTrace, err := os.ReadFile("../../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)
	block1 = &encoding.Block{}
	err = json.Unmarshal(templateBlockTrace, block1)
	assert.NoError(t, err)

	templateBlockTrace, err = os.ReadFile("../../common/testdata/blockTrace_03.json")
	assert.NoError(t, err)
	block2 = &encoding.Block{}
	err = json.Unmarshal(templateBlockTrace, block2)
	assert.NoError(t, err)

	chunk = &encoding.Chunk{Blocks: []*encoding.Block{block1, block2}}
	hardForkChunk1 = &encoding.Chunk{Blocks: []*encoding.Block{block1}}
	hardForkChunk2 = &encoding.Chunk{Blocks: []*encoding.Block{block2}}

	assert.NoError(t, err)

	batch = &encoding.Batch{Chunks: []*encoding.Chunk{chunk}}
	hardForkBatch1 = &encoding.Batch{Index: 1, Chunks: []*encoding.Chunk{hardForkChunk1}}
	hardForkBatch2 = &encoding.Batch{Index: 2, Chunks: []*encoding.Chunk{hardForkChunk2}}
}

func TestApis(t *testing.T) {
	// Set up the test environment.
	setEnv(t)

	t.Run("TestHandshake", testHandshake)
	t.Run("TestFailedHandshake", testFailedHandshake)
	t.Run("TestGetTaskBlocked", testGetTaskBlocked)
	t.Run("TestOutdatedProverVersion", testOutdatedProverVersion)
	t.Run("TestValidProof", testValidProof)
	t.Run("TestInvalidProof", testInvalidProof)
	t.Run("TestProofGeneratedFailed", testProofGeneratedFailed)
	t.Run("TestTimeoutProof", testTimeoutProof)
	t.Run("TestHardFork", testHardForkAssignTask)
}

func testHandshake(t *testing.T) {
	// Setup coordinator and http server.
	coordinatorURL := randomURL()
	proofCollector, httpHandler := setupCoordinator(t, 1, coordinatorURL, map[string]int64{"homestead": forkNumberOne})
	defer func() {
		proofCollector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	chunkProver := newMockProver(t, "prover_chunk_test", coordinatorURL, message.ProofTypeChunk, version.Version)
	assert.True(t, chunkProver.healthCheckSuccess(t))
}

func testFailedHandshake(t *testing.T) {
	// Setup coordinator and http server.
	coordinatorURL := randomURL()
	proofCollector, httpHandler := setupCoordinator(t, 1, coordinatorURL, map[string]int64{"homestead": forkNumberOne})
	defer func() {
		proofCollector.Stop()
	}()

	// Try to perform handshake without token
	chunkProver := newMockProver(t, "prover_chunk_test", coordinatorURL, message.ProofTypeChunk, version.Version)
	assert.True(t, chunkProver.healthCheckSuccess(t))

	// Try to perform handshake with server shutdown
	assert.NoError(t, httpHandler.Shutdown(context.Background()))
	time.Sleep(time.Second)
	batchProver := newMockProver(t, "prover_batch_test", coordinatorURL, message.ProofTypeBatch, version.Version)
	assert.True(t, batchProver.healthCheckFailure(t))
}

func testGetTaskBlocked(t *testing.T) {
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL, map[string]int64{"homestead": forkNumberOne})
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	chunkProver := newMockProver(t, "prover_chunk_test", coordinatorURL, message.ProofTypeChunk, version.Version)
	assert.True(t, chunkProver.healthCheckSuccess(t))

	batchProver := newMockProver(t, "prover_batch_test", coordinatorURL, message.ProofTypeBatch, version.Version)
	assert.True(t, batchProver.healthCheckSuccess(t))

	err := proverBlockListOrm.InsertProverPublicKey(context.Background(), chunkProver.proverName, chunkProver.publicKey())
	assert.NoError(t, err)

	expectedErr := fmt.Errorf("return prover task err:check prover task parameter failed, error:public key %s is blocked from fetching tasks. ProverName: %s, ProverVersion: %s", chunkProver.publicKey(), chunkProver.proverName, chunkProver.proverVersion)
	code, errMsg := chunkProver.tryGetProverTask(t, message.ProofTypeChunk)
	assert.Equal(t, types.ErrCoordinatorGetTaskFailure, code)
	assert.Equal(t, expectedErr, fmt.Errorf(errMsg))

	expectedErr = fmt.Errorf("get empty prover task")
	code, errMsg = batchProver.tryGetProverTask(t, message.ProofTypeBatch)
	assert.Equal(t, types.ErrCoordinatorEmptyProofData, code)
	assert.Equal(t, expectedErr, fmt.Errorf(errMsg))

	err = proverBlockListOrm.InsertProverPublicKey(context.Background(), batchProver.proverName, batchProver.publicKey())
	assert.NoError(t, err)

	err = proverBlockListOrm.DeleteProverPublicKey(context.Background(), chunkProver.publicKey())
	assert.NoError(t, err)

	expectedErr = fmt.Errorf("get empty prover task")
	code, errMsg = chunkProver.tryGetProverTask(t, message.ProofTypeChunk)
	assert.Equal(t, types.ErrCoordinatorEmptyProofData, code)
	assert.Equal(t, expectedErr, fmt.Errorf(errMsg))

	expectedErr = fmt.Errorf("return prover task err:check prover task parameter failed, error:public key %s is blocked from fetching tasks. ProverName: %s, ProverVersion: %s", batchProver.publicKey(), batchProver.proverName, batchProver.proverVersion)
	code, errMsg = batchProver.tryGetProverTask(t, message.ProofTypeBatch)
	assert.Equal(t, types.ErrCoordinatorGetTaskFailure, code)
	assert.Equal(t, expectedErr, fmt.Errorf(errMsg))
}

func testOutdatedProverVersion(t *testing.T) {
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL, map[string]int64{"homestead": forkNumberOne})
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	chunkProver := newMockProver(t, "prover_chunk_test", coordinatorURL, message.ProofTypeChunk, "v1.0.0")
	assert.True(t, chunkProver.healthCheckSuccess(t))

	batchProver := newMockProver(t, "prover_batch_test", coordinatorURL, message.ProofTypeBatch, "v1.999.999")
	assert.True(t, chunkProver.healthCheckSuccess(t))

	expectedErr := fmt.Errorf("return prover task err:check prover task parameter failed, error:incompatible prover version. please upgrade your prover, minimum allowed version: %s, actual version: %s", version.Version, chunkProver.proverVersion)
	code, errMsg := chunkProver.tryGetProverTask(t, message.ProofTypeChunk)
	assert.Equal(t, types.ErrCoordinatorGetTaskFailure, code)
	assert.Equal(t, expectedErr, fmt.Errorf(errMsg))

	expectedErr = fmt.Errorf("return prover task err:check prover task parameter failed, error:incompatible prover version. please upgrade your prover, minimum allowed version: %s, actual version: %s", version.Version, batchProver.proverVersion)
	code, errMsg = batchProver.tryGetProverTask(t, message.ProofTypeBatch)
	assert.Equal(t, types.ErrCoordinatorGetTaskFailure, code)
	assert.Equal(t, expectedErr, fmt.Errorf(errMsg))
}

func testHardForkAssignTask(t *testing.T) {
	tests := []struct {
		name                  string
		proofType             message.ProofType
		forkNumbers           map[string]int64
		proverForkNames       []string
		exceptTaskNumber      int
		exceptGetTaskErrCodes []int
		exceptGetTaskErrMsgs  []string
	}{
		{ // hard fork 4, prover 4  block [2-3]
			name:                  "noTaskForkChunkProverVersionLargeOrEqualThanHardFork",
			proofType:             message.ProofTypeChunk,
			forkNumbers:           map[string]int64{"bernoulli": forkNumberFour},
			exceptTaskNumber:      0,
			proverForkNames:       []string{"bernoulli", "bernoulli"},
			exceptGetTaskErrCodes: []int{types.ErrCoordinatorEmptyProofData, types.ErrCoordinatorEmptyProofData},
			exceptGetTaskErrMsgs:  []string{"get empty prover task", "get empty prover task"},
		},
		{
			name:                  "noTaskForkBatchProverVersionLargeOrEqualThanHardFork",
			proofType:             message.ProofTypeBatch,
			forkNumbers:           map[string]int64{"bernoulli": forkNumberFour},
			exceptTaskNumber:      0,
			proverForkNames:       []string{"bernoulli", "bernoulli"},
			exceptGetTaskErrCodes: []int{types.ErrCoordinatorEmptyProofData, types.ErrCoordinatorEmptyProofData},
			exceptGetTaskErrMsgs:  []string{"get empty prover task", "get empty prover task"},
		},
		{ // hard fork 1, prover 1 block [2-3]
			name:                  "noTaskForkChunkProverVersionLessThanHardFork",
			proofType:             message.ProofTypeChunk,
			forkNumbers:           map[string]int64{"istanbul": forkNumberTwo, "homestead": forkNumberOne},
			exceptTaskNumber:      0,
			proverForkNames:       []string{"homestead", "homestead"},
			exceptGetTaskErrCodes: []int{types.ErrCoordinatorEmptyProofData, types.ErrCoordinatorEmptyProofData},
			exceptGetTaskErrMsgs:  []string{"get empty prover task", "get empty prover task"},
		},
		{
			name:                  "noTaskForkBatchProverVersionLessThanHardFork",
			proofType:             message.ProofTypeBatch,
			forkNumbers:           map[string]int64{"istanbul": forkNumberTwo, "homestead": forkNumberOne},
			exceptTaskNumber:      0,
			proverForkNames:       []string{"homestead", "homestead"},
			exceptGetTaskErrCodes: []int{types.ErrCoordinatorEmptyProofData, types.ErrCoordinatorEmptyProofData},
			exceptGetTaskErrMsgs:  []string{"get empty prover task", "get empty prover task"},
		},
		{
			name:                  "noTaskForkBatchProverVersionLessThanHardForkProverNumberEqual0",
			proofType:             message.ProofTypeBatch,
			forkNumbers:           map[string]int64{"istanbul": forkNumberTwo, "london": forkNumberThree},
			exceptTaskNumber:      0,
			proverForkNames:       []string{"", ""},
			exceptGetTaskErrCodes: []int{types.ErrCoordinatorEmptyProofData, types.ErrCoordinatorEmptyProofData},
			exceptGetTaskErrMsgs:  []string{"get empty prover task", "get empty prover task"},
		},
		{ // hard fork 3, prover 3 block [2-3]
			name:                  "oneTaskForkChunkProverVersionLargeOrEqualThanHardFork",
			proofType:             message.ProofTypeChunk,
			forkNumbers:           map[string]int64{"london": forkNumberThree},
			exceptTaskNumber:      1,
			proverForkNames:       []string{"london", "london"},
			exceptGetTaskErrCodes: []int{types.Success, types.ErrCoordinatorEmptyProofData},
			exceptGetTaskErrMsgs:  []string{"", "get empty prover task"},
		},
		{
			name:                  "oneTaskForkBatchProverVersionLargeOrEqualThanHardFork",
			proofType:             message.ProofTypeBatch,
			forkNumbers:           map[string]int64{"london": forkNumberThree},
			exceptTaskNumber:      1,
			proverForkNames:       []string{"london", "london"},
			exceptGetTaskErrCodes: []int{types.Success, types.ErrCoordinatorEmptyProofData},
			exceptGetTaskErrMsgs:  []string{"", "get empty prover task"},
		},
		{ // hard fork 2, prover 2 block [2-3]
			name:                  "oneTaskForkChunkProverVersionLessThanHardFork",
			proofType:             message.ProofTypeChunk,
			forkNumbers:           map[string]int64{"istanbul": forkNumberTwo, "london": forkNumberThree},
			exceptTaskNumber:      1,
			proverForkNames:       []string{"istanbul", "istanbul"},
			exceptGetTaskErrCodes: []int{types.Success, types.ErrCoordinatorEmptyProofData},
			exceptGetTaskErrMsgs:  []string{"", "get empty prover task"},
		},
		{
			name:                  "oneTaskForkBatchProverVersionLessThanHardFork",
			proofType:             message.ProofTypeBatch,
			forkNumbers:           map[string]int64{"istanbul": forkNumberTwo, "london": forkNumberThree},
			exceptTaskNumber:      1,
			proverForkNames:       []string{"istanbul", "istanbul"},
			exceptGetTaskErrCodes: []int{types.Success, types.ErrCoordinatorEmptyProofData},
			exceptGetTaskErrMsgs:  []string{"", "get empty prover task"},
		},
		{ // hard fork 2, prover 2 block [2-3]
			name:                  "twoTaskForkChunkProverVersionLargeOrEqualThanHardFork",
			proofType:             message.ProofTypeChunk,
			forkNumbers:           map[string]int64{"istanbul": forkNumberTwo},
			exceptTaskNumber:      2,
			proverForkNames:       []string{"istanbul", "istanbul"},
			exceptGetTaskErrCodes: []int{types.Success, types.Success},
			exceptGetTaskErrMsgs:  []string{"", ""},
		},
		{
			name:                  "twoTaskForkBatchProverVersionLargeOrEqualThanHardFork",
			proofType:             message.ProofTypeBatch,
			forkNumbers:           map[string]int64{"istanbul": forkNumberTwo},
			exceptTaskNumber:      2,
			proverForkNames:       []string{"istanbul", "istanbul"},
			exceptGetTaskErrCodes: []int{types.Success, types.Success},
			exceptGetTaskErrMsgs:  []string{"", ""},
		},
		{ // hard fork 4, prover 3 block [2-3]
			name:                  "twoTaskForkChunkProverVersionLessThanHardFork",
			proofType:             message.ProofTypeChunk,
			forkNumbers:           map[string]int64{"bernoulli": forkNumberFour, "istanbul": forkNumberTwo},
			exceptTaskNumber:      2,
			proverForkNames:       []string{"istanbul", "istanbul"},
			exceptGetTaskErrCodes: []int{types.Success, types.Success},
			exceptGetTaskErrMsgs:  []string{"", ""},
		},
		{ // hard fork 3, prover1:2 prover2:3 block [2-3]
			name:                  "twoTaskForkChunkProverVersionMiddleHardFork",
			proofType:             message.ProofTypeChunk,
			forkNumbers:           map[string]int64{"istanbul": forkNumberTwo, "london": forkNumberThree},
			exceptTaskNumber:      2,
			proverForkNames:       []string{"istanbul", "london"},
			exceptGetTaskErrCodes: []int{types.Success, types.Success},
			exceptGetTaskErrMsgs:  []string{"", ""},
		},
		{
			name:                  "twoTaskForkBatchProverVersionMiddleHardFork",
			proofType:             message.ProofTypeBatch,
			forkNumbers:           map[string]int64{"istanbul": forkNumberTwo, "london": forkNumberThree},
			exceptTaskNumber:      2,
			proverForkNames:       []string{"istanbul", "london"},
			exceptGetTaskErrCodes: []int{types.Success, types.Success},
			exceptGetTaskErrMsgs:  []string{"", ""},
		},
		{ // hard fork 3, prover1:2 prover2:3 block [2-3]
			name:                  "twoTaskForkChunkProverVersionMiddleHardForkProverNumberEqual0",
			proofType:             message.ProofTypeChunk,
			forkNumbers:           map[string]int64{"london": forkNumberThree},
			exceptTaskNumber:      2,
			proverForkNames:       []string{"", "london"},
			exceptGetTaskErrCodes: []int{types.Success, types.Success},
			exceptGetTaskErrMsgs:  []string{"", ""},
		},
		{
			name:                  "twoTaskForkBatchProverVersionMiddleHardForkProverNumberEqual0",
			proofType:             message.ProofTypeBatch,
			forkNumbers:           map[string]int64{"london": forkNumberThree},
			exceptTaskNumber:      2,
			proverForkNames:       []string{"", "london"},
			exceptGetTaskErrCodes: []int{types.Success, types.Success},
			exceptGetTaskErrMsgs:  []string{"", ""},
		},
		{ // hard fork 2, prover 2 block [2-3]
			name:                  "oneTaskForkChunkProverVersionLessThanHardForkProverNumberEqual0",
			proofType:             message.ProofTypeChunk,
			forkNumbers:           map[string]int64{"london": forkNumberThree},
			exceptTaskNumber:      1,
			proverForkNames:       []string{"", ""},
			exceptGetTaskErrCodes: []int{types.Success, types.ErrCoordinatorEmptyProofData},
			exceptGetTaskErrMsgs:  []string{"", "get empty prover task"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordinatorURL := randomURL()
			collector, httpHandler := setupCoordinator(t, 3, coordinatorURL, tt.forkNumbers)
			defer func() {
				collector.Stop()
				assert.NoError(t, httpHandler.Shutdown(context.Background()))
			}()

			chunkProof := &message.ChunkProof{
				StorageTrace: []byte("testStorageTrace"),
				Protocol:     []byte("testProtocol"),
				Proof:        []byte("testProof"),
				Instances:    []byte("testInstance"),
				Vk:           []byte("testVk"),
				ChunkInfo:    nil,
			}

			// the insert block number is 2 and 3
			// chunk1 batch1 contains block number 2
			// chunk2 batch2 contains block number 3
			err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
			assert.NoError(t, err)

			dbHardForkChunk1, err := chunkOrm.InsertChunk(context.Background(), hardForkChunk1)
			assert.NoError(t, err)
			err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 2, dbHardForkChunk1.Hash)
			assert.NoError(t, err)
			err = chunkOrm.UpdateProofAndProvingStatusByHash(context.Background(), dbHardForkChunk1.Hash, chunkProof, types.ProvingTaskUnassigned, 1)
			assert.NoError(t, err)
			dbHardForkBatch1, err := batchOrm.InsertBatch(context.Background(), hardForkBatch1)
			assert.NoError(t, err)
			err = chunkOrm.UpdateBatchHashInRange(context.Background(), 0, 0, dbHardForkBatch1.Hash)
			assert.NoError(t, err)
			err = batchOrm.UpdateChunkProofsStatusByBatchHash(context.Background(), dbHardForkBatch1.Hash, types.ChunkProofsStatusReady)
			assert.NoError(t, err)

			dbHardForkChunk2, err := chunkOrm.InsertChunk(context.Background(), hardForkChunk2)
			assert.NoError(t, err)
			err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 3, 100, dbHardForkChunk2.Hash)
			assert.NoError(t, err)
			err = chunkOrm.UpdateProofAndProvingStatusByHash(context.Background(), dbHardForkChunk2.Hash, chunkProof, types.ProvingTaskUnassigned, 1)
			assert.NoError(t, err)
			dbHardForkBatch2, err := batchOrm.InsertBatch(context.Background(), hardForkBatch2)
			assert.NoError(t, err)
			err = chunkOrm.UpdateBatchHashInRange(context.Background(), 1, 1, dbHardForkBatch2.Hash)
			assert.NoError(t, err)
			err = batchOrm.UpdateChunkProofsStatusByBatchHash(context.Background(), dbHardForkBatch2.Hash, types.ChunkProofsStatusReady)
			assert.NoError(t, err)

			getTaskNumber := 0
			for i := 0; i < 2; i++ {
				mockProver := newMockProver(t, fmt.Sprintf("mock_prover_%d", i), coordinatorURL, tt.proofType, version.Version)
				proverTask, errCode, errMsg := mockProver.getProverTask(t, tt.proofType, tt.proverForkNames[i])
				assert.Equal(t, tt.exceptGetTaskErrCodes[i], errCode)
				assert.Equal(t, tt.exceptGetTaskErrMsgs[i], errMsg)
				if errCode != types.Success {
					continue
				}
				getTaskNumber++
				mockProver.submitProof(t, proverTask, verifiedSuccess, types.Success)
			}
			assert.Equal(t, getTaskNumber, tt.exceptTaskNumber)
		})
	}
}

func testValidProof(t *testing.T) {
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL, map[string]int64{"istanbul": forkNumberTwo})
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), batch)
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

		provers[i] = newMockProver(t, "prover_test"+strconv.Itoa(i), coordinatorURL, proofType, version.Version)

		proofStatus := verifiedSuccess
		proverTask, errCode, errMsg := provers[i].getProverTask(t, proofType, "istanbul")
		assert.Equal(t, errCode, types.Success)
		assert.Equal(t, errMsg, "")
		assert.NotNil(t, proverTask)
		provers[i].submitProof(t, proverTask, proofStatus, types.Success)
	}

	// verify proof status
	var (
		tick     = time.Tick(1500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)

	var (
		chunkProofStatus    types.ProvingStatus
		batchProofStatus    types.ProvingStatus
		chunkActiveAttempts int16
		chunkMaxAttempts    int16
		batchActiveAttempts int16
		batchMaxAttempts    int16
	)

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

			chunkActiveAttempts, chunkMaxAttempts, err = chunkOrm.GetAttemptsByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			assert.Equal(t, 1, int(chunkMaxAttempts))
			assert.Equal(t, 0, int(chunkActiveAttempts))

			batchActiveAttempts, batchMaxAttempts, err = batchOrm.GetAttemptsByHash(context.Background(), batch.Hash)
			assert.NoError(t, err)
			assert.Equal(t, 1, int(batchMaxAttempts))
			assert.Equal(t, 0, int(batchActiveAttempts))

		case <-tickStop:
			t.Error("failed to check proof status", "chunkProofStatus", chunkProofStatus.String(), "batchProofStatus", batchProofStatus.String())
			return
		}
	}
}

func testInvalidProof(t *testing.T) {
	// Setup coordinator and ws server.
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL, map[string]int64{"istanbul": forkNumberTwo})
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), batch)
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
		provers[i] = newMockProver(t, "prover_test"+strconv.Itoa(i), coordinatorURL, proofType, version.Version)
		proverTask, errCode, errMsg := provers[i].getProverTask(t, proofType, "istanbul")
		assert.NotNil(t, proverTask)
		assert.Equal(t, errCode, types.Success)
		assert.Equal(t, errMsg, "")
		provers[i].submitProof(t, proverTask, verifiedFailed, types.ErrCoordinatorHandleZkProofFailure)
	}

	// verify proof status
	var (
		tick     = time.Tick(1500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)

	var (
		chunkProofStatus    types.ProvingStatus
		batchProofStatus    types.ProvingStatus
		chunkActiveAttempts int16
		chunkMaxAttempts    int16
		batchActiveAttempts int16
		batchMaxAttempts    int16
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
			chunkActiveAttempts, chunkMaxAttempts, err = chunkOrm.GetAttemptsByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			assert.Equal(t, 1, int(chunkMaxAttempts))
			assert.Equal(t, 0, int(chunkActiveAttempts))

			batchActiveAttempts, batchMaxAttempts, err = batchOrm.GetAttemptsByHash(context.Background(), batch.Hash)
			assert.NoError(t, err)
			assert.Equal(t, 1, int(batchMaxAttempts))
			assert.Equal(t, 0, int(batchActiveAttempts))
		case <-tickStop:
			t.Error("failed to check proof status", "chunkProofStatus", chunkProofStatus.String(), "batchProofStatus", batchProofStatus.String())
			return
		}
	}
}

func testProofGeneratedFailed(t *testing.T) {
	// Setup coordinator and ws server.
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL, map[string]int64{"istanbul": forkNumberTwo})
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), batch)
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
		provers[i] = newMockProver(t, "prover_test"+strconv.Itoa(i), coordinatorURL, proofType, version.Version)
		proverTask, errCode, errMsg := provers[i].getProverTask(t, proofType, "istanbul")
		assert.NotNil(t, proverTask)
		assert.Equal(t, errCode, types.Success)
		assert.Equal(t, errMsg, "")
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
		chunkActiveAttempts          int16
		chunkMaxAttempts             int16
		batchActiveAttempts          int16
		batchMaxAttempts             int16
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

			chunkActiveAttempts, chunkMaxAttempts, err = chunkOrm.GetAttemptsByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			assert.Equal(t, 1, int(chunkMaxAttempts))
			assert.Equal(t, 0, int(chunkActiveAttempts))

			batchActiveAttempts, batchMaxAttempts, err = batchOrm.GetAttemptsByHash(context.Background(), batch.Hash)
			assert.NoError(t, err)
			assert.Equal(t, 1, int(batchMaxAttempts))
			assert.Equal(t, 0, int(batchActiveAttempts))

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
	collector, httpHandler := setupCoordinator(t, 1, coordinatorURL, map[string]int64{"istanbul": forkNumberTwo})
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	var (
		chunkActiveAttempts int16
		chunkMaxAttempts    int16
		batchActiveAttempts int16
		batchMaxAttempts    int16
	)

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), batch)
	assert.NoError(t, err)
	err = batchOrm.UpdateChunkProofsStatusByBatchHash(context.Background(), batch.Hash, types.ChunkProofsStatusReady)
	assert.NoError(t, err)

	// create first chunk & batch mock prover, that will not send any proof.
	chunkProver1 := newMockProver(t, "prover_test"+strconv.Itoa(0), coordinatorURL, message.ProofTypeChunk, version.Version)
	proverChunkTask, errChunkCode, errChunkMsg := chunkProver1.getProverTask(t, message.ProofTypeChunk, "istanbul")
	assert.NotNil(t, proverChunkTask)
	assert.Equal(t, errChunkCode, types.Success)
	assert.Equal(t, errChunkMsg, "")

	batchProver1 := newMockProver(t, "prover_test"+strconv.Itoa(1), coordinatorURL, message.ProofTypeBatch, version.Version)
	proverBatchTask, errBatchCode, errBatchMsg := batchProver1.getProverTask(t, message.ProofTypeBatch, "istanbul")
	assert.NotNil(t, proverBatchTask)
	assert.Equal(t, errBatchCode, types.Success)
	assert.Equal(t, errBatchMsg, "")

	// verify proof status, it should be assigned, because prover didn't send any proof
	chunkProofStatus, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, chunkProofStatus, types.ProvingTaskAssigned)

	batchProofStatus, err := batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
	assert.NoError(t, err)
	assert.Equal(t, batchProofStatus, types.ProvingTaskAssigned)

	chunkActiveAttempts, chunkMaxAttempts, err = chunkOrm.GetAttemptsByHash(context.Background(), dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(chunkMaxAttempts))
	assert.Equal(t, 1, int(chunkActiveAttempts))

	batchActiveAttempts, batchMaxAttempts, err = batchOrm.GetAttemptsByHash(context.Background(), batch.Hash)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(batchMaxAttempts))
	assert.Equal(t, 1, int(batchActiveAttempts))

	// wait coordinator to reset the prover task proving status
	time.Sleep(time.Duration(conf.ProverManager.BatchCollectionTimeSec*2) * time.Second)

	// create second mock prover, that will send valid proof.
	chunkProver2 := newMockProver(t, "prover_test"+strconv.Itoa(2), coordinatorURL, message.ProofTypeChunk, version.Version)
	proverChunkTask2, chunkTask2ErrCode, chunkTask2ErrMsg := chunkProver2.getProverTask(t, message.ProofTypeChunk, "istanbul")
	assert.NotNil(t, proverChunkTask2)
	assert.Equal(t, chunkTask2ErrCode, types.Success)
	assert.Equal(t, chunkTask2ErrMsg, "")
	chunkProver2.submitProof(t, proverChunkTask2, verifiedSuccess, types.Success)

	batchProver2 := newMockProver(t, "prover_test"+strconv.Itoa(3), coordinatorURL, message.ProofTypeBatch, version.Version)
	proverBatchTask2, batchTask2ErrCode, batchTask2ErrMsg := batchProver2.getProverTask(t, message.ProofTypeBatch, "istanbul")
	assert.NotNil(t, proverBatchTask2)
	assert.Equal(t, batchTask2ErrCode, types.Success)
	assert.Equal(t, batchTask2ErrMsg, "")
	batchProver2.submitProof(t, proverBatchTask2, verifiedSuccess, types.Success)

	// verify proof status, it should be verified now, because second prover sent valid proof
	chunkProofStatus2, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, chunkProofStatus2, types.ProvingTaskVerified)

	batchProofStatus2, err := batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
	assert.NoError(t, err)
	assert.Equal(t, batchProofStatus2, types.ProvingTaskVerified)

	chunkActiveAttempts, chunkMaxAttempts, err = chunkOrm.GetAttemptsByHash(context.Background(), dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, 2, int(chunkMaxAttempts))
	assert.Equal(t, 0, int(chunkActiveAttempts))

	batchActiveAttempts, batchMaxAttempts, err = batchOrm.GetAttemptsByHash(context.Background(), batch.Hash)
	assert.NoError(t, err)
	assert.Equal(t, 2, int(batchMaxAttempts))
	assert.Equal(t, 0, int(batchActiveAttempts))
}
