package migrate

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"
)

var (
	base *docker.App
	pgDB *sql.DB
)

func initEnv(t *testing.T) {
	// Start db container.
	base.RunDBImage(t)
	pgDB = base.DBClient(t)
}

func TestMigrate(t *testing.T) {
	base = docker.NewDockerApp()
	initEnv(t)

	t.Run("testCurrent", testCurrent)
	t.Run("testStatus", testStatus)
	t.Run("testResetDB", testResetDB)
	t.Run("testMigrate", testMigrate)
	t.Run("testRollback", testRollback)

	t.Cleanup(func() {
		base.Free()
	})
}

func testCurrent(t *testing.T) {
	cur, err := Current(pgDB)
	assert.NoError(t, err)
	assert.Equal(t, 0, int(cur))
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
	assert.Equal(t, 6, int(cur))
}

func testMigrate(t *testing.T) {
	assert.NoError(t, Migrate(pgDB))
	cur, err := Current(pgDB)
	assert.NoError(t, err)
	assert.Equal(t, true, cur > 0)
}

func testRollback(t *testing.T) {
	version, err := Current(pgDB)
	assert.NoError(t, err)
	assert.Equal(t, true, version > 0)

	assert.NoError(t, Rollback(pgDB, nil))

	cur, err := Current(pgDB)
	assert.NoError(t, err)
	assert.Equal(t, true, cur+1 == version)
}
