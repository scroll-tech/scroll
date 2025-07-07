//go:build mock_verifier

package api

import (
	"scroll-tech/coordinator/internal/config"

	"github.com/gin-gonic/gin"
)

type l2Syncer struct{}

func createL2Syncer(_ *config.Config) (*l2Syncer, error) {
	return &l2Syncer{}, nil
}

// getLatestBlockNumber gets the latest block number, using cache if available and not expired
func (syncer *l2Syncer) getLatestBlockNumber(_ *gin.Context) (uint64, error) {
	return 99999994, nil
}
