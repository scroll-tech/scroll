package logic

import (
	"context"
	"math/big"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	backendabi "scroll-tech/bridge-history-api/abi"
	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/orm"
	"scroll-tech/bridge-history-api/internal/utils"
)

// L1FilterResult l1 fetcher result
type L1FilterResult struct {
	FailedGatewayRouterTxs []*orm.CrossMessage
	DepositMessages        []*orm.CrossMessage
	RelayedMessages        []*orm.CrossMessage
	BatchEvents            []*orm.BatchEvent
	MessageQueueEvents     []*orm.MessageQueueEvent
}

// L1FetcherLogic the l1 fetcher's logic
type L1FetcherLogic struct {
	cfg             *config.LayerConfig
	client          *ethclient.Client
	addressList     []common.Address
	parser          *L1EventParser
	db              *gorm.DB
	crossMessageOrm *orm.CrossMessage
	batchEventOrm   *orm.BatchEvent
}

// NewL1FetcherLogic create l1 fetcher logic
func NewL1FetcherLogic(cfg *config.LayerConfig, db *gorm.DB, client *ethclient.Client) *L1FetcherLogic {
	addressList := []common.Address{
		common.HexToAddress(cfg.ETHGatewayAddr),

		common.HexToAddress(cfg.StandardERC20GatewayAddr),
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

	return &L1FetcherLogic{
		db:              db,
		crossMessageOrm: orm.NewCrossMessage(db),
		batchEventOrm:   orm.NewBatchEvent(db),
		cfg:             cfg,
		client:          client,
		addressList:     addressList,
		parser:          NewL1EventParser(),
	}
}

func (f *L1FetcherLogic) gatewayRouterFailedTxs(ctx context.Context, from, to uint64) (map[uint64]uint64, []*orm.CrossMessage, error) {
	blocks, err := utils.GetL1BlocksInRange(ctx, f.client, from, to)
	if err != nil {
		log.Error("failed to get L1 blocks in range", "from", from, "to", to, "err", err)
		return nil, nil, err
	}

	blockTimestampsMap := make(map[uint64]uint64)
	var l1FailedGatewayRouterTxs []*orm.CrossMessage
	for i := from; i <= to; i++ {
		block := blocks[i-from]
		blockTimestampsMap[block.NumberU64()] = block.Time()

		for _, tx := range block.Transactions() {
			txTo := tx.To()
			if txTo == nil {
				continue
			}
			toAddress := txTo.String()

			if toAddress != f.cfg.GatewayRouterAddr {
				continue
			}

			var receipt *types.Receipt
			receipt, receiptErr := f.client.TransactionReceipt(ctx, tx.Hash())
			if receiptErr != nil {
				log.Error("Failed to get transaction receipt", "txHash", tx.Hash().String(), "err", receiptErr)
				return nil, nil, receiptErr
			}

			// Check if the transaction failed
			if receipt.Status != types.ReceiptStatusFailed {
				continue
			}

			signer := types.NewLondonSigner(new(big.Int).SetUint64(tx.ChainId().Uint64()))
			sender, senderErr := signer.Sender(tx)
			if senderErr != nil {
				log.Error("get sender failed", "chain id", tx.ChainId().Uint64(), "tx hash", tx.Hash().String(), "err", senderErr)
				return nil, nil, senderErr
			}

			l1FailedGatewayRouterTxs = append(l1FailedGatewayRouterTxs, &orm.CrossMessage{
				L1TxHash:       tx.Hash().String(),
				MessageType:    int(orm.MessageTypeL1SentMessage),
				Sender:         sender.String(),
				Receiver:       (*tx.To()).String(),
				L1BlockNumber:  receipt.BlockNumber.Uint64(),
				BlockTimestamp: block.Time(),
				TxStatus:       int(orm.TxStatusTypeSentFailed),
			})
		}
	}
	return blockTimestampsMap, l1FailedGatewayRouterTxs, nil
}

func (f *L1FetcherLogic) l1FetcherLogs(ctx context.Context, from, to uint64) ([]types.Log, error) {
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from), // inclusive
		ToBlock:   new(big.Int).SetUint64(to),   // inclusive
		Addresses: f.addressList,
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

	eventLogs, err := f.client.FilterLogs(ctx, query)
	if err != nil {
		log.Error("failed to filter L1 event logs", "from", from, "to", to, "err", err)
		return nil, err
	}
	return eventLogs, nil
}

// L1Fetcher l1 fetcher
func (f *L1FetcherLogic) L1Fetcher(ctx context.Context, from, to uint64) (*L1FilterResult, error) {
	log.Info("fetch and save L1 events", "from", from, "to", to)

	blockTimestampsMap, l1FailedGatewayRouterTxs, err := f.gatewayRouterFailedTxs(ctx, from, to)
	if err != nil {
		log.Error("L1Fetcher gatewayRouterFailedTxs failed", "from", from, "to", to, "error", err)
		return nil, err
	}

	eventLogs, err := f.l1FetcherLogs(ctx, from, to)
	if err != nil {
		log.Error("L1Fetcher l1FetcherLogs failed", "from", from, "to", to, "error", err)
		return nil, err
	}

	l1DepositMessages, l1RelayedMessages, err := f.parser.ParseL1CrossChainEventLogs(eventLogs, blockTimestampsMap)
	if err != nil {
		log.Error("failed to parse L1 cross chain event logs", "from", from, "to", to, "err", err)
		return nil, err
	}

	l1BatchEvents, err := f.parser.ParseL1BatchEventLogs(ctx, eventLogs, f.client)
	if err != nil {
		log.Error("failed to parse L1 batch event logs", "from", from, "to", to, "err", err)
		return nil, err
	}

	l1MessageQueueEvents, err := f.parser.ParseL1MessageQueueEventLogs(eventLogs)
	if err != nil {
		log.Error("failed to parse L1 message queue event logs", "from", from, "to", to, "err", err)
		return nil, err
	}

	res := L1FilterResult{
		FailedGatewayRouterTxs: l1FailedGatewayRouterTxs,
		DepositMessages:        l1DepositMessages,
		RelayedMessages:        l1RelayedMessages,
		BatchEvents:            l1BatchEvents,
		MessageQueueEvents:     l1MessageQueueEvents,
	}
	return &res, nil
}
