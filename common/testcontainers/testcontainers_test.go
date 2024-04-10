package testcontainers

import (
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestNewTestcontainerApps tests NewTestcontainerApps
func TestNewTestcontainerApps(t *testing.T) {
	var (
		err          error
		endpoint     string
		gormDBclient *gorm.DB
		ethclient    *ethclient.Client
	)

	// test start testcontainers
	testApps := NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())
	endpoint, err = testApps.GetDBEndPoint()
	assert.NoError(t, err)
	assert.NotEmpty(t, endpoint)
	gormDBclient, err = testApps.GetGormDBClient()
	assert.NoError(t, err)
	assert.NotNil(t, gormDBclient)

	assert.NoError(t, testApps.StartL1GethContainer())
	endpoint, err = testApps.GetL1GethEndPoint()
	assert.NoError(t, err)
	assert.NotEmpty(t, endpoint)
	ethclient, err = testApps.GetL1GethClient()
	assert.NoError(t, err)
	assert.NotNil(t, ethclient)

	assert.NoError(t, testApps.StartL2GethContainer())
	endpoint, err = testApps.GetL2GethEndPoint()
	assert.NoError(t, err)
	assert.NotEmpty(t, endpoint)
	ethclient, err = testApps.GetL2GethClient()
	assert.NoError(t, err)
	assert.NotNil(t, ethclient)

	// test free testcontainers
	testApps.Free()
	endpoint, err = testApps.GetDBEndPoint()
	assert.EqualError(t, err, "postgres is not running")
	assert.Empty(t, endpoint)

	endpoint, err = testApps.GetL1GethEndPoint()
	assert.EqualError(t, err, "l1 geth is not running")
	assert.Empty(t, endpoint)

	endpoint, err = testApps.GetL2GethEndPoint()
	assert.EqualError(t, err, "l2 geth is not running")
	assert.Empty(t, endpoint)
}
