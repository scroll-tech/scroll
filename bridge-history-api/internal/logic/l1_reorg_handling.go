package logic

import (
	"context"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
)

// L1ReorgSafeDepth represents the number of block confirmations considered safe against L1 chain reorganizations.
// Reorganizations at this depth under normal cases are extremely unlikely.
const L1ReorgSafeDepth = 64

// L1ReorgHandlingLogic the L1 reorg handling logic.
type L1ReorgHandlingLogic struct {
	client *ethclient.Client
}

// NewL1ReorgHandlingLogic creates L1 reorg handling logic.
func NewL1ReorgHandlingLogic(client *ethclient.Client) *L1ReorgHandlingLogic {
	return &L1ReorgHandlingLogic{
		client: client,
	}
}

// HandleL1Reorg performs L1 reorg handling by detecting reorgs and updating sync height.
func (l *L1ReorgHandlingLogic) HandleL1Reorg(ctx context.Context, blockNumber uint64, blockHash common.Hash) (bool, uint64, error) {
	reorgDetected, err := l.detectReorg(ctx, blockNumber, blockHash)
	if err != nil {
		log.Error("failed to detect reorg", "err", err)
		return false, 0, err
	}

	if reorgDetected {
		var resyncHeight uint64
		if blockNumber > L1ReorgSafeDepth {
			resyncHeight = blockNumber - L1ReorgSafeDepth
		}
		return true, resyncHeight, nil
	}

	return false, 0, nil
}

func (l *L1ReorgHandlingLogic) detectReorg(ctx context.Context, blockNumber uint64, blockHash common.Hash) (bool, error) {
	currentHeader, err := l.client.HeaderByNumber(ctx, big.NewInt(0).SetUint64(blockNumber))
	if err != nil {
		log.Error("failed to get L1 header by number", "height", blockNumber, "err", err)
		return false, err
	}

	if currentHeader == nil {
		log.Warn("cannot fetch current L1 block header", "height", blockNumber, "last block hash", blockHash.String())
		return true, nil
	}

	if blockHash != currentHeader.Hash() {
		log.Warn("block hash mismatch, L1 reorg happened", "height", blockNumber, "last block hash", blockHash.String(), "current block hash", currentHeader.Hash().String())
		return true, nil
	}

	return false, nil
}
