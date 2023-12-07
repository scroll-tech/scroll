package fetcher

import (
	"context"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	backendabi "scroll-tech/bridge-history-api/abi"
	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/logic"
	"scroll-tech/bridge-history-api/internal/orm"
	"scroll-tech/bridge-history-api/internal/utils"
)

// L1MessageFetcher fetches cross message events from L1 and saves them to database.
type L1MessageFetcher struct {
	ctx             context.Context
	cfg             *config.LayerConfig
	db              *gorm.DB
	crossMessageOrm *orm.CrossMessage
	batchEventOrm   *orm.BatchEvent
	client          *ethclient.Client
	addressList     []common.Address
}

// NewL1MessageFetcher creates a new L1MessageFetcher instance.
func NewL1MessageFetcher(ctx context.Context, cfg *config.LayerConfig, db *gorm.DB, client *ethclient.Client) (*L1MessageFetcher, error) {
	addressList := []common.Address{
		common.HexToAddress(cfg.ETHGatewayAddr),

		common.HexToAddress(cfg.StandardERC20Gateway),
		common.HexToAddress(cfg.CustomERC20GatewayAddr),
		common.HexToAddress(cfg.WETHGatewayAddr),
		common.HexToAddress(cfg.DAIGatewayAddr),

		common.HexToAddress(cfg.ERC721GatewayAddr),
		common.HexToAddress(cfg.ERC1155GatewayAddr),

		common.HexToAddress(cfg.MessengerAddr),

		common.HexToAddress(cfg.ScrollChainAddr),

		common.HexToAddress(cfg.MessageQueueAddr),
	}

	// Optional erc20 gateways.
	if cfg.USDCGatewayAddr != "" {
		addressList = append(addressList, common.HexToAddress(cfg.USDCGatewayAddr))
	}

	if cfg.LIDOGatewayAddr != "" {
		addressList = append(addressList, common.HexToAddress(cfg.LIDOGatewayAddr))
	}

	return &L1MessageFetcher{
		ctx:             ctx,
		cfg:             cfg,
		db:              db,
		crossMessageOrm: orm.NewCrossMessage(db),
		batchEventOrm:   orm.NewBatchEvent(db),
		client:          client,
		addressList:     addressList,
	}, nil
}

// Start starts the L1 message fetching process.
func (c *L1MessageFetcher) Start() {
	tick := time.NewTicker(time.Duration(c.cfg.BlockTime) * time.Second)
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				tick.Stop()
				return
			case <-tick.C:
				c.fetchAndSaveEvents(c.cfg.Confirmation)
			}
		}
	}()
}

func (c *L1MessageFetcher) fetchAndSaveEvents(confirmation uint64) {
	endHeight, err := utils.GetBlockNumber(c.ctx, c.client, confirmation)
	if err != nil {
		log.Error("failed to get L1 safe block number", "err", err)
		return
	}

	l1SentMessageProcessedHeight, err := c.crossMessageOrm.GetMessageProcessedHeightInDB(c.ctx, orm.MessageTypeL1SentMessage)
	if err != nil {
		log.Error("failed to get L1 cross message processed height", "err", err)
		return
	}
	startHeight := c.cfg.StartHeight
	if l1SentMessageProcessedHeight+1 > startHeight {
		startHeight = l1SentMessageProcessedHeight + 1
	}
	log.Info("fetch and save missing L1 events", "start height", startHeight, "config height", c.cfg.StartHeight, "db height", l1SentMessageProcessedHeight)

	for from := startHeight; from <= endHeight; from += c.cfg.FetchLimit {
		to := from + c.cfg.FetchLimit - 1
		if to > endHeight {
			to = endHeight
		}
		err = c.doFetchAndSaveEvents(c.ctx, from, to, c.addressList)
		if err != nil {
			log.Error("failed to fetch and save L1 events", "from", from, "to", to, "err", err)
			return
		}
	}
}

