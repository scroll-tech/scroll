package logic

import (
	"context"
	"math/big"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	backendabi "scroll-tech/bridge-history-api/abi"
	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/orm"
	"scroll-tech/bridge-history-api/internal/utils"
)

// L2ReorgSafeDepth represents the number of block confirmations considered safe against L2 chain reorganizations.
// Reorganizations at this depth under normal cases are extremely unlikely.
const L2ReorgSafeDepth = 256

// L2FilterResult the L2 filter result
type L2FilterResult struct {
	FailedGatewayRouterTransactions []*orm.CrossMessage
	WithdrawMessages                []*orm.CrossMessage
	RelayedMessages                 []*orm.CrossMessage
}

// L2FetcherLogic the L2 fetcher logic
type L2FetcherLogic struct {
	cfg             *config.LayerConfig
	client          *ethclient.Client
	addressList     []common.Address
	parser          *L2EventParser
	db              *gorm.DB
	crossMessageOrm *orm.CrossMessage
	batchEventOrm   *orm.BatchEvent

	l2FetcherLogicFetchedTotal *prometheus.CounterVec
}

// NewL2FetcherLogic create L2 fetcher logic
func NewL2FetcherLogic(cfg *config.LayerConfig, db *gorm.DB, client *ethclient.Client) *L2FetcherLogic {
	addressList := []common.Address{
		common.HexToAddress(cfg.ETHGatewayAddr),

		common.HexToAddress(cfg.StandardERC20GatewayAddr),
		common.HexToAddress(cfg.CustomERC20GatewayAddr),
		common.HexToAddress(cfg.WETHGatewayAddr),
		common.HexToAddress(cfg.DAIGatewayAddr),

		common.HexToAddress(cfg.ERC721GatewayAddr),
		common.HexToAddress(cfg.ERC1155GatewayAddr),

		common.HexToAddress(cfg.MessengerAddr),
	}

	// Optional erc20 gateways.
	if cfg.USDCGatewayAddr != "" {
		addressList = append(addressList, common.HexToAddress(cfg.USDCGatewayAddr))
	}

	if cfg.LIDOGatewayAddr != "" {
		addressList = append(addressList, common.HexToAddress(cfg.LIDOGatewayAddr))
	}

	f := &L2FetcherLogic{
		db:              db,
		crossMessageOrm: orm.NewCrossMessage(db),
		batchEventOrm:   orm.NewBatchEvent(db),
		cfg:             cfg,
		client:          client,
		addressList:     addressList,
		parser:          NewL2EventParser(),
	}

	reg := prometheus.DefaultRegisterer
	f.l2FetcherLogicFetchedTotal = promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
		Name: "L2 fetcher logic fetched total",
		Help: "The total number of events or failed txs fetched in L2 fetcher logic.",
	}, []string{"type"})

	return f
}

func (f *L2FetcherLogic) gatewayRouterFailedTransactions(ctx context.Context, from, to uint64, lastBlockHash common.Hash) (bool, uint64, map[uint64]uint64, []*orm.CrossMessage, []*orm.CrossMessage, error) {
	blocks, err := utils.GetL2BlocksInRange(ctx, f.client, from, to)
	if err != nil {
		log.Error("failed to get L2 blocks in range", "from", from, "to", to, "err", err)
		return false, 0, nil, nil, nil, err
	}

	for i, block := range blocks {
		if i == 0 && block.ParentHash() != lastBlockHash {
			log.Warn("L2 reorg detected", "reorg height", block.NumberU64(), "expected parent hash", block.ParentHash(), "local parent hash", lastBlockHash)
			return true, block.NumberU64(), nil, nil, nil, nil
		}
		if i != 0 && block.ParentHash() != blocks[i-1].ParentHash() {
			log.Warn("L2 reorg detected", "reorg height", block.NumberU64(), "expected parent hash", block.ParentHash(), "local parent hash", blocks[i-1].ParentHash())
			return true, block.NumberU64(), nil, nil, nil, nil
		}
	}

	var l2FailedGatewayRouterTransactions []*orm.CrossMessage
	var l2RevertedRelayedMessageTransactions []*orm.CrossMessage
	blockTimestampsMap := make(map[uint64]uint64)

	for i := from; i <= to; i++ {
		block := blocks[i-from]
		blockTimestampsMap[block.NumberU64()] = block.Time()

		for _, tx := range block.Transactions() {
			txTo := tx.To()
			if txTo == nil {
				continue
			}
			toAddress := txTo.String()

			if toAddress == f.cfg.GatewayRouterAddr {
				receipt, receiptErr := f.client.TransactionReceipt(ctx, tx.Hash())
				if receiptErr != nil {
					log.Error("Failed to get transaction receipt", "txHash", tx.Hash().String(), "err", receiptErr)
					return false, 0, nil, nil, nil, receiptErr
				}

				// Check if the transaction is failed
				if receipt.Status == types.ReceiptStatusFailed {
					signer := types.LatestSignerForChainID(new(big.Int).SetUint64(tx.ChainId().Uint64()))
					sender, signerErr := signer.Sender(tx)
					if signerErr != nil {
						log.Error("get sender failed", "chain id", tx.ChainId().Uint64(), "tx hash", tx.Hash().String(), "err", signerErr)
						return false, 0, nil, nil, nil, signerErr
					}

					l2FailedGatewayRouterTransactions = append(l2FailedGatewayRouterTransactions, &orm.CrossMessage{
						L2TxHash:       tx.Hash().String(),
						MessageType:    int(orm.MessageTypeL2SentMessage),
						Sender:         sender.String(),
						Receiver:       (*tx.To()).String(),
						L2BlockNumber:  receipt.BlockNumber.Uint64(),
						BlockTimestamp: block.Time(),
						TxStatus:       int(orm.TxStatusTypeSentFailed),
					})
				}
			}

			if tx.Type() == types.L1MessageTxType {
				receipt, receiptErr := f.client.TransactionReceipt(ctx, tx.Hash())
				if receiptErr != nil {
					log.Error("Failed to get transaction receipt", "txHash", tx.Hash().String(), "err", receiptErr)
					return false, 0, nil, nil, nil, receiptErr
				}

				// Check if the transaction is failed
				if receipt.Status == types.ReceiptStatusFailed {
					l2RevertedRelayedMessageTransactions = append(l2RevertedRelayedMessageTransactions, &orm.CrossMessage{
						MessageHash:   common.BytesToHash(crypto.Keccak256(tx.AsL1MessageTx().Data)).String(),
						L2TxHash:      tx.Hash().String(),
						TxStatus:      int(orm.TxStatusTypeRelayedTransactionReverted),
						L2BlockNumber: receipt.BlockNumber.Uint64(),
						MessageType:   int(orm.MessageTypeL1SentMessage),
					})
				}
			}
		}
	}
	return false, 0, blockTimestampsMap, l2FailedGatewayRouterTransactions, l2RevertedRelayedMessageTransactions, nil
}

