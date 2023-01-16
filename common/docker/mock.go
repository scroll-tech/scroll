package docker

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/utils"
)

var (
	l1StartPort = 10000
	l2StartPort = 20000
	dbStartPort = 30000
	rsStartPort = 40000
)

// NewTestL1Docker starts and returns l1geth docker
func NewTestL1Docker(t *testing.T) ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	imgL1geth := NewImgGeth(t, "scroll_l1geth", "", "", 0, l1StartPort+int(id.Int64()))
	assert.NoError(t, imgL1geth.Start())

	// try 3 times to get chainID until is ok.
	utils.TryTimes(3, func() bool {
		client, _ := ethclient.Dial(imgL1geth.Endpoint())
		if client != nil {
			if _, err := client.ChainID(context.Background()); err == nil {
				return true
			}
		}
		return false
	})

	return imgL1geth
}

// NewTestL2Docker starts and returns l2geth docker
func NewTestL2Docker(t *testing.T) ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	imgL2geth := NewImgGeth(t, "scroll_l2geth", "", "", 0, l2StartPort+int(id.Int64()))
	assert.NoError(t, imgL2geth.Start())

	// try 3 times to get chainID until is ok.
	utils.TryTimes(3, func() bool {
		client, _ := ethclient.Dial(imgL2geth.Endpoint())
		if client != nil {
			if _, err := client.ChainID(context.Background()); err == nil {
				return true
			}
		}
		return false
	})

	return imgL2geth
}

// NewTestDBDocker starts and returns database docker
func NewTestDBDocker(t *testing.T, driverName string) ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	imgDB := NewImgDB(t, driverName, "123456", "test_db", dbStartPort+int(id.Int64()))
	assert.NoError(t, imgDB.Start())

	// try 5 times until the db is ready.
	utils.TryTimes(5, func() bool {
		db, _ := sqlx.Open(driverName, imgDB.Endpoint())
		if db != nil {
			return db.Ping() == nil
		}
		return false
	})

	return imgDB
}

func NewTestRedisDocker(t *testing.T) ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	imgRedis := NewImgRedis(t, "redis", rsStartPort+int(id.Int64()))
	assert.NoError(t, imgRedis.Start())

	op, err := redis.ParseURL(imgRedis.Endpoint())
	assert.NoError(t, err)
	if t.Failed() {
		return nil
	}

	rdb := redis.NewClient(op)
	utils.TryTimes(3, func() bool {
		err = rdb.Ping(context.Background()).Err()
		return err == nil
	})

	return imgRedis
}
