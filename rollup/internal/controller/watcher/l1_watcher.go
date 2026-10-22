package watcher

import (
	"context"
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	"scroll-tech/rollup/internal/orm"
)

// L1WatcherClient will listen for smart contract events from Eth L1.
type L1WatcherClient struct {
	ctx        context.Context
	rpcClient  *rpc.Client
	l1BlockOrm *orm.L1Block

	// The height of the block that the watcher has retrieved header rlp
	processedBlockHeight uint64

	metrics *l1WatcherMetrics
}

// NewL1WatcherClient returns a new instance of L1WatcherClient.
func NewL1WatcherClient(ctx context.Context, rpcClient *rpc.Client, startHeight uint64, db *gorm.DB, reg prometheus.Registerer) *L1WatcherClient {
	l1BlockOrm := orm.NewL1Block(db)
	savedL1BlockHeight, err := l1BlockOrm.GetLatestL1BlockHeight(ctx)
	if err != nil {
		log.Warn("Failed to fetch latest L1 block height from db", "err", err)
		savedL1BlockHeight = 0
	}
	if savedL1BlockHeight < startHeight {
		savedL1BlockHeight = startHeight
	}

	w := &L1WatcherClient{
		ctx:        ctx,
		rpcClient:  rpcClient,
		l1BlockOrm: l1BlockOrm,

		processedBlockHeight: savedL1BlockHeight,
		metrics:              initL1WatcherMetrics(reg),
	}

	// Seed the gauge from the resumed height instead of starting from 0.
	w.metrics.l1WatcherFetchBlockHeaderProcessedBlockHeight.Set(float64(w.processedBlockHeight))

	return w
}

// ProcessedBlockHeight get processedBlockHeight
// Currently only use for unit test
func (w *L1WatcherClient) ProcessedBlockHeight() uint64 {
	return w.processedBlockHeight
}

// FetchBlockHeader pull latest L1 blocks and save in DB
func (w *L1WatcherClient) FetchBlockHeader(blockHeight uint64) error {
	w.metrics.l1WatcherFetchBlockHeaderTotal.Inc()

	// Use the RPC-reported hash, not types.Header.Hash(). The SDK header follows Scroll L2 rules:
	// it lacks newer L1 fields (e.g. EIP-7685 requestsHash) and encodes Scroll-only fields after baseFee.
	// For post-Shanghai L1 blocks the computed hash is therefore wrong.
	var block *struct {
		Hash          common.Hash     `json:"hash"`
		BaseFee       *hexutil.Big    `json:"baseFeePerGas"`
		ExcessBlobGas *hexutil.Uint64 `json:"excessBlobGas"`
	}
	err := w.rpcClient.CallContext(w.ctx, &block, "eth_getBlockByNumber", hexutil.EncodeUint64(blockHeight), false)
	if err != nil {
		log.Warn("Failed to get block", "height", blockHeight, "err", err)
		return err
	}

	if block == nil {
		log.Warn("Received nil block", "height", blockHeight)
		return errors.New("received nil block")
	}

	if block.Hash == (common.Hash{}) {
		log.Warn("Received block without hash", "height", blockHeight)
		return errors.New("received block without hash")
	}

	var baseFee uint64
	if block.BaseFee != nil {
		baseFee = block.BaseFee.ToInt().Uint64()
	}

	var blobBaseFee uint64
	if excess := block.ExcessBlobGas; excess != nil {
		// Leave it up to the L1 node to compute the correct blob base fee.
		// Previously we would compute it locally using `CalcBlobFee`, but
		// that approach requires syncing any future L1 configuration changes.
		// Note: The fetched blob base fee might not correspond to the block
		// that we fetched in the previous step, but this is acceptable.
		var blobBaseFeeHex hexutil.Big
		if err := w.rpcClient.CallContext(w.ctx, &blobBaseFeeHex, "eth_blobBaseFee"); err != nil {
			return fmt.Errorf("failed to call eth_blobBaseFee, err: %w", err)
		}
		// A correct L1 node could not return a value that overflows uint64
		blobBaseFee = blobBaseFeeHex.ToInt().Uint64()
	}

	l1Block := orm.L1Block{
		Number:          blockHeight,
		Hash:            block.Hash.String(),
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
