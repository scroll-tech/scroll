package logic

import (
	"context"
	"math/big"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	backendabi "scroll-tech/bridge-history-api/abi"
	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/orm"
	btypes "scroll-tech/bridge-history-api/internal/types"
	"scroll-tech/bridge-history-api/internal/utils"
)

// L1ReorgSafeDepth represents the number of block confirmations considered safe against L1 chain reorganizations.
// Reorganizations at this depth under normal cases are extremely unlikely.
const L1ReorgSafeDepth = 64

// L1FilterResult L1 fetcher result
type L1FilterResult struct {
	DepositMessages          []*orm.CrossMessage
	RelayedMessages          []*orm.CrossMessage
	BatchEvents              []*orm.BatchEvent
	MessageQueueEvents       []*orm.MessageQueueEvent
	RevertedTxs              []*orm.CrossMessage
	BridgeBatchDepositEvents []*orm.BridgeBatchDepositEvent
}

// L1FetcherLogic the L1 fetcher logic
type L1FetcherLogic struct {
	cfg             *config.FetcherConfig
	client          *ethclient.Client
	addressList     []common.Address
	gatewayList     []common.Address
	parser          *L1EventParser
	db              *gorm.DB
	crossMessageOrm *orm.CrossMessage
	batchEventOrm   *orm.BatchEvent

	l1FetcherLogicFetchedTotal *prometheus.CounterVec
}

// NewL1FetcherLogic creates L1 fetcher logic
func NewL1FetcherLogic(cfg *config.FetcherConfig, db *gorm.DB, client *ethclient.Client) *L1FetcherLogic {
	addressList := []common.Address{
		common.HexToAddress(cfg.StandardERC20GatewayAddr),
		common.HexToAddress(cfg.CustomERC20GatewayAddr),
		common.HexToAddress(cfg.DAIGatewayAddr),

		common.HexToAddress(cfg.ERC721GatewayAddr),
		common.HexToAddress(cfg.ERC1155GatewayAddr),

		common.HexToAddress(cfg.MessengerAddr),

		common.HexToAddress(cfg.ScrollChainAddr),

		common.HexToAddress(cfg.MessageQueueAddr),
	}

	gatewayList := []common.Address{
		common.HexToAddress(cfg.StandardERC20GatewayAddr),
		common.HexToAddress(cfg.CustomERC20GatewayAddr),
		common.HexToAddress(cfg.DAIGatewayAddr),

		common.HexToAddress(cfg.ERC721GatewayAddr),
		common.HexToAddress(cfg.ERC1155GatewayAddr),

		common.HexToAddress(cfg.MessengerAddr),

		common.HexToAddress(cfg.GatewayRouterAddr),
	}

	// Optional gateways.
	if common.HexToAddress(cfg.USDCGatewayAddr) != (common.Address{}) {
		addressList = append(addressList, common.HexToAddress(cfg.USDCGatewayAddr))
		gatewayList = append(gatewayList, common.HexToAddress(cfg.USDCGatewayAddr))
	}

	if common.HexToAddress(cfg.LIDOGatewayAddr) != (common.Address{}) {
		addressList = append(addressList, common.HexToAddress(cfg.LIDOGatewayAddr))
		gatewayList = append(gatewayList, common.HexToAddress(cfg.LIDOGatewayAddr))
	}

	if common.HexToAddress(cfg.PufferGatewayAddr) != (common.Address{}) {
		addressList = append(addressList, common.HexToAddress(cfg.PufferGatewayAddr))
		gatewayList = append(gatewayList, common.HexToAddress(cfg.PufferGatewayAddr))
	}

	if common.HexToAddress(cfg.BatchBridgeGatewayAddr) != (common.Address{}) {
		addressList = append(addressList, common.HexToAddress(cfg.BatchBridgeGatewayAddr))
		gatewayList = append(gatewayList, common.HexToAddress(cfg.BatchBridgeGatewayAddr))
	}

	if common.HexToAddress(cfg.ETHGatewayAddr) != (common.Address{}) {
		addressList = append(addressList, common.HexToAddress(cfg.ETHGatewayAddr))
		gatewayList = append(gatewayList, common.HexToAddress(cfg.ETHGatewayAddr))
	}

	if common.HexToAddress(cfg.WETHGatewayAddr) != (common.Address{}) {
		addressList = append(addressList, common.HexToAddress(cfg.WETHGatewayAddr))
		gatewayList = append(gatewayList, common.HexToAddress(cfg.WETHGatewayAddr))
	}

	if common.HexToAddress(cfg.GasTokenGatewayAddr) != (common.Address{}) {
		addressList = append(addressList, common.HexToAddress(cfg.GasTokenGatewayAddr))
		gatewayList = append(gatewayList, common.HexToAddress(cfg.GasTokenGatewayAddr))
	}

	if common.HexToAddress(cfg.WrappedTokenGatewayAddr) != (common.Address{}) {
		addressList = append(addressList, common.HexToAddress(cfg.WrappedTokenGatewayAddr))
		gatewayList = append(gatewayList, common.HexToAddress(cfg.WrappedTokenGatewayAddr))
	}

	log.Info("L1 Fetcher configured with the following address list", "addresses", addressList, "gateways", gatewayList)

	f := &L1FetcherLogic{
		db:              db,
		crossMessageOrm: orm.NewCrossMessage(db),
		batchEventOrm:   orm.NewBatchEvent(db),
		cfg:             cfg,
		client:          client,
		addressList:     addressList,
		gatewayList:     gatewayList,
		parser:          NewL1EventParser(cfg, client),
	}

	reg := prometheus.DefaultRegisterer
	f.l1FetcherLogicFetchedTotal = promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
		Name: "L1_fetcher_logic_fetched_total",
		Help: "The total number of events or failed txs fetched in L1 fetcher logic.",
	}, []string{"type"})

	return f
}

