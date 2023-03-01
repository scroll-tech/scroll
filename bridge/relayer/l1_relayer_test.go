package relayer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/relayer"
	"scroll-tech/database/migrate"

	"scroll-tech/database"
)

// testCreateNewRelayer test create new relayer instance and stop
func testCreateNewL1Relayer(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	_, err = relayer.NewLayer1Relayer(context.Background(), db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
}
