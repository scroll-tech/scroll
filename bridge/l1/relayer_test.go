package l1_test

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database/migrate"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l1"

	"scroll-tech/database"

	"scroll-tech/common/docker"
	"scroll-tech/common/viper"
)

// TestCreateNewRelayer test create new relayer instance and stop
func TestCreateNewL1Relayer(t *testing.T) {
	assert.NoError(t, config.NewConfig("../config.json"))
	l1docker := docker.NewTestL1Docker(t)
	defer l1docker.Stop()
	viper.Set("l2_config.relayer_config.sender_config.endpoint", l1docker.Endpoint())
	viper.Set("l1_config.endpoint", l1docker.Endpoint())

	client, err := ethclient.Dial(l1docker.Endpoint())
	assert.NoError(t, err)

	driverName := viper.GetViper().GetString("db_config.driver_name")
	dbImg := docker.NewTestDBDocker(t, driverName)
	defer dbImg.Stop()
	viper.Set("db_config.driver_name", driverName)
	viper.Set("db_config.dsn", dbImg.Endpoint())

	// Create db handler and reset db.
	db, err := database.NewOrmFactory(viper.Sub("db_config"))
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	relayer, err := l1.NewLayer1Relayer(context.Background(), client, db, viper.Sub("l2_config.relayer_config"))
	assert.NoError(t, err)
	defer relayer.Stop()

	relayer.Start()
}
