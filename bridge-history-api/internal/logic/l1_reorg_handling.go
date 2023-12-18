package logic

import (
	"context"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/bridge-history-api/internal/orm"
)

// L1ReorgSafeDepth represents the number of block confirmations considered safe against L1 chain reorganizations.
// Reorganizations at this depth under normal cases are extremely unlikely.
const L1ReorgSafeDepth = 64

// L1ReorgHandlingLogic the L1 reorg handling logic.
type L1ReorgHandlingLogic struct {
	client          *ethclient.Client
	crossMessageOrm *orm.CrossMessage
	batchEventOrm   *orm.BatchEvent
}

// NewL1ReorgHandlingLogic creates L1 reorg handling logic.
func NewL1ReorgHandlingLogic(db *gorm.DB, client *ethclient.Client) *L1ReorgHandlingLogic {
	return &L1ReorgHandlingLogic{
		client:          client,
		crossMessageOrm: orm.NewCrossMessage(db),
		batchEventOrm:   orm.NewBatchEvent(db),
	}
}

// HandleL1Reorg performs L1 reorg handling by detecting reorgs and updating sync height.
func (l *L1ReorgHandlingLogic) HandleL1Reorg(ctx context.Context) (bool, uint64, error) {
	reorgDetected, reorgDetectedHeight, err := l.detectReorg(ctx)
	if err != nil {
		log.Error("failed to detect reorg", "err", err)
		return false, 0, err
	}

	if reorgDetected {
		resyncHeight := uint64(1)
		if reorgDetectedHeight > L1ReorgSafeDepth {
			resyncHeight = reorgDetectedHeight - L1ReorgSafeDepth
		}
		return true, resyncHeight - 1, nil
	}

	return false, 0, nil
}

func (l *L1ReorgHandlingLogic) detectReorg(ctx context.Context) (bool, uint64, error) {
	batchBlockNumber, batchBlockHash, err := l.batchEventOrm.GetMaxL1BlockNumberAndHash(ctx)
	if err != nil {
		log.Error("failed to get max L1 block number and hash in batch event orm", "err", err)
		return false, 0, err
	}

	messageBlockNumber, messageBlockHash, err := l.crossMessageOrm.GetMaxL1BlockNumberAndHash(ctx)
	if err != nil {
		log.Error("failed to get max L1 block number and hash in cross message orm", "err", err)
		return false, 0, err
	}

	var localBlockNumber uint64
	var localBlockHash common.Hash
	if batchBlockNumber > messageBlockNumber {
		localBlockNumber = batchBlockNumber
		localBlockHash = batchBlockHash
	} else {
		localBlockNumber = messageBlockNumber
		localBlockHash = messageBlockHash
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
