package l1

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database/migrate"

	"scroll-tech/database"
)

// testCreateNewRelayer test create new relayer instance and stop
func testCreateNewL1Relayer(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig, cfg.RedisConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	client, err := ethclient.Dial(l1gethImg.Endpoint())
	assert.NoError(t, err)

	relayer, err := NewLayer1Relayer(context.Background(), client, 1, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	relayer.Start()
}
