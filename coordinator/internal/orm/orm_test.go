package orm

import (
	"context"
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/common/types"

	"scroll-tech/database/migrate"
)

var (
	base *docker.App

	db            *gorm.DB
	proverTaskOrm *ProverTask
)

func TestMain(m *testing.M) {
	t := &testing.T{}
	setupEnv(t)
	defer tearDownEnv(t)
	m.Run()
}

func setupEnv(t *testing.T) {
	base = docker.NewDockerApp()
	base.RunDBImage(t)
	var err error
	db, err = database.InitDB(
		&database.Config{
			DSN:        base.DBConfig.DSN,
			DriverName: base.DBConfig.DriverName,
			MaxOpenNum: base.DBConfig.MaxOpenNum,
			MaxIdleNum: base.DBConfig.MaxIdleNum,
		},
	)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	proverTaskOrm = NewProverTask(db)
}

func tearDownEnv(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	sqlDB.Close()
	base.Free()
}

func TestProverTaskOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	reward := big.NewInt(0)
	reward.SetString("18446744073709551616", 10) // 1 << 64, uint64 maximum 1<<64 -1

	proverTask := ProverTask{
		TaskID:          "test-hash",
		ProverName:      "roller-0",
		ProverPublicKey: "0",
		ProvingStatus:   int16(types.RollerAssigned),
		Reward:          decimal.NewFromBigInt(reward, 0),
	}

	err = proverTaskOrm.SetProverTask(context.Background(), &proverTask)
	assert.NoError(t, err)
	proverTasks, err := proverTaskOrm.GetProverTasksByHashes(context.Background(), []string{"test-hash"})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(proverTasks))
	assert.Equal(t, proverTask.ProverName, proverTasks[0].ProverName)

	// // test decimal reward, get reward
	resultReward := proverTask.Reward.BigInt()
	assert.Equal(t, resultReward, reward)
	assert.Equal(t, resultReward.String(), "18446744073709551616")

	proverTask.ProvingStatus = int16(types.RollerProofValid)
	err = proverTaskOrm.SetProverTask(context.Background(), &proverTask)
	assert.NoError(t, err)
	proverTasks, err = proverTaskOrm.GetProverTasksByHashes(context.Background(), []string{"test-hash"})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(proverTasks))
	assert.Equal(t, proverTask.ProvingStatus, proverTasks[0].ProvingStatus)
}
