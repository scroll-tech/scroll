package orm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/utils"

	"scroll-tech/common/docker"
	"scroll-tech/common/types"

	"scroll-tech/database/migrate"
)

var (
	base *docker.App

	db             *gorm.DB
	sessionInfoOrm *SessionInfo
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
	db, err = utils.InitDB(
		&config.DBConfig{
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

	sessionInfoOrm = NewSessionInfo(db)
}

func tearDownEnv(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	sqlDB.Close()
	base.Free()
}

func TestSessionInfoOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	sessionInfo := SessionInfo{
		TaskID:          "test-hash",
		RollerName:      "roller-0",
		RollerPublicKey: "0",
		ProvingStatus:   int16(types.RollerAssigned),
	}

	err = sessionInfoOrm.SetSessionInfo(context.Background(), &sessionInfo)
	assert.NoError(t, err)
	sessionInfos, err := sessionInfoOrm.GetSessionInfosByHashes(context.Background(), []string{"test-hash"})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(sessionInfos))
	assert.Equal(t, sessionInfo.RollerName, sessionInfos[0].RollerName)

	sessionInfo.ProvingStatus = int16(types.RollerProofValid)
	err = sessionInfoOrm.SetSessionInfo(context.Background(), &sessionInfo)
	assert.NoError(t, err)
	sessionInfos, err = sessionInfoOrm.GetSessionInfosByHashes(context.Background(), []string{"test-hash"})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(sessionInfos))
	assert.Equal(t, sessionInfo.ProvingStatus, sessionInfos[0].ProvingStatus)
}
