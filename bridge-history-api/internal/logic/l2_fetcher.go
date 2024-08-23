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
	btypes "scroll-tech/bridge-history-api/internal/types"
	"scroll-tech/bridge-history-api/internal/utils"
)

// L2ReorgSafeDepth represents the number of block confirmations considered safe against L2 chain reorganizations.
// Reorganizations at this depth under normal cases are extremely unlikely.
const L2ReorgSafeDepth = 256

// L2FilterResult the L2 filter result
type L2FilterResult struct {
	WithdrawMessages          []*orm.CrossMessage
	RelayedMessages           []*orm.CrossMessage // relayed, failed relayed, relay tx reverted.
	OtherRevertedTxs          []*orm.CrossMessage // reverted txs except relay tx reverted.
	BridgeBatchDepositMessage []*orm.BridgeBatchDepositEvent
}

// L2FetcherLogic the L2 fetcher logic
type L2FetcherLogic struct {
	cfg             *config.FetcherConfig
	client          *ethclient.Client
	addressList     []common.Address
	gatewayList     []common.Address
	parser          *L2EventParser
	db              *gorm.DB
	crossMessageOrm *orm.CrossMessage
	batchEventOrm   *orm.BatchEvent

	l2FetcherLogicFetchedTotal *prometheus.CounterVec
}

// NewL2FetcherLogic create L2 fetcher logic
func NewL2FetcherLogic(cfg *config.FetcherConfig, db *gorm.DB, client *ethclient.Client) *L2FetcherLogic {
	addressList := []common.Address{
		common.HexToAddress(cfg.ETHGatewayAddr),

		common.HexToAddress(cfg.StandardERC20GatewayAddr),
		common.HexToAddress(cfg.CustomERC20GatewayAddr),
		common.HexToAddress(cfg.DAIGatewayAddr),

		common.HexToAddress(cfg.ERC721GatewayAddr),
		common.HexToAddress(cfg.ERC1155GatewayAddr),

		common.HexToAddress(cfg.MessengerAddr),
	}

	gatewayList := []common.Address{
		common.HexToAddress(cfg.ETHGatewayAddr),

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

	if common.HexToAddress(cfg.WETHGatewayAddr) != (common.Address{}) {
		addressList = append(addressList, common.HexToAddress(cfg.WETHGatewayAddr))
		gatewayList = append(gatewayList, common.HexToAddress(cfg.WETHGatewayAddr))
	}

	log.Info("L2 Fetcher configured with the following address list", "addresses", addressList, "gateways", gatewayList)

	f := &L2FetcherLogic{
		db:              db,
		crossMessageOrm: orm.NewCrossMessage(db),
		batchEventOrm:   orm.NewBatchEvent(db),
		cfg:             cfg,
		client:          client,
		addressList:     addressList,
		gatewayList:     gatewayList,
		parser:          NewL2EventParser(cfg, client),
	}

	reg := prometheus.DefaultRegisterer
	f.l2FetcherLogicFetchedTotal = promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
		Name: "L2_fetcher_logic_fetched_total",
		Help: "The total number of events or failed txs fetched in L2 fetcher logic.",
	}, []string{"type"})

	return f
}

func (f *L2FetcherLogic) getBlocksAndDetectReorg(ctx context.Context, from, to uint64, lastBlockHash common.Hash) (bool, uint64, common.Hash, []*types.Block, error) {
	blocks, err := utils.GetBlocksInRange(ctx, f.client, from, to)
	if err != nil {
		log.Error("failed to get L2 blocks in range", "from", from, "to", to, "err", err)
		return false, 0, common.Hash{}, nil, err
	}

	for _, block := range blocks {
		if block.ParentHash() != lastBlockHash {
			log.Warn("L2 reorg detected", "reorg height", block.NumberU64()-1, "expected hash", block.ParentHash().String(), "local hash", lastBlockHash.String())
			var resyncHeight uint64
			if block.NumberU64() > L2ReorgSafeDepth+1 {
				resyncHeight = block.NumberU64() - L2ReorgSafeDepth - 1
			}
			header, err := f.client.HeaderByNumber(ctx, new(big.Int).SetUint64(resyncHeight))
			if err != nil {
				log.Error("failed to get L2 header by number", "block number", resyncHeight, "err", err)
				return false, 0, common.Hash{}, nil, err
			}
			return true, resyncHeight, header.Hash(), nil, nil
		}
		lastBlockHash = block.Hash()
	}

	return false, 0, lastBlockHash, blocks, nil
}