func (f *L1FetcherLogic) getBlocksAndDetectReorg(ctx context.Context, from, to uint64, lastBlockHash common.Hash) (bool, uint64, common.Hash, []*types.Block, error) {
	blocks, err := utils.GetBlocksInRange(ctx, f.client, from, to)
	if err != nil {
		log.Error("failed to get L1 blocks in range", "from", from, "to", to, "err", err)
		return false, 0, common.Hash{}, nil, err
	}

	for _, block := range blocks {
		if block.ParentHash() != lastBlockHash {
			log.Warn("L1 reorg detected", "reorg height", block.NumberU64()-1, "expected hash", block.ParentHash().String(), "local hash", lastBlockHash.String())
			var resyncHeight uint64
			if block.NumberU64() > L1ReorgSafeDepth+1 {
				resyncHeight = block.NumberU64() - L1ReorgSafeDepth - 1
			}
			header, err := f.client.HeaderByNumber(ctx, new(big.Int).SetUint64(resyncHeight))
			if err != nil {
				log.Error("failed to get L1 header by number", "block number", resyncHeight, "err", err)
				return false, 0, common.Hash{}, nil, err
			}
			return true, resyncHeight, header.Hash(), nil, nil
		}
		lastBlockHash = block.Hash()
	}

	return false, 0, lastBlockHash, blocks, nil
}

