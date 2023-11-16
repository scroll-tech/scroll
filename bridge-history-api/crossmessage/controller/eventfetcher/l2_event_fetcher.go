package eventfetcher

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
	"gorm.io/gorm"

	backendabi "bridge-history-api/abi"
	"bridge-history-api/config"
	"bridge-history-api/crossmessage/controller/messageproof"
	"bridge-history-api/crossmessage/logic"
	"bridge-history-api/orm"
	"bridge-history-api/utils"
)

// L2MessageFetcher fetches cross message events from L2 and saves them to database.
type L2MessageFetcher struct {
	ctx             context.Context
	cfg             *config.LayerConfig
	db              *gorm.DB
	crossMessageOrm *orm.CrossMessage
	client          *ethclient.Client
	addressList     []common.Address
}

// NewL2MessageFetcher creates a new L2MessageFetcher instance.
func NewL2MessageFetcher(ctx context.Context, cfg *config.LayerConfig, db *gorm.DB, client *ethclient.Client) (*L2MessageFetcher, error) {
	addressList := []common.Address{
		common.HexToAddress(cfg.ETHGatewayAddr),

		common.HexToAddress(cfg.StandardERC20Gateway),
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
	}, nil
}

// Start starts the L2 message fetching process.
func (c *L2MessageFetcher) Start() {
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
	endHeight, err := utils.GetBlockNumber(c.ctx, c.client, confirmation)
	if err != nil {
		log.Error("failed to get L1 safe block number", "err", err)
		return
	}

	l2SentMessageProcessedHeight, err := c.crossMessageOrm.GetMessageProcessedHeightInDB(c.ctx, orm.MessageTypeL2SentMessage)
	if err != nil {
		log.Error("failed to get L2 cross message processed height", "err", err)
		return
	}
	startHeight := c.cfg.StartHeight
	if l2SentMessageProcessedHeight+1 > startHeight {
		startHeight = l2SentMessageProcessedHeight + 1
	}
	log.Info("fetch and save missing L2 events", "start height", startHeight, "config height", c.cfg.StartHeight, "db height", l2SentMessageProcessedHeight)

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
	}
}

func (c *L2MessageFetcher) doFetchAndSaveEvents(ctx context.Context, from uint64, to uint64, addrList []common.Address) error {
	log.Info("fetch and save L2 events", "from", from, "to", to)
	var l2FailedGatewayRouterTxs []*orm.CrossMessage
	blockTimestampsMap := make(map[uint64]uint64)
	for number := from; number <= to; number++ {
		blockNumber := new(big.Int).SetUint64(number)
		block, err := c.client.GetBlockByNumberOrHash(ctx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(number)))
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
					log.Error("Failed to get transaction receipt", "txHash", tx.Hash(), "err", err)
					return err
				}

				signer := types.NewLondonSigner(new(big.Int).SetUint64(c.cfg.ChainID))
				sender, err := signer.Sender(tx)
				if err != nil {
					log.Error("get sender failed", "chain id", c.cfg.ChainID, "tx hash", tx.Hash().String(), "err", err)
					return err
				}

				// Check if the transaction failed
				if receipt.Status == types.ReceiptStatusFailed {
					l2FailedGatewayRouterTxs = append(l2FailedGatewayRouterTxs, &orm.CrossMessage{
						L2TxHash:       tx.Hash().String(),
						MessageType:    int(orm.MessageTypeL2SentMessage),
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

	if err = c.updateL2WithdrawMessageProofs(ctx, l2WithdrawMessages); err != nil {
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
		if txErr := c.crossMessageOrm.InsertFailedMessages(ctx, l2FailedGatewayRouterTxs, tx); txErr != nil {
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

func (c *L2MessageFetcher) updateL2WithdrawMessageProofs(ctx context.Context, l2WithdrawMessages []*orm.CrossMessage) error {
	withdrawTrie := messageproof.NewWithdrawTrie()
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
	return nil
}
