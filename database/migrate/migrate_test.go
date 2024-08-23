package migrate

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/testcontainers"
)

var (
	testApps *testcontainers.TestcontainerApps
	pgDB     *sql.DB
)

func setupEnv(t *testing.T) {
	// Start db container.
	testApps = testcontainers.NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())
	gormClient, err := testApps.GetGormDBClient()
	assert.NoError(t, err)
	pgDB, err = gormClient.DB()
	assert.NoError(t, err)
}

func TestMain(m *testing.M) {
	defer func() {
		if testApps != nil {
			testApps.Free()
		}
	}()
	m.Run()
}

func TestMigrate(t *testing.T) {
	setupEnv(t)
	t.Run("testCurrent", testCurrent)
	t.Run("testStatus", testStatus)
	t.Run("testResetDB", testResetDB)
	t.Run("testMigrate", testMigrate)
	t.Run("testRollback", testRollback)
}

func testCurrent(t *testing.T) {
	cur, err := Current(pgDB)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), cur)
}

func testStatus(t *testing.T) {
	status := Status(pgDB)
	assert.NoError(t, status)
}

func testResetDB(t *testing.T) {
	assert.NoError(t, ResetDB(pgDB))
	cur, err := Current(pgDB)
	assert.NoError(t, err)
	// total number of tables.
	assert.Equal(t, int64(24), cur)
}

func testMigrate(t *testing.T) {
	assert.NoError(t, Migrate(pgDB))
	cur, err := Current(pgDB)
	assert.NoError(t, err)
	assert.Equal(t, int64(24), cur)
}

func testRollback(t *testing.T) {
	version, err := Current(pgDB)
	assert.NoError(t, err)
	assert.Equal(t, int64(24), version)

	assert.NoError(t, Rollback(pgDB, nil))

	cur, err := Current(pgDB)
	assert.NoError(t, err)
	assert.Equal(t, version, cur+1)

	targetVersion := int64(0)
	assert.NoError(t, Rollback(pgDB, &targetVersion))

	cur, err = Current(pgDB)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), cur)
}
