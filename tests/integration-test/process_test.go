package integration_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"

	"scroll-tech/common/docker"
)

func testDatabaseOperation(t *testing.T) {
	cmd := docker.NewCmd(t)

	cmd.OpenLog(true)
	// Wait reset result
	cmd.ExpectWithTimeout(false, time.Second*3, "successful to reset")
	cmd.Run("db_cli-test", "--log.debug", "reset", "--config", "../../database/config.json", "--db.dsn", dbImg.Endpoint())
	cmd.WaitExit()

	// Wait migrate result
	cmd.ExpectWithTimeout(false, time.Second*3, "current version: 5")
	cmd.Run("db_cli-test", "--log.debug", "migrate", "--config", "../../database/config.json", "--db.dsn", dbImg.Endpoint())
	cmd.WaitExit()
}

func testBridgeOperation(t *testing.T) {
	cmd := docker.NewCmd(t)

	db, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        dbImg.Endpoint(),
	})
	assert.NoError(t, err)
	// Reset db.
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))

	// Start bridge service.
	cmd.Run("bridge-test", "--log.debug",
		"--config", "../../bridge/config.json",
		"--l1.endpoint", l1gethImg.Endpoint(),
		"--l2.endpoint", l2gethImg.Endpoint(),
		"--db.dsn", dbImg.Endpoint())

	//cmd.ExpectWithTimeout(false, time.Second*20, "")
	<-time.After(time.Second * 20)
	cmd.WaitExit()
}
