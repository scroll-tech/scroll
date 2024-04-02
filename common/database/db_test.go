package database_test

import (
	"scroll-tech/common/database"
	"scroll-tech/common/testcontainers"
	"scroll-tech/common/version"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB(t *testing.T) {
	var err error
	version.Version = "v4.1.98-aaa-bbb-ccc"

	testApps := testcontainers.NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())

	db, err := testApps.GetGormDBClient()
	assert.NoError(t, err)

	sqlDB, err := database.Ping(db)
	assert.NoError(t, err)
	assert.NotNil(t, sqlDB)

	assert.NoError(t, database.CloseDB(db))
}