func (f *L2FetcherLogic) l2FetcherLogs(ctx context.Context, from, to uint64) ([]types.Log, error) {
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from), // inclusive
		ToBlock:   new(big.Int).SetUint64(to),   // inclusive
		Addresses: f.addressList,
		Topics:    make([][]common.Hash, 1),
	}
	query.Topics[0] = make([]common.Hash, 7)
	query.Topics[0][0] = backendabi.L2WithdrawETHSig
	query.Topics[0][1] = backendabi.L2WithdrawERC20Sig
	query.Topics[0][2] = backendabi.L2WithdrawERC721Sig
	query.Topics[0][3] = backendabi.L2WithdrawERC1155Sig
	query.Topics[0][4] = backendabi.L2SentMessageEventSig
	query.Topics[0][5] = backendabi.L2RelayedMessageEventSig
	query.Topics[0][6] = backendabi.L2FailedRelayedMessageEventSig

	eventLogs, err := f.client.FilterLogs(ctx, query)
	if err != nil {
		log.Error("Failed to filter L2 event logs", "from", from, "to", to, "err", err)
		return nil, err
	}
	return eventLogs, nil
}

// L2Fetcher L2 fetcher
func (f *L2FetcherLogic) L2Fetcher(ctx context.Context, from, to uint64, lastBlockHash common.Hash) (bool, uint64, *L2FilterResult, error) {
	log.Info("fetch and save L2 events", "from", from, "to", to)

	isReorg, reorgHeight, blockTimestampsMap, l2FailedGatewayRouterTransactions, l2RevertedRelayedMessageTransactions, routerErr := f.gatewayRouterFailedTransactions(ctx, from, to, lastBlockHash)
	if routerErr != nil {
		log.Error("L2Fetcher gatewayRouterFailedTransactions failed", "from", from, "to", to, "error", routerErr)
		return false, 0, nil, routerErr
	}

	if isReorg {
		var resyncHeight uint64
		if reorgHeight > L2ReorgSafeDepth {
			resyncHeight = reorgHeight - L2ReorgSafeDepth
		}
		return true, resyncHeight, nil, nil
	}

	eventLogs, err := f.l2FetcherLogs(ctx, from, to)
	if err != nil {
		log.Error("L2Fetcher l2FetcherLogs failed", "from", from, "to", to, "error", err)
		return false, 0, nil, err
	}

	l2WithdrawMessages, l2RelayedMessages, err := f.parser.ParseL2EventLogs(eventLogs, blockTimestampsMap)
	if err != nil {
		log.Error("failed to parse L2 event logs", "from", from, "to", to, "err", err)
		return false, 0, nil, err
	}

	res := L2FilterResult{
		FailedGatewayRouterTransactions: l2FailedGatewayRouterTransactions,
		WithdrawMessages:                l2WithdrawMessages,
		RelayedMessages:                 append(l2RelayedMessages, l2RevertedRelayedMessageTransactions...),
	}

	f.updateMetrics(res)

	return false, 0, &res, nil
}

func (f *L2FetcherLogic) updateMetrics(res L2FilterResult) {
	f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_failed_gateway_router_transaction").Add(float64(len(res.FailedGatewayRouterTransactions)))

	for _, withdrawMessage := range res.WithdrawMessages {
		switch orm.TokenType(withdrawMessage.TokenType) {
		case orm.TokenTypeETH:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_withdraw_eth").Add(1)
		case orm.TokenTypeERC20:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_withdraw_erc20").Add(1)
		case orm.TokenTypeERC721:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_withdraw_erc721").Add(1)
		case orm.TokenTypeERC1155:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_withdraw_erc1155").Add(1)
		}
	}

	for _, relayedMessage := range res.RelayedMessages {
		switch orm.TxStatusType(relayedMessage.TxStatus) {
		case orm.TxStatusTypeRelayed:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_relayed_message").Add(1)
		case orm.TxStatusTypeFailedRelayed:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_failed_relayed_message").Add(1)
		case orm.TxStatusTypeRelayedTransactionReverted:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_relayed_message_transaction_reverted").Add(1)
		}
	}
}
