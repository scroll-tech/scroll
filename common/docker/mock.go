package docker

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

var (
	l1StartPort = 11000
	l2StartPort = 12000
	dbStartPort = 13000
)

// NewTestL1Docker starts and returns l1geth docker
func NewTestL1Docker(t *testing.T) ImgInstance {
	imgL1geth := NewImgGeth(t, "scroll_l1geth", "", "", 0, l1StartPort+rand.Intn(2000))
	assert.NoError(t, imgL1geth.Start())
	return imgL1geth
}

// NewTestL2Docker starts and returns l2geth docker
func NewTestL2Docker(t *testing.T) ImgInstance {
	imgL2geth := NewImgGeth(t, "scroll_l2geth", "", "", 0, l2StartPort+rand.Intn(2000))
	assert.NoError(t, imgL2geth.Start())
	return imgL2geth
}

// NewTestDBDocker starts and returns database docker
func NewTestDBDocker(t *testing.T, driverName string) ImgInstance {
	imgDB := NewImgDB(t, driverName, "123456", "test_db", dbStartPort+rand.Intn(2000))
	assert.NoError(t, imgDB.Start())
	return imgDB
}