func (f *L1FetcherLogic) getRevertedTxs(ctx context.Context, from, to uint64, blocks []*types.Block) (map[uint64]uint64, []*orm.CrossMessage, error) {
	var l1RevertedTxs []*orm.CrossMessage
	blockTimestampsMap := make(map[uint64]uint64)

	for i := from; i <= to; i++ {
		block := blocks[i-from]
		blockTimestampsMap[block.NumberU64()] = block.Time()

		for _, tx := range block.Transactions() {
			// Gateways: L1 deposit.
			// Messenger: L1 deposit retry (replayMessage), L1 deposit refund (dropMessage), L2 withdrawal's claim (relayMessageWithProof).
			if !isTransactionToGateway(tx, f.gatewayList) {
				continue
			}

			var receipt *types.Receipt
			receipt, receiptErr := f.client.TransactionReceipt(ctx, tx.Hash())
			if receiptErr != nil {
				log.Error("Failed to get transaction receipt", "txHash", tx.Hash().String(), "err", receiptErr)
				return nil, nil, receiptErr
			}

			// Check if the transaction is failed
			if receipt.Status != types.ReceiptStatusFailed {
				continue
			}

			signer := types.LatestSignerForChainID(new(big.Int).SetUint64(tx.ChainId().Uint64()))
			sender, senderErr := signer.Sender(tx)
			if senderErr != nil {
				log.Error("get sender failed", "chain id", tx.ChainId().Uint64(), "tx hash", tx.Hash().String(), "err", senderErr)
				return nil, nil, senderErr
			}

			l1RevertedTxs = append(l1RevertedTxs, &orm.CrossMessage{
				L1TxHash:       tx.Hash().String(),
				MessageType:    int(btypes.MessageTypeL1SentMessage),
				Sender:         sender.String(),
				Receiver:       (*tx.To()).String(),
				L1BlockNumber:  receipt.BlockNumber.Uint64(),
				BlockTimestamp: block.Time(),
				TxStatus:       int(btypes.TxStatusTypeSentTxReverted),
			})
		}
	}
	return blockTimestampsMap, l1RevertedTxs, nil
}

func (f *L1FetcherLogic) l1FetcherLogs(ctx context.Context, from, to uint64) ([]types.Log, error) {
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from), // inclusive
		ToBlock:   new(big.Int).SetUint64(to),   // inclusive
		Addresses: f.addressList,
		Topics:    make([][]common.Hash, 1),
	}

	query.Topics[0] = make([]common.Hash, 16)
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
	query.Topics[0][13] = backendabi.L1ResetDequeuedTransactionEventSig
	query.Topics[0][14] = backendabi.L1BridgeBatchDepositSig
	query.Topics[0][15] = backendabi.L1DepositWrappedTokenSig

	eventLogs, err := f.client.FilterLogs(ctx, query)
	if err != nil {
		log.Error("failed to filter L1 event logs", "from", from, "to", to, "err", err)
		return nil, err
	}
	return eventLogs, nil
}

// L1Fetcher L1 fetcher
func (f *L1FetcherLogic) L1Fetcher(ctx context.Context, from, to uint64, lastBlockHash common.Hash) (bool, uint64, common.Hash, *L1FilterResult, error) {
	log.Info("fetch and save L1 events", "from", from, "to", to)

	isReorg, reorgHeight, blockHash, blocks, getErr := f.getBlocksAndDetectReorg(ctx, from, to, lastBlockHash)
	if getErr != nil {
		log.Error("L1Fetcher getBlocksAndDetectReorg failed", "from", from, "to", to, "error", getErr)
		return false, 0, common.Hash{}, nil, getErr
	}

	if isReorg {
		return isReorg, reorgHeight, blockHash, nil, nil
	}

	blockTimestampsMap, l1RevertedTxs, err := f.getRevertedTxs(ctx, from, to, blocks)
	if err != nil {
		log.Error("L1Fetcher getRevertedTxs failed", "from", from, "to", to, "error", err)
		return false, 0, common.Hash{}, nil, err
	}

	eventLogs, err := f.l1FetcherLogs(ctx, from, to)
	if err != nil {
		log.Error("L1Fetcher l1FetcherLogs failed", "from", from, "to", to, "error", err)
		return false, 0, common.Hash{}, nil, err
	}

	l1DepositMessages, l1RelayedMessages, l1BridgeBatchDepositMessages, err := f.parser.ParseL1CrossChainEventLogs(ctx, eventLogs, blockTimestampsMap)
	if err != nil {
		log.Error("failed to parse L1 cross chain event logs", "from", from, "to", to, "err", err)
		return false, 0, common.Hash{}, nil, err
	}

	l1BatchEvents, err := f.parser.ParseL1BatchEventLogs(ctx, eventLogs, f.client)
	if err != nil {
		log.Error("failed to parse L1 batch event logs", "from", from, "to", to, "err", err)
		return false, 0, common.Hash{}, nil, err
	}

	l1MessageQueueEvents, err := f.parser.ParseL1MessageQueueEventLogs(eventLogs, l1DepositMessages)
	if err != nil {
		log.Error("failed to parse L1 message queue event logs", "from", from, "to", to, "err", err)
		return false, 0, common.Hash{}, nil, err
	}

	res := L1FilterResult{
		DepositMessages:          l1DepositMessages,
		RelayedMessages:          l1RelayedMessages,
		BatchEvents:              l1BatchEvents,
		MessageQueueEvents:       l1MessageQueueEvents,
		RevertedTxs:              l1RevertedTxs,
		BridgeBatchDepositEvents: l1BridgeBatchDepositMessages,
	}

	f.updateMetrics(res)

	return false, 0, blockHash, &res, nil
}

