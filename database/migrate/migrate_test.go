package migrate

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"

	"scroll-tech/database"
	"scroll-tech/database/cache"
)

var (
	pgDB     *sqlx.DB
	dbImg    docker.ImgInstance
	redisImg docker.ImgInstance
)

func initEnv(t *testing.T) error {
	// Start db container.
	dbImg = docker.NewTestDBDocker(t, "postgres")
	redisImg = docker.NewTestRedisDocker(t)

	// Create db orm handler.
	factory, err := database.NewOrmFactory(&database.DBConfig{
		Persistence: &database.PersistenceConfig{
			DriverName: "postgres",
			DSN:        dbImg.Endpoint(),
		},
		Redis: &cache.RedisConfig{
			Expirations: map[string]int64{"trace": 30},
			URL:         redisImg.Endpoint(),
		},
	})
	if err != nil {
		return err
	}
	pgDB = factory.GetDB()
	return nil
}

func TestMigrate(t *testing.T) {
	if err := initEnv(t); err != nil {
		t.Fatal(err)
	}

	t.Run("testCurrent", testCurrent)
	t.Run("testStatus", testStatus)
	t.Run("testResetDB", testResetDB)
	t.Run("testMigrate", testMigrate)
	t.Run("testRollback", testRollback)

	t.Cleanup(func() {
		if dbImg != nil {
			assert.NoError(t, dbImg.Stop())
		}
		if redisImg != nil {
			assert.NoError(t, redisImg.Stop())
		}
	})
}

func testCurrent(t *testing.T) {
	cur, err := Current(pgDB.DB)
	assert.NoError(t, err)
	assert.Equal(t, 0, int(cur))
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
	assert.Equal(t, 6, int(cur))
}

func testMigrate(t *testing.T) {
	assert.NoError(t, Migrate(pgDB.DB))
	cur, err := Current(pgDB.DB)
	assert.NoError(t, err)
	assert.Equal(t, true, cur > 0)
}

func testRollback(t *testing.T) {
	version, err := Current(pgDB.DB)
	assert.NoError(t, err)
	assert.Equal(t, true, version > 0)

	assert.NoError(t, Rollback(pgDB.DB, nil))

	cur, err := Current(pgDB.DB)
	assert.NoError(t, err)
	assert.Equal(t, true, cur+1 == version)
}
