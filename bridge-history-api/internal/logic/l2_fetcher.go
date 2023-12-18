package logic

import (
	"context"
	"math/big"

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

// L2FilterResult the L2 filter result
type L2FilterResult struct {
	FailedGatewayRouterTxs []*orm.CrossMessage
	WithdrawMessages       []*orm.CrossMessage
	RelayedMessages        []*orm.CrossMessage
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

	return &L2FetcherLogic{
		db:              db,
		crossMessageOrm: orm.NewCrossMessage(db),
		batchEventOrm:   orm.NewBatchEvent(db),
		cfg:             cfg,
		client:          client,
		addressList:     addressList,
		parser:          NewL2EventParser(),
	}
}

func (f *L2FetcherLogic) gatewayRouterFailedTxs(ctx context.Context, from, to uint64) (map[uint64]uint64, []*orm.CrossMessage, []*orm.CrossMessage, error) {
	var l2FailedGatewayRouterTxs []*orm.CrossMessage
	var l2RevertedRelayedMessages []*orm.CrossMessage
	blockTimestampsMap := make(map[uint64]uint64)

	blocks, err := utils.GetL2BlocksInRange(ctx, f.client, from, to)
	if err != nil {
		log.Error("failed to get L2 blocks in range", "from", from, "to", to, "err", err)
		return nil, nil, nil, err
	}

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
					return nil, nil, nil, receiptErr
				}

				// Check if the transaction failed
				if receipt.Status == types.ReceiptStatusFailed {
					signer := types.LatestSignerForChainID(new(big.Int).SetUint64(tx.ChainId().Uint64()))
					sender, signerErr := signer.Sender(tx)
					if signerErr != nil {
						log.Error("get sender failed", "chain id", tx.ChainId().Uint64(), "tx hash", tx.Hash().String(), "err", signerErr)
						return nil, nil, nil, signerErr
					}

					l2FailedGatewayRouterTxs = append(l2FailedGatewayRouterTxs, &orm.CrossMessage{
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
					return nil, nil, nil, receiptErr
				}

				// Check if the transaction failed
				if receipt.Status == types.ReceiptStatusFailed {
					l2RevertedRelayedMessages = append(l2RevertedRelayedMessages, &orm.CrossMessage{
						MessageHash:   common.BytesToHash(crypto.Keccak256(tx.AsL1MessageTx().Data)).String(),
						L2TxHash:      tx.Hash().String(),
						TxStatus:      int(orm.TxStatusTypeRelayedTxReverted),
						L2BlockNumber: receipt.BlockNumber.Uint64(),
						MessageType:   int(orm.MessageTypeL1SentMessage),
					})
				}
			}
		}
	}
	return blockTimestampsMap, l2FailedGatewayRouterTxs, l2RevertedRelayedMessages, nil
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
func (f *L2FetcherLogic) L2Fetcher(ctx context.Context, from, to uint64) (*L2FilterResult, error) {
	log.Info("fetch and save L1 events", "from", from, "to", to)

	blockTimestampsMap, l2FailedGatewayRouterTxs, l2RevertedRelayedMessages, routerErr := f.gatewayRouterFailedTxs(ctx, from, to)
	if routerErr != nil {
		log.Error("L2Fetcher gatewayRouterFailedTxs failed", "from", from, "to", to, "error", routerErr)
		return nil, routerErr
	}

	eventLogs, err := f.l2FetcherLogs(ctx, from, to)
	if err != nil {
		log.Error("L2Fetcher l2FetcherLogs failed", "from", from, "to", to, "error", err)
		return nil, err
	}

	l2WithdrawMessages, l2RelayedMessages, err := f.parser.ParseL2EventLogs(eventLogs, blockTimestampsMap)
	if err != nil {
		log.Error("failed to parse L2 event logs", "from", from, "to", to, "err", err)
		return nil, err
	}

	res := L2FilterResult{
		FailedGatewayRouterTxs: l2FailedGatewayRouterTxs,
		WithdrawMessages:       l2WithdrawMessages,
		RelayedMessages:        append(l2RelayedMessages, l2RevertedRelayedMessages...),
	}
	return &res, nil
}