func (f *L2FetcherLogic) getRevertedTxs(ctx context.Context, from, to uint64, blocks []*types.Block) (map[uint64]uint64, []*orm.CrossMessage, []*orm.CrossMessage, error) {
	var l2RevertedUserTxs []*orm.CrossMessage
	var l2RevertedRelayedMessageTxs []*orm.CrossMessage
	blockTimestampsMap := make(map[uint64]uint64)

	for i := from; i <= to; i++ {
		block := blocks[i-from]
		blockTimestampsMap[block.NumberU64()] = block.Time()

		for _, tx := range block.Transactions() {
			if tx.IsL1MessageTx() {
				receipt, receiptErr := f.client.TransactionReceipt(ctx, tx.Hash())
				if receiptErr != nil {
					log.Error("Failed to get transaction receipt", "txHash", tx.Hash().String(), "err", receiptErr)
					return nil, nil, nil, receiptErr
				}

				// Check if the transaction is failed
				if receipt.Status == types.ReceiptStatusFailed {
					l2RevertedRelayedMessageTxs = append(l2RevertedRelayedMessageTxs, &orm.CrossMessage{
						MessageHash:   common.BytesToHash(crypto.Keccak256(tx.AsL1MessageTx().Data)).String(),
						L2TxHash:      tx.Hash().String(),
						TxStatus:      int(btypes.TxStatusTypeRelayTxReverted),
						L2BlockNumber: receipt.BlockNumber.Uint64(),
						MessageType:   int(btypes.MessageTypeL1SentMessage),
					})
				}
				continue
			}

			// Gateways: L2 withdrawal.
			if !isTransactionToGateway(tx, f.gatewayList) {
				continue
			}

			receipt, receiptErr := f.client.TransactionReceipt(ctx, tx.Hash())
			if receiptErr != nil {
				log.Error("Failed to get transaction receipt", "txHash", tx.Hash().String(), "err", receiptErr)
				return nil, nil, nil, receiptErr
			}

			// Check if the transaction is failed
			if receipt.Status == types.ReceiptStatusFailed {
				signer := types.LatestSignerForChainID(new(big.Int).SetUint64(tx.ChainId().Uint64()))
				sender, signerErr := signer.Sender(tx)
				if signerErr != nil {
					log.Error("get sender failed", "chain id", tx.ChainId().Uint64(), "tx hash", tx.Hash().String(), "err", signerErr)
					return nil, nil, nil, signerErr
				}

				l2RevertedUserTxs = append(l2RevertedUserTxs, &orm.CrossMessage{
					L2TxHash:       tx.Hash().String(),
					MessageType:    int(btypes.MessageTypeL2SentMessage),
					Sender:         sender.String(),
					Receiver:       (*tx.To()).String(),
					L2BlockNumber:  receipt.BlockNumber.Uint64(),
					BlockTimestamp: block.Time(),
					TxStatus:       int(btypes.TxStatusTypeSentTxReverted),
				})
			}
		}
	}
	return blockTimestampsMap, l2RevertedUserTxs, l2RevertedRelayedMessageTxs, nil
}

func (f *L2FetcherLogic) l2FetcherLogs(ctx context.Context, from, to uint64) ([]types.Log, error) {
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from), // inclusive
		ToBlock:   new(big.Int).SetUint64(to),   // inclusive
		Addresses: f.addressList,
		Topics:    make([][]common.Hash, 1),
	}
	query.Topics[0] = make([]common.Hash, 9)
	query.Topics[0][0] = backendabi.L2WithdrawETHSig
	query.Topics[0][1] = backendabi.L2WithdrawERC20Sig
	query.Topics[0][2] = backendabi.L2WithdrawERC721Sig
	query.Topics[0][3] = backendabi.L2WithdrawERC1155Sig
	query.Topics[0][4] = backendabi.L2SentMessageEventSig
	query.Topics[0][5] = backendabi.L2RelayedMessageEventSig
	query.Topics[0][6] = backendabi.L2FailedRelayedMessageEventSig
	query.Topics[0][7] = backendabi.L2BridgeBatchDistributeSig
	query.Topics[0][8] = backendabi.L2BridgeBatchDistributeFailedSig

	eventLogs, err := f.client.FilterLogs(ctx, query)
	if err != nil {
		log.Error("Failed to filter L2 event logs", "from", from, "to", to, "err", err)
		return nil, err
	}
	return eventLogs, nil
}

