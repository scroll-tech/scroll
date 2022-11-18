package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	_ "scroll-tech/bridge/cmd/app"
	"scroll-tech/common/docker"
	_ "scroll-tech/coordinator/cmd/app"
	"scroll-tech/database"
	_ "scroll-tech/database/cmd/app"
	_ "scroll-tech/roller/cmd/app"
)

var (
	l1gethImg docker.ImgInstance
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance
	l2db      database.OrmFactory
)

func setupEnv(t *testing.T) {
	l1gethImg = docker.NewTestL1Docker(t)
	l2gethImg = docker.NewTestL2Docker(t)
	dbImg = docker.NewTestDBDocker(t, "postgres")

	// Create db handler and reset db.
	var err error
	l2db, err = database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        dbImg.Endpoint(),
	})
	assert.NoError(t, err)
}

func free(t *testing.T) {
	assert.NoError(t, l2db.Close())
	assert.NoError(t, l1gethImg.Stop())
	assert.NoError(t, l2gethImg.Stop())
	assert.NoError(t, dbImg.Stop())
}

func TestVersion(t *testing.T) {
	setupEnv(t)

	// test cmd --version
	t.Run("testBridgeCmd", testBridgeCmd)
	t.Run("testCoordinatorCmd", testCoordinatorCmd)
	t.Run("testDatabaseCmd", testDatabaseCmd)
	t.Run("testRollerCmd", testRollerCmd)

	// test db_cli
	t.Run("testDatabaseOperation", testDatabaseOperation)
	// test bridge service
	t.Run("testBridgeOperation", testBridgeOperation)

	t.Cleanup(func() {
		free(t)
	})
}
