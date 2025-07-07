//go:build !mock_verifier

package api

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/coordinator/internal/config"
)

type l2Syncer struct {
	l2gethClient    *ethclient.Client
	lastBlockNumber struct {
		sync.RWMutex
		data uint64
		t    time.Time
	}
}

func createL2Syncer(cfg *config.Config) (*l2Syncer, error) {

	if cfg.L2 == nil || cfg.L2.Endpoint == nil {
		return nil, fmt.Errorf("l2 endpoint is not set in config")
	} else {
		l2gethClient, err := ethclient.Dial(cfg.L2.Endpoint.Url)
		if err != nil {
			return nil, fmt.Errorf("dial l2geth endpoint fail, err: %s", err)
		}
		return &l2Syncer{
			l2gethClient: l2gethClient,
		}, nil
	}
}

// getLatestBlockNumber gets the latest block number, using cache if available and not expired
func (syncer *l2Syncer) getLatestBlockNumber(ctx *gin.Context) (uint64, error) {
	// First check if we have a cached value that's still valid
	syncer.lastBlockNumber.RLock()
	if !syncer.lastBlockNumber.t.IsZero() && time.Since(syncer.lastBlockNumber.t) < time.Second*10 {
		blockNumber := syncer.lastBlockNumber.data
		syncer.lastBlockNumber.RUnlock()
		return blockNumber, nil
	}
	syncer.lastBlockNumber.RUnlock()

	// If not cached or expired, fetch from the client
	if syncer.l2gethClient == nil {
		return 0, errors.New("L2 geth client not initialized")
	}

	blockNumber, err := syncer.l2gethClient.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block number: %w", err)
	}

	// Update the cache
	syncer.lastBlockNumber.Lock()
	syncer.lastBlockNumber.data = blockNumber
	syncer.lastBlockNumber.t = time.Now()
	syncer.lastBlockNumber.Unlock()

	log.Debug("updated block height reference", "height", blockNumber)
	return blockNumber, nil
}