func (c *L1MessageFetcher) doFetchAndSaveEvents(ctx context.Context, from uint64, to uint64, addrList []common.Address) error {
	log.Info("fetch and save L1 events", "from", from, "to", to)
	var l1FailedGatewayRouterTxs []*orm.CrossMessage
	blockTimestampsMap := make(map[uint64]uint64)
	for number := from; number <= to; number++ {
		blockNumber := new(big.Int).SetUint64(number)
		block, err := c.client.BlockByNumber(ctx, blockNumber)
		if err != nil {
			log.Error("failed to get block by number", "number", blockNumber.String(), "err", err)
			return err
		}
		blockTimestampsMap[block.NumberU64()] = block.Time()

		for _, tx := range block.Transactions() {
			to := tx.To()
			if to == nil {
				continue
			}
			toAddress := to.String()
			if toAddress == c.cfg.GatewayRouterAddr {
				receipt, err := c.client.TransactionReceipt(ctx, tx.Hash())
				if err != nil {
					log.Error("Failed to get transaction receipt", "txHash", tx.Hash().String(), "err", err)
					return err
				}

				// Check if the transaction failed
				if receipt.Status == types.ReceiptStatusFailed {
					signer := types.NewLondonSigner(new(big.Int).SetUint64(tx.ChainId().Uint64()))
					sender, err := signer.Sender(tx)
					if err != nil {
						log.Error("get sender failed", "chain id", tx.ChainId().Uint64(), "tx hash", tx.Hash().String(), "err", err)
						return err
					}
					l1FailedGatewayRouterTxs = append(l1FailedGatewayRouterTxs, &orm.CrossMessage{
						L1TxHash:       tx.Hash().String(),
						MessageType:    int(orm.MessageTypeL1SentMessage),
						Sender:         sender.String(),
						Receiver:       (*tx.To()).String(),
						BlockTimestamp: block.Time(),
						TxStatus:       int(orm.TxStatusTypeSentFailed),
					})
				}
			}
		}
	}

	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from), // inclusive
		ToBlock:   new(big.Int).SetUint64(to),   // inclusive
		Addresses: addrList,
		Topics:    make([][]common.Hash, 1),
	}
	query.Topics[0] = make([]common.Hash, 13)
	query.Topics[0][0] = backendabi.L1DepositETHSig
	query.Topics[0][1] = backendabi.L1DepositERC20Sig
	query.Topics[0][2] = backendabi.L1DepositERC721Sig
	query.Topics[0][3] = backendabi.L1DepositERC1155Sig
	query.Topics[0][4] = backendabi.L1SentMessageEventSig
	query.Topics[0][5] = backendabi.L1RelayedMessageEventSig
	query.Topics[0][6] = backendabi.L1FailedRelayedMessageEventSig
	query.Topics[0][7] = backendabi.L1CommitBatchEventSig
	query.Topics[0][8] = backendabi.L1RevertBatchEventSig
	query.Topics[0][9] = backendabi.L1FinalizeBatchEventSig
	query.Topics[0][10] = backendabi.L1QueueTransactionEventSig
	query.Topics[0][11] = backendabi.L1DequeueTransactionEventSig
	query.Topics[0][12] = backendabi.L1DropTransactionEventSig

	logs, err := c.client.FilterLogs(ctx, query)
	if err != nil {
		log.Error("failed to filter L1 event logs", "from", from, "to", to, "err", err)
		return err
	}
	l1DepositMessages, l1RelayedMessages, err := logic.ParseL1CrossChainEventLogs(ctx, logs, blockTimestampsMap, c.client)
	if err != nil {
		log.Error("failed to parse L1 cross chain event logs", "from", from, "to", to, "err", err)
		return err
	}
	l1BatchEvents, err := logic.ParseL1BatchEventLogs(ctx, logs, blockTimestampsMap, c.client)
	if err != nil {
		log.Error("failed to parse L1 batch event logs", "from", from, "to", to, "err", err)
		return err
	}
	l1MessageQueueEvents, err := logic.ParseL1MessageQueueEventLogs(ctx, logs, blockTimestampsMap, c.client)
	if err != nil {
		log.Error("failed to parse L1 message queue event logs", "from", from, "to", to, "err", err)
		return err
	}
	err = c.db.Transaction(func(tx *gorm.DB) error {
		if txErr := c.crossMessageOrm.InsertOrUpdateL1Messages(ctx, l1DepositMessages, tx); txErr != nil {
			log.Error("failed to insert L1 deposit messages", "from", from, "to", to, "err", txErr)
			return txErr
		}
		if txErr := c.crossMessageOrm.InsertOrUpdateL1RelayedMessagesOfL2Withdrawals(ctx, l1RelayedMessages, tx); txErr != nil {
			log.Error("failed to update L1 relayed messages of L2 withdrawals", "from", from, "to", to, "err", txErr)
			return txErr
		}
		if txErr := c.batchEventOrm.InsertOrUpdateBatchEvents(ctx, l1BatchEvents, tx); txErr != nil {
			log.Error("failed to insert or update batch events", "from", from, "to", to, "err", txErr)
			return txErr
		}
		if txErr := c.crossMessageOrm.UpdateL1MessageQueueEventsInfo(ctx, l1MessageQueueEvents, tx); txErr != nil {
			log.Error("failed to insert L1 message queue events", "from", from, "to", to, "err", txErr)
			return txErr
		}
		if txErr := c.crossMessageOrm.InsertFailedGatewayRouterTxs(ctx, l1FailedGatewayRouterTxs, tx); txErr != nil {
			log.Error("failed to insert L1 failed gateway router transactions", "from", from, "to", to, "err", txErr)
			return txErr
		}
		return nil
	})
	if err != nil {
		log.Error("failed to update db of L1 events", "from", from, "to", to, "err", err)
		return err
	}
	if err = c.updateBatchIndexAndStatus(ctx); err != nil {
		log.Error("failed to update batch index and status", "err", err)
		return err
	}
	return nil
}

func (c *L1MessageFetcher) updateBatchIndexAndStatus(ctx context.Context) error {
	latestMessageHeight, err := c.crossMessageOrm.GetLatestFinalizedL2WithdrawalBlockHeight(ctx)
	if err != nil {
		log.Error("failed to get latest finalized L2 sent message block height", "error", err)
		return err
	}
	batches, err := c.batchEventOrm.GetBatchesGEBlockHeight(ctx, latestMessageHeight+1)
	if err != nil {
		log.Error("failed to get batches >= block height", "error", err)
		return err
	}
	for _, batch := range batches {
		log.Info("update batch info of L2 withdrawals", "index", batch.BatchIndex, "start", batch.StartBlockNumber, "end", batch.EndBlockNumber)
		if err := c.crossMessageOrm.UpdateBatchStatusOfL2Withdrawals(ctx, batch.StartBlockNumber, batch.EndBlockNumber, batch.BatchIndex); err != nil {
			log.Error("failed to update batch status of L2 sent messages", "start", batch.StartBlockNumber, "end", batch.EndBlockNumber, "index", batch.BatchIndex, "error", err)
			return err
		}
	}
	return nil
}
