package logic

import (
	"context"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
)

// L2ReorgSafeDepth represents the number of block confirmations considered safe against L2 chain reorganizations.
// Reorganizations at this depth under normal cases are extremely unlikely.
const L2ReorgSafeDepth = 256

// L2ReorgHandlingLogic the L2 reorg handling logic.
type L2ReorgHandlingLogic struct {
	client *ethclient.Client
}

// NewL2ReorgHandlingLogic creates L2 reorg handling logic.
func NewL2ReorgHandlingLogic(client *ethclient.Client) *L2ReorgHandlingLogic {
	return &L2ReorgHandlingLogic{
		client: client,
	}
}

// HandleL2Reorg performs L2 reorg handling by detecting reorgs and updating sync height.
func (l *L2ReorgHandlingLogic) HandleL2Reorg(ctx context.Context, blockNumber uint64, blockHash common.Hash) (bool, uint64, error) {
	l2ReorgDetected, err := l.detectL2Reorg(ctx, blockNumber, blockHash)
	if err != nil {
		log.Error("failed to detect L2 reorg", "err", err)
		return false, 0, err
	}

	if l2ReorgDetected {
		startHeight := uint64(1)
		if blockNumber > L2ReorgSafeDepth {
			startHeight = blockNumber - L2ReorgSafeDepth
		}
		return true, startHeight - 1, nil
	}

	return false, 0, nil
}

func (l *L2ReorgHandlingLogic) detectL2Reorg(ctx context.Context, blockNumber uint64, blockHash common.Hash) (bool, error) {
	currentHeader, err := l.client.HeaderByNumber(ctx, big.NewInt(0).SetUint64(blockNumber))
	if err != nil {
		log.Error("failed to get header by number", "height", blockNumber, "err", err)
		return false, err
	}

	if currentHeader == nil {
		log.Warn("cannot fetch remote block header", "height", blockNumber, "last block hash", blockHash.String())
		return true, nil
	}

	if blockHash != currentHeader.Hash() {
		log.Warn("block hash mismatch, reorg happened", "height", blockNumber, "last block hash", blockHash.String(), "current block hash", currentHeader.Hash().String())
		return true, nil
	}

	return false, nil
}
