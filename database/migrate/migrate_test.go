package migrate

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"

	"scroll-tech/common/viper"

	"scroll-tech/database"
)

var (
	pgDB  *sqlx.DB
	dbImg docker.ImgInstance

	vp *viper.Viper
)

func initEnv(t *testing.T) error {
	// Start db container.
	dbImg = docker.NewTestDBDocker(t, "postgres")

	var err error
	vp, err = viper.NewViper("../config.json", "")
	assert.NoError(t, err)
	vp.Set("dsn", dbImg.Endpoint())

	// Create db orm handler.
	factory, err := database.NewOrmFactory(vp)
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
	assert.Equal(t, 5, int(cur))
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
