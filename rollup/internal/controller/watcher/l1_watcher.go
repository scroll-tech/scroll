package watcher

import (
	"context"
	"errors"
	"math/big"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/consensus/misc"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	"scroll-tech/rollup/internal/orm"
)

// L1WatcherClient will listen for smart contract events from Eth L1.
type L1WatcherClient struct {
	ctx          context.Context
	client       *ethclient.Client
	l1MessageOrm *orm.L1Message
	l1BlockOrm   *orm.L1Block
	batchOrm     *orm.Batch

	// The number of new blocks to wait for a block to be confirmed
	confirmations rpc.BlockNumber

	// The height of the block that the watcher has retrieved event logs
	processedMsgHeight uint64
	// The height of the block that the watcher has retrieved header rlp
	processedBlockHeight uint64

	metrics *l1WatcherMetrics
}

// NewL1WatcherClient returns a new instance of L1WatcherClient.
func NewL1WatcherClient(ctx context.Context, client *ethclient.Client, startHeight uint64, confirmations rpc.BlockNumber, db *gorm.DB, reg prometheus.Registerer) *L1WatcherClient {
	l1MessageOrm := orm.NewL1Message(db)
	savedHeight, err := l1MessageOrm.GetLayer1LatestWatchedHeight()
	if err != nil {
		log.Warn("Failed to fetch height from db", "err", err)
		savedHeight = 0
	}
	if savedHeight < int64(startHeight) {
		savedHeight = int64(startHeight)
	}

	l1BlockOrm := orm.NewL1Block(db)
	savedL1BlockHeight, err := l1BlockOrm.GetLatestL1BlockHeight(ctx)
	if err != nil {
		log.Warn("Failed to fetch latest L1 block height from db", "err", err)
		savedL1BlockHeight = 0
	}
	if savedL1BlockHeight < startHeight {
		savedL1BlockHeight = startHeight
	}

	return &L1WatcherClient{
		ctx:           ctx,
		client:        client,
		l1MessageOrm:  l1MessageOrm,
		l1BlockOrm:    l1BlockOrm,
		batchOrm:      orm.NewBatch(db),
		confirmations: confirmations,

		processedMsgHeight:   uint64(savedHeight),
		processedBlockHeight: savedL1BlockHeight,
		metrics:              initL1WatcherMetrics(reg),
	}
}

// ProcessedBlockHeight get processedBlockHeight
// Currently only use for unit test
func (w *L1WatcherClient) ProcessedBlockHeight() uint64 {
	return w.processedBlockHeight
}

// Confirmations get confirmations
// Currently only use for unit test
func (w *L1WatcherClient) Confirmations() rpc.BlockNumber {
	return w.confirmations
}

// SetConfirmations set the confirmations for L1WatcherClient
// Currently only use for unit test
func (w *L1WatcherClient) SetConfirmations(confirmations rpc.BlockNumber) {
	w.confirmations = confirmations
}

// FetchBlockHeader pull latest L1 blocks and save in DB
func (w *L1WatcherClient) FetchBlockHeader(blockHeight uint64) error {
	w.metrics.l1WatcherFetchBlockHeaderTotal.Inc()

	var block *gethTypes.Header
	block, err := w.client.HeaderByNumber(w.ctx, big.NewInt(int64(blockHeight)))
	if err != nil {
		log.Warn("Failed to get block", "height", blockHeight, "err", err)
		return err
	}

	if block == nil {
		log.Warn("Received nil block", "height", blockHeight)
		return errors.New("received nil block")
	}

	var baseFee uint64
	if block.BaseFee != nil {
		baseFee = block.BaseFee.Uint64()
	}

	var blobBaseFee uint64
	if excess := block.ExcessBlobGas; excess != nil {
		blobBaseFee = misc.CalcBlobFee(*excess).Uint64()
	}

	l1Block := orm.L1Block{
		Number:          blockHeight,
		Hash:            block.Hash().String(),
		BaseFee:         baseFee,
		BlobBaseFee:     blobBaseFee,
		GasOracleStatus: int16(types.GasOraclePending),
	}

	err = w.l1BlockOrm.InsertL1Blocks(w.ctx, []orm.L1Block{l1Block})
	if err != nil {
		log.Warn("Failed to insert L1 block to db", "blockHeight", blockHeight, "err", err)
		return err
	}

	// update processed height
	w.processedBlockHeight = blockHeight
	w.metrics.l1WatcherFetchBlockHeaderProcessedBlockHeight.Set(float64(w.processedBlockHeight))
	return nil
}
