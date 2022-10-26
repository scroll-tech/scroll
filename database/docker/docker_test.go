package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

func TestDB(t *testing.T) {
	img := NewImgDB(t, "postgres", "123456", "test", 5433)
	assert.NoError(t, img.Start())
	defer img.Stop()

	factory, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        img.Endpoint(),
	})
	assert.NoError(t, err)
	db := factory.GetDB()

	version := int64(0)
	assert.NoError(t, migrate.Rollback(db.DB, &version))

	assert.NoError(t, migrate.Migrate(db.DB))

	vs, err := migrate.Current(db.DB)
	assert.NoError(t, err)

	t.Logf("current version:%d", vs)
}
