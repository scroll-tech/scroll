package logic

import (
	"context"
	"math/big"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/bridge-history-api/internal/orm"
)

// L2ReorgSafeDepth represents the number of block confirmations considered safe against L2 chain reorganizations.
// Reorganizations at this depth under normal cases are extremely unlikely.
const L2ReorgSafeDepth = 256

// L2ReorgHandlingLogic the L2 reorg handling logic.
type L2ReorgHandlingLogic struct {
	client          *ethclient.Client
	crossMessageOrm *orm.CrossMessage
}

// NewL2ReorgHandlingLogic creates L2 reorg handling logic.
func NewL2ReorgHandlingLogic(db *gorm.DB, client *ethclient.Client) *L2ReorgHandlingLogic {
	return &L2ReorgHandlingLogic{
		client:          client,
		crossMessageOrm: orm.NewCrossMessage(db),
	}
}

// HandleL2Reorg performs L2 reorg handling by detecting reorgs and updating sync height.
func (l *L2ReorgHandlingLogic) HandleL2Reorg(ctx context.Context) (bool, uint64, error) {
	l2ReorgDetected, l2ReorgDetectedHeight, err := l.detectL2Reorg(ctx)
	if err != nil {
		log.Error("failed to detect L2 reorg", "err", err)
		return false, 0, err
	}

	if l2ReorgDetected {
		startHeight := uint64(1)
		if l2ReorgDetectedHeight > L2ReorgSafeDepth {
			startHeight = l2ReorgDetectedHeight - L2ReorgSafeDepth
		}
		return true, startHeight - 1, nil
	}

	return false, 0, nil
}

func (l *L2ReorgHandlingLogic) detectL2Reorg(ctx context.Context) (bool, uint64, error) {
	localBlockNumber, localBlockHash, err := l.crossMessageOrm.GetMaxL2BlockNumberAndHash(ctx)
	if err != nil {
		log.Error("failed to get max L1 block number and hash in cross message orm", "err", err)
		return false, 0, err
	}

	if localBlockNumber == 0 {
		log.Warn("no local info of latest block number and hash")
		return false, 0, nil
	}

	remoteHeader, err := l.client.HeaderByNumber(ctx, big.NewInt(0).SetUint64(localBlockNumber))
	if err != nil {
		log.Error("failed to get header by number", "height", localBlockNumber, "err", err)
		return false, 0, err
	}

	if remoteHeader == nil {
		log.Warn("cannot fetch remote block header", "blockNumber", localBlockNumber, "local hash", localBlockHash.String())
		return true, localBlockNumber, nil
	}

	if localBlockHash != remoteHeader.Hash() {
		log.Warn("block hash mismatch, reorg happened", "height", localBlockNumber, "local hash", localBlockHash.String(), "remote hash", remoteHeader.Hash().String())
		return true, localBlockNumber, nil
	}

	return false, 0, nil
}
