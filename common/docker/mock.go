package docker

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	l1StartPort = 11000
	l2StartPort = 12000
	dbStartPort = 13000
)

// NewTestL1Docker starts and returns l1geth docker
func NewTestL1Docker(t *testing.T) ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	imgL1geth := NewImgGeth(t, "scroll_l1geth", "", "", 0, l1StartPort+int(id.Int64()))
	assert.NoError(t, imgL1geth.Start())
	return imgL1geth
}

// NewTestL2Docker starts and returns l2geth docker
func NewTestL2Docker(t *testing.T) ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	imgL2geth := NewImgGeth(t, "scroll_l2geth", "", "", 0, l2StartPort+int(id.Int64()))
	assert.NoError(t, imgL2geth.Start())
	return imgL2geth
}

// NewTestDBDocker starts and returns database docker
func NewTestDBDocker(t *testing.T, driverName string) ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	imgDB := NewImgDB(t, driverName, "123456", "test_db", dbStartPort+int(id.Int64()))
	assert.NoError(t, imgDB.Start())
	return imgDB
}
