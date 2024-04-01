package migrate

import (
	"context"
	"scroll-tech/common/testcontainers"
	tc "scroll-tech/common/testcontainers"
	"scroll-tech/database"
	db "scroll-tech/database"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var (
	testApps *testcontainers.TestcontainerApps
	pgDB     *sqlx.DB
)

func setupEnv(t *testing.T) {
	// Start db container.
	testApps = tc.NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())
	// Create db orm handler.
	dsn, err := testApps.GetDBEndPoint()
	assert.NoError(t, err)
	factory, err := db.NewOrmFactory(&database.DBConfig{
		DSN:        dsn,
		DriverName: "postgres",
		MaxOpenNum: 200,
		MaxIdleNum: 20,
	})
	assert.NoError(t, err)
	pgDB = factory.GetDB()
}

func TestMain(m *testing.M) {
	defer func() {
		if testApps != nil {
			testApps.Free(context.Background())
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
	cur, err := Current(pgDB.DB)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), cur)
}

func testStatus(t *testing.T) {
	status := Status(pgDB.DB)
	assert.NoError(t, status)
}

func testResetDB(t *testing.T) {
	assert.NoError(t, ResetDB(pgDB.DB))
	cur, err := Current(pgDB.DB)
	assert.NoError(t, err)
	// total number of tables.
	assert.Equal(t, int64(16), cur)
}

func testMigrate(t *testing.T) {
	assert.NoError(t, Migrate(pgDB.DB))
	cur, err := Current(pgDB.DB)
	assert.NoError(t, err)
	assert.Equal(t, int64(16), cur)
}

func testRollback(t *testing.T) {
	version, err := Current(pgDB.DB)
	assert.NoError(t, err)
	assert.Equal(t, int64(16), version)

	assert.NoError(t, Rollback(pgDB.DB, nil))

	cur, err := Current(pgDB.DB)
	assert.NoError(t, err)
	assert.Equal(t, version, cur+1)

	targetVersion := int64(0)
	assert.NoError(t, Rollback(pgDB.DB, &targetVersion))

	cur, err = Current(pgDB.DB)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), cur)
}
