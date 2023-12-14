package fetcher

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	backendabi "scroll-tech/bridge-history-api/abi"
	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/logic"
	"scroll-tech/bridge-history-api/internal/orm"
	"scroll-tech/bridge-history-api/internal/utils"
)

// L2MessageFetcher fetches cross message events from L2 and saves them to database.
type L2MessageFetcher struct {
	ctx             context.Context
	cfg             *config.LayerConfig
	db              *gorm.DB
	crossMessageOrm *orm.CrossMessage
	client          *ethclient.Client
	addressList     []common.Address
	syncInfo        *SyncInfo
}

// NewL2MessageFetcher creates a new L2MessageFetcher instance.
func NewL2MessageFetcher(ctx context.Context, cfg *config.LayerConfig, db *gorm.DB, client *ethclient.Client, syncInfo *SyncInfo) (*L2MessageFetcher, error) {
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

	return &L2MessageFetcher{
		ctx:             ctx,
		cfg:             cfg,
		db:              db,
		crossMessageOrm: orm.NewCrossMessage(db),
		client:          client,
		addressList:     addressList,
		syncInfo:        syncInfo,
	}, nil
}

// Start starts the L2 message fetching process.
func (c *L2MessageFetcher) Start() {
	l2SentMessageSyncedHeight, err := c.crossMessageOrm.GetMessageSyncedHeightInDB(c.ctx, orm.MessageTypeL2SentMessage)
	if err != nil {
		log.Error("failed to get L2 cross message processed height", "err", err)
		return
	}
	c.syncInfo.SetL2ScanHeight(l2SentMessageSyncedHeight)
	log.Info("Start L2 message fetcher", "message synced height", l2SentMessageSyncedHeight)

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

func (c *L2MessageFetcher) fetchAndSaveEvents(confirmation uint64) {
	startHeight := c.syncInfo.GetL2ScanHeight() + 1
	endHeight, err := utils.GetBlockNumber(c.ctx, c.client, confirmation)
	if err != nil {
		log.Error("failed to get L1 safe block number", "err", err)
		return
	}
	log.Info("fetch and save missing L2 events", "start height", startHeight, "end height", endHeight)

	for from := startHeight; from <= endHeight; from += c.cfg.FetchLimit {
		to := from + c.cfg.FetchLimit - 1
		if to > endHeight {
			to = endHeight
		}
		err = c.doFetchAndSaveEvents(c.ctx, from, to, c.addressList)
		if err != nil {
			log.Error("failed to fetch and save L2 events", "from", from, "to", to, "err", err)
			return
		}
		c.syncInfo.SetL2ScanHeight(to)
	}
}

func (c *L2MessageFetcher) doFetchAndSaveEvents(ctx context.Context, from uint64, to uint64, addrList []common.Address) error {
	log.Info("fetch and save L2 events", "from", from, "to", to)
	var l2FailedGatewayRouterTxs []*orm.CrossMessage
	var l2RevertedRelayedMessages []*orm.CrossMessage
	blockTimestampsMap := make(map[uint64]uint64)
	blocks, err := utils.GetL2BlocksInRange(c.ctx, c.client, from, to)
	if err != nil {
		log.Error("failed to get L2 blocks in range", "from", from, "to", to, "err", err)
		return err
	}
	for i := from; i <= to; i++ {
		block := blocks[i-from]
		blockTimestampsMap[block.NumberU64()] = block.Time()

		for _, tx := range block.Transactions() {
			to := tx.To()
			if to == nil {
				continue
			}
			toAddress := to.String()
			if toAddress == c.cfg.GatewayRouterAddr {
				var receipt *types.Receipt
				receipt, err = c.client.TransactionReceipt(ctx, tx.Hash())
				if err != nil {
					log.Error("Failed to get transaction receipt", "txHash", tx.Hash().String(), "err", err)
					return err
				}

				// Check if the transaction failed
				if receipt.Status == types.ReceiptStatusFailed {
					signer := types.NewLondonSigner(new(big.Int).SetUint64(tx.ChainId().Uint64()))
					var sender common.Address
					sender, err = signer.Sender(tx)
					if err != nil {
						log.Error("get sender failed", "chain id", tx.ChainId().Uint64(), "tx hash", tx.Hash().String(), "err", err)
						return err
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
				var receipt *types.Receipt
				receipt, err = c.client.TransactionReceipt(ctx, tx.Hash())
				if err != nil {
					log.Error("Failed to get transaction receipt", "txHash", tx.Hash().String(), "err", err)
					return err
				}
				// Check if the transaction failed
				if receipt.Status == types.ReceiptStatusFailed {
					l2RevertedRelayedMessages = append(l2RevertedRelayedMessages, &orm.CrossMessage{
						MessageHash:   "0x" + common.Bytes2Hex(crypto.Keccak256(tx.AsL1MessageTx().Data)),
						L2TxHash:      tx.Hash().String(),
						TxStatus:      int(orm.TxStatusTypeRelayedTxReverted),
						L2BlockNumber: receipt.BlockNumber.Uint64(),
						MessageType:   int(orm.MessageTypeL1SentMessage),
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
	query.Topics[0] = make([]common.Hash, 7)
	query.Topics[0][0] = backendabi.L2WithdrawETHSig
	query.Topics[0][1] = backendabi.L2WithdrawERC20Sig
	query.Topics[0][2] = backendabi.L2WithdrawERC721Sig
	query.Topics[0][3] = backendabi.L2WithdrawERC1155Sig
	query.Topics[0][4] = backendabi.L2SentMessageEventSig
	query.Topics[0][5] = backendabi.L2RelayedMessageEventSig
	query.Topics[0][6] = backendabi.L2FailedRelayedMessageEventSig

	logs, err := c.client.FilterLogs(ctx, query)
	if err != nil {
		log.Error("Failed to filter L2 event logs", "from", from, "to", to, "err", err)
		return err
	}
	l2WithdrawMessages, l2RelayedMessages, err := logic.ParseL2EventLogs(logs, blockTimestampsMap)
	if err != nil {
		log.Error("failed to parse L2 event logs", "from", from, "to", to, "err", err)
		return err
	}

	if err = c.updateL2WithdrawMessageProofs(ctx, l2WithdrawMessages, to); err != nil {
		log.Error("failed to update withdraw message proofs", "err", err)
	}

	err = c.db.Transaction(func(tx *gorm.DB) error {
		if txErr := c.crossMessageOrm.InsertOrUpdateL2Messages(ctx, l2WithdrawMessages, tx); txErr != nil {
			log.Error("failed to insert L2 withdrawal messages", "from", from, "to", to, "err", txErr)
			return txErr
		}
		if txErr := c.crossMessageOrm.InsertOrUpdateL2RelayedMessagesOfL1Deposits(ctx, l2RelayedMessages, tx); txErr != nil {
			log.Error("failed to update L2 relayed messages of L1 deposits", "from", from, "to", to, "err", txErr)
			return txErr
		}
		if txErr := c.crossMessageOrm.InsertOrUpdateL2RevertedRelayedMessagesOfL1Deposits(ctx, l2RevertedRelayedMessages, tx); txErr != nil {
			log.Error("failed to update L2 relayed messages of L1 deposits", "from", from, "to", to, "err", txErr)
			return txErr
		}
		if txErr := c.crossMessageOrm.InsertFailedGatewayRouterTxs(ctx, l2FailedGatewayRouterTxs, tx); txErr != nil {
			log.Error("failed to insert L2 failed gateway router transactions", "from", from, "to", to, "err", txErr)
			return txErr
		}
		return nil
	})
	if err != nil {
		log.Error("failed to update db of L2 events", "from", from, "to", to, "err", err)
		return err
	}
	return nil
}

func (c *L2MessageFetcher) updateL2WithdrawMessageProofs(ctx context.Context, l2WithdrawMessages []*orm.CrossMessage, endBlock uint64) error {
	withdrawTrie := utils.NewWithdrawTrie()
	message, err := c.crossMessageOrm.GetLatestL2Withdrawal(ctx)
	if err != nil {
		log.Error("failed to get latest L2 sent message event", "err", err)
		return err
	}
	if message != nil {
		withdrawTrie.Initialize(message.MessageNonce, common.HexToHash(message.MessageHash), message.MerkleProof)
	}
	messageHashes := make([]common.Hash, len(l2WithdrawMessages))
	for i, message := range l2WithdrawMessages {
		messageHashes[i] = common.HexToHash(message.MessageHash)
	}
	proofs := withdrawTrie.AppendMessages(messageHashes)
	if len(l2WithdrawMessages) != len(proofs) {
		log.Error("invalid proof array length", "L2 withdrawal messages length", len(l2WithdrawMessages), "proofs length", len(proofs))
		return fmt.Errorf("invalid proof array length: got %d proofs for %d l2WithdrawMessages", len(proofs), len(l2WithdrawMessages))
	}
	for i, proof := range proofs {
		l2WithdrawMessages[i].MerkleProof = proof
	}
	// Verify if local info is correct.
	withdrawRoot, err := c.client.StorageAt(ctx, common.HexToAddress(c.cfg.MessageQueueAddr), common.Hash{}, new(big.Int).SetUint64(endBlock))
	if err != nil {
		log.Error("failed to get withdraw root", "number", endBlock, "error", err)
		return fmt.Errorf("failed to get withdraw root: %v, number: %v", err, endBlock)
	}
	if common.BytesToHash(withdrawRoot) != withdrawTrie.MessageRoot() {
		log.Error("withdraw root mismatch", "expected", common.BytesToHash(withdrawRoot).String(), "got", withdrawTrie.MessageRoot().String())
		return fmt.Errorf("withdraw root mismatch. expected: %v, got: %v", common.BytesToHash(withdrawRoot), withdrawTrie.MessageRoot())
	}
	return nil
}