func (f *L1FetcherLogic) updateMetrics(res L1FilterResult) {
	f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_failed_gateway_router_transaction").Add(float64(len(res.RevertedTxs)))

	for _, depositMessage := range res.DepositMessages {
		switch btypes.TokenType(depositMessage.TokenType) {
		case btypes.TokenTypeETH:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_deposit_eth").Add(1)
		case btypes.TokenTypeERC20:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_deposit_erc20").Add(1)
		case btypes.TokenTypeERC721:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_deposit_erc721").Add(1)
		case btypes.TokenTypeERC1155:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_deposit_erc1155").Add(1)
		}
	}

	for _, relayedMessage := range res.RelayedMessages {
		switch btypes.TxStatusType(relayedMessage.TxStatus) {
		case btypes.TxStatusTypeRelayed:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_relayed_message").Add(1)
		case btypes.TxStatusTypeFailedRelayed:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_failed_relayed_message").Add(1)
		}
		// Have not tracked L1 relayed message reverted transaction yet.
		// 1. need to parse calldata of tx.
		// 2. hard to track internal tx.
	}

	for _, batchEvent := range res.BatchEvents {
		switch btypes.BatchStatusType(batchEvent.BatchStatus) {
		case btypes.BatchStatusTypeCommitted:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_commit_batch_event").Add(1)
		case btypes.BatchStatusTypeReverted:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_revert_batch_event").Add(1)
		case btypes.BatchStatusTypeFinalized:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_finalize_batch_event").Add(1)
		}
	}

	for _, messageQueueEvent := range res.MessageQueueEvents {
		switch messageQueueEvent.EventType {
		case btypes.MessageQueueEventTypeQueueTransaction: // sendMessage is filtered out, only leaving replayMessage or appendEnforcedTransaction.
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_replay_message_or_enforced_transaction").Add(1)
		case btypes.MessageQueueEventTypeDequeueTransaction:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_skip_message").Add(1)
		case btypes.MessageQueueEventTypeDropTransaction:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_drop_message").Add(1)
		// one ResetDequeuedTransaction event could indicate reset multiple skipped messages,
		// this metric only counts the number of events, not the number of skipped messages.
		case btypes.MessageQueueEventTypeResetDequeuedTransaction:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_reset_skipped_messages").Add(1)
		}
	}

	for _, bridgeBatchDepositEvent := range res.BridgeBatchDepositEvents {
		switch btypes.TokenType(bridgeBatchDepositEvent.TokenType) {
		case btypes.TokenTypeETH:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_bridge_batch_deposit_eth").Add(1)
		case btypes.TokenTypeERC20:
			f.l1FetcherLogicFetchedTotal.WithLabelValues("L1_bridge_batch_deposit_erc20").Add(1)
		}
	}
}