// L2Fetcher L2 fetcher
func (f *L2FetcherLogic) L2Fetcher(ctx context.Context, from, to uint64, lastBlockHash common.Hash) (bool, uint64, common.Hash, *L2FilterResult, error) {
	log.Info("fetch and save L2 events", "from", from, "to", to)

	isReorg, reorgHeight, blockHash, blocks, getErr := f.getBlocksAndDetectReorg(ctx, from, to, lastBlockHash)
	if getErr != nil {
		log.Error("L2Fetcher getBlocksAndDetectReorg failed", "from", from, "to", to, "error", getErr)
		return false, 0, common.Hash{}, nil, getErr
	}

	if isReorg {
		return isReorg, reorgHeight, blockHash, nil, nil
	}

	blockTimestampsMap, revertedUserTxs, revertedRelayMsgs, routerErr := f.getRevertedTxs(ctx, from, to, blocks)
	if routerErr != nil {
		log.Error("L2Fetcher getRevertedTxs failed", "from", from, "to", to, "error", routerErr)
		return false, 0, common.Hash{}, nil, routerErr
	}

	eventLogs, err := f.l2FetcherLogs(ctx, from, to)
	if err != nil {
		log.Error("L2Fetcher l2FetcherLogs failed", "from", from, "to", to, "error", err)
		return false, 0, common.Hash{}, nil, err
	}

	l2WithdrawMessages, l2RelayedMessages, l2BridgeBatchDepositMessages, err := f.parser.ParseL2EventLogs(ctx, eventLogs, blockTimestampsMap)
	if err != nil {
		log.Error("failed to parse L2 event logs", "from", from, "to", to, "err", err)
		return false, 0, common.Hash{}, nil, err
	}

	res := L2FilterResult{
		WithdrawMessages:          l2WithdrawMessages,
		RelayedMessages:           append(l2RelayedMessages, revertedRelayMsgs...),
		OtherRevertedTxs:          revertedUserTxs,
		BridgeBatchDepositMessage: l2BridgeBatchDepositMessages,
	}

	f.updateMetrics(res)

	return false, 0, blockHash, &res, nil
}

func (f *L2FetcherLogic) updateMetrics(res L2FilterResult) {
	f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_failed_gateway_router_transaction").Add(float64(len(res.OtherRevertedTxs)))

	for _, withdrawMessage := range res.WithdrawMessages {
		switch btypes.TokenType(withdrawMessage.TokenType) {
		case btypes.TokenTypeETH:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_withdraw_eth").Add(1)
		case btypes.TokenTypeERC20:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_withdraw_erc20").Add(1)
		case btypes.TokenTypeERC721:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_withdraw_erc721").Add(1)
		case btypes.TokenTypeERC1155:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_withdraw_erc1155").Add(1)
		}
	}

	for _, relayedMessage := range res.RelayedMessages {
		switch btypes.TxStatusType(relayedMessage.TxStatus) {
		case btypes.TxStatusTypeRelayed:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_relayed_message").Add(1)
		case btypes.TxStatusTypeFailedRelayed:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_failed_relayed_message").Add(1)
		case btypes.TxStatusTypeRelayTxReverted:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_reverted_relayed_message_transaction").Add(1)
		}
	}

	for _, bridgeBatchDepositMessage := range res.BridgeBatchDepositMessage {
		switch btypes.TxStatusType(bridgeBatchDepositMessage.TxStatus) {
		case btypes.TxStatusBridgeBatchDistribute:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_bridge_batch_distribute_message").Add(1)
		case btypes.TxStatusBridgeBatchDistributeFailed:
			f.l2FetcherLogicFetchedTotal.WithLabelValues("L2_bridge_batch_distribute_failed_message").Add(1)
		}
	}
}

func isTransactionToGateway(tx *types.Transaction, gatewayList []common.Address) bool {
	if tx.To() == nil {
		return false
	}
	for _, gateway := range gatewayList {
		if *tx.To() == gateway {
			return true
		}
	}
	return false
}
