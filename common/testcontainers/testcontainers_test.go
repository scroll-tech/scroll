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

	testApps := NewTestcontainerApps()

	// test start testcontainers
	assert.NoError(t, testApps.StartPostgresContainer())
	endpoint, err = testApps.GetDBEndPoint()
	assert.NoError(t, err)
	assert.NotEmpty(t, endpoint)
	gormDBclient, err = testApps.GetGormDBClient()
	assert.NoError(t, err)
	assert.NotNil(t, gormDBclient)

	assert.NoError(t, testApps.StartL2GethContainer())
	endpoint, err = testApps.GetL2GethEndPoint()
	assert.NoError(t, err)
	assert.NotEmpty(t, endpoint)
	ethclient, err = testApps.GetL2GethClient()
	assert.NoError(t, err)
	assert.NotNil(t, ethclient)

	assert.NoError(t, testApps.StartPoSL1Container())
	endpoint, err = testApps.GetPoSL1EndPoint()
	assert.NoError(t, err)
	assert.NotEmpty(t, endpoint)
	ethclient, err = testApps.GetPoSL1Client()
	assert.NoError(t, err)
	assert.NotNil(t, ethclient)

	assert.NoError(t, testApps.StartWeb3SignerContainer(1))
	endpoint, err = testApps.GetWeb3SignerEndpoint()
	assert.NoError(t, err)
	assert.NotEmpty(t, endpoint)

	// test free testcontainers
	testApps.Free()
	endpoint, err = testApps.GetDBEndPoint()
	assert.EqualError(t, err, "postgres is not running")
	assert.Empty(t, endpoint)

	endpoint, err = testApps.GetL2GethEndPoint()
	assert.EqualError(t, err, "l2 geth is not running")
	assert.Empty(t, endpoint)

	endpoint, err = testApps.GetPoSL1EndPoint()
	assert.EqualError(t, err, "PoS L1 container is not running")
	assert.Empty(t, endpoint)

	endpoint, err = testApps.GetWeb3SignerEndpoint()
	assert.EqualError(t, err, "web3signer is not running")
	assert.Empty(t, endpoint)
}
