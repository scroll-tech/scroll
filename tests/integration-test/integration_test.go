package integration_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/integration-test/orm"
	rapp "scroll-tech/prover/cmd/app"

	"scroll-tech/database/migrate"

	capp "scroll-tech/coordinator/cmd/app"

	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	bcmd "scroll-tech/bridge/cmd"
)

var (
	base           *docker.App
	bridgeApp      *bcmd.MockApp
	coordinatorApp *capp.CoordinatorApp
	chunkProverApp *rapp.ProverApp
	batchProverApp *rapp.ProverApp
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()
	bridgeApp = bcmd.NewBridgeApp(base, "../../bridge/conf/config.json")
	coordinatorApp = capp.NewCoordinatorApp(base, "../../coordinator/conf/config.json")
	chunkProverApp = rapp.NewProverApp(base, "../../prover/config.json", coordinatorApp.HTTPEndpoint(), message.ProofTypeChunk)
	batchProverApp = rapp.NewProverApp(base, "../../prover/config.json", coordinatorApp.HTTPEndpoint(), message.ProofTypeBatch)
	m.Run()
	bridgeApp.Free()
	coordinatorApp.Free()
	chunkProverApp.Free()
	batchProverApp.Free()
	base.Free()
}

func TestCoordinatorProverInteractionWithoutData(t *testing.T) {
	// Start postgres docker containers.
	base.RunL2Geth(t)
	base.RunDBImage(t)
	// Reset db.
	assert.NoError(t, migrate.ResetDB(base.DBClient(t)))

	// Run coordinator app.
	coordinatorApp.RunApp(t)

	// Run prover app.
	chunkProverApp.RunApp(t) // chunk prover login.
	batchProverApp.RunApp(t) // batch prover login.

	chunkProverApp.ExpectWithTimeout(t, true, 60*time.Second, "get empty prover task") // get prover task without data.
	batchProverApp.ExpectWithTimeout(t, true, 60*time.Second, "get empty prover task") // get prover task without data.

	// Free apps.
	chunkProverApp.WaitExit()
	batchProverApp.WaitExit()
	coordinatorApp.WaitExit()
}

func TestCoordinatorProverInteractionWithData(t *testing.T) {
	// Start postgres docker containers.
	base.RunL2Geth(t)
	base.RunDBImage(t)

	// Init data
	dbCfg := &database.Config{
		DSN:        base.DBConfig.DSN,
		DriverName: base.DBConfig.DriverName,
		MaxOpenNum: base.DBConfig.MaxOpenNum,
		MaxIdleNum: base.DBConfig.MaxIdleNum,
	}

	db, err := database.InitDB(dbCfg)
	assert.NoError(t, err)

	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	batchOrm := orm.NewBatch(db)
	chunkOrm := orm.NewChunk(db)
	l2BlockOrm := orm.NewL2Block(db)

	wrappedBlock := &types.WrappedBlock{
		Header: &gethTypes.Header{
			Number:     big.NewInt(1),
			ParentHash: common.Hash{},
			Difficulty: big.NewInt(0),
			BaseFee:    big.NewInt(0),
		},
		Transactions:   nil,
		WithdrawRoot:   common.Hash{},
		RowConsumption: &gethTypes.RowConsumption{},
	}
	chunk := &types.Chunk{Blocks: []*types.WrappedBlock{wrappedBlock}}

	err = l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), 0, 0, dbChunk.Hash, dbChunk.Hash, []*types.Chunk{chunk})
	assert.NoError(t, err)
	err = chunkOrm.UpdateBatchHashInRange(context.Background(), 0, 0, batch.Hash)
	assert.NoError(t, err)

	// Run coordinator app.
	coordinatorApp.RunApp(t)

	// Run prover app.
	chunkProverApp.RunApp(t) // chunk prover login.
	batchProverApp.RunApp(t) // batch prover login.

	time.Sleep(60 * time.Second) // TODO(colinlyguo): replace time.Sleep(60 * time.Second) with expected results.

	// Free apps.
	chunkProverApp.WaitExit()
	batchProverApp.WaitExit()
	coordinatorApp.WaitExit()
}
