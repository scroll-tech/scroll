package l1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/database/migrate"

	"scroll-tech/database"
)

// TestCreateNewL1Relayer test create new relayer instance and stop
func TestCreateNewL1Relayer(t *testing.T) {
	// Start docker containers.
	base.RunImages(t)
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	relayer, err := NewLayer1Relayer(context.Background(), db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	relayer.Start()
}
