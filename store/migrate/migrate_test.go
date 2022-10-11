package migrate

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"

	"scroll-tech/internal/docker"

	"scroll-tech/store"
	db_config "scroll-tech/store/config"
)

var (
	pgDB *sqlx.DB
	img  docker.ImgInstance
)

func initEnv(t *testing.T) {
	img = docker.NewImgDB(t, "postgres", "123456", "test_1", 5434)
	assert.NoError(t, img.Start())

	var err error
	pgDB, err = store.NewConnection(&db_config.DBConfig{
		DriverName: db_config.GetEnvWithDefault("TEST_DB_DRIVER", "postgres"),
		DSN:        img.Endpoint(),
	})
	assert.NoError(t, err)
}

func TestMegration(t *testing.T) {
	initEnv(t)
	defer img.Stop()

	err := Migrate(pgDB.DB)
	assert.NoError(t, err)

	db := pgDB.DB
	version0, err := goose.GetDBVersion(db)
	assert.NoError(t, err)
	t.Log("current version is ", version0)

	// rollback one version
	assert.NoError(t, Rollback(db, nil))

	version1, err := Current(db)
	assert.NoError(t, err)

	// check version expect less than 1
	assert.Equal(t, version0-1, version1)
}
