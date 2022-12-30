package migrate

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
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
	vp, err = viper.NewViper("../config.json", true)
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

func TestMigration(t *testing.T) {
	defer func() {
		if dbImg != nil {
			assert.NoError(t, dbImg.Stop())
		}
	}()
	if err := initEnv(t); err != nil {
		t.Fatal(err)
	}

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
