package service

import (
	"context"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/common/types"
	"scroll-tech/database/migrate"
	"scroll-tech/miner-api/internal/config"
	"scroll-tech/miner-api/internal/orm"
	"testing"
)

var (
	proverPubkey = "key"

	task1 = orm.ProverTask{
		TaskID:          "1",
		ProverPublicKey: proverPubkey,
		ProverName:      "prover-0",
		ProvingStatus:   int16(types.RollerAssigned),
		Reward:          decimal.NewFromInt(10),
	}

	task2 = orm.ProverTask{
		TaskID:          "2",
		ProverPublicKey: proverPubkey,
		ProverName:      "prover-1",
		ProvingStatus:   int16(types.RollerAssigned),
		Reward:          decimal.NewFromInt(12),
	}

	service *ProverTaskService
)

func insertSomeProverTasks(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	ptdb := orm.NewProverTask(db)
	err = ptdb.SetProverTask(context.Background(), &task1)
	assert.NoError(t, err)

	err = ptdb.SetProverTask(context.Background(), &task2)
	assert.NoError(t, err)
}

func TestProverTaskService(t *testing.T) {
	// start database image
	base := docker.NewDockerApp()
	defer base.Free()
	cfg, err := config.NewConfig("../config.json")
	assert.NoError(t, err)
	cfg.DBConfig.DSN = base.DBImg.Endpoint()
	base.RunDBImage(t)

	db, err := database.InitDB(cfg.DBConfig)
	assert.NoError(t, err)

	// insert some tasks
	insertSomeProverTasks(t, db)

	ptdb := orm.NewProverTask(db)
	service = NewProverTaskService(ptdb)

	t.Run("testGetTasksByProver", testGetTasksByProver)
	t.Run("testGetTotalRewards", testGetTotalRewards)
	t.Run("testGetTask", testGetTask)
}

func testGetTasksByProver(t *testing.T) {
	tasks, err := service.GetTasksByProver(proverPubkey)
	assert.NoError(t, err)
	assert.Equal(t, task2, *tasks[0])
	assert.Equal(t, task1, *tasks[1])
}

func testGetTotalRewards(t *testing.T) {
	rewards, err := service.GetTotalRewards(proverPubkey)
	assert.NoError(t, err)
	assert.Equal(t, decimal.NewFromInt(22), rewards)
}

func testGetTask(t *testing.T) {
	task, err := service.GetTask("2")
	assert.NoError(t, err)
	assert.Equal(t, task2, *task)
}
