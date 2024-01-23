package logic

import (
	"context"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	backendabi "scroll-tech/bridge-history-api/abi"
	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/orm"
	"scroll-tech/bridge-history-api/internal/utils"
)

// L1EventParser the l1 event parser
type L1EventParser struct {
	cfg    *config.FetcherConfig
	client *ethclient.Client
}

// NewL1EventParser creates l1 event parser
func NewL1EventParser(cfg *config.FetcherConfig, client *ethclient.Client) *L1EventParser {
	return &L1EventParser{
		cfg:    cfg,
		client: client,
	}
}

// ParseL1CrossChainEventLogs parses L1 watched cross chain events.
func (e *L1EventParser) ParseL1CrossChainEventLogs(ctx context.Context, logs []types.Log, blockTimestampsMap map[uint64]uint64) ([]*orm.CrossMessage, []*orm.CrossMessage, error) {
	var l1DepositMessages []*orm.CrossMessage
	var l1RelayedMessages []*orm.CrossMessage
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L1DepositETHSig:
			event := backendabi.ETHMessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ETHGatewayABI, &event, "DepositETH", vlog); err != nil {
				log.Error("Failed to unpack DepositETH event", "err", err)
				return nil, nil, err
			}
			lastMessage := l1DepositMessages[len(l1DepositMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeETH)
			lastMessage.TokenAmounts = event.Amount.String()
		case backendabi.L1DepositERC20Sig:
			event := backendabi.ERC20MessageEvent{}
			err := utils.UnpackLog(backendabi.IL1ERC20GatewayABI, &event, "DepositERC20", vlog)
			if err != nil {
				log.Error("Failed to unpack DepositERC20 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l1DepositMessages[len(l1DepositMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeERC20)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenAmounts = event.Amount.String()
		case backendabi.L1DepositERC721Sig:
			event := backendabi.ERC721MessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ERC721GatewayABI, &event, "DepositERC721", vlog); err != nil {
				log.Error("Failed to unpack DepositERC721 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l1DepositMessages[len(l1DepositMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeERC721)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = event.TokenID.String()
		case backendabi.L1BatchDepositERC721Sig:
			event := backendabi.BatchERC721MessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ERC721GatewayABI, &event, "BatchDepositERC721", vlog); err != nil {
				log.Error("Failed to unpack BatchDepositERC721 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l1DepositMessages[len(l1DepositMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeERC721)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = utils.ConvertBigIntArrayToString(event.TokenIDs)
		case backendabi.L1DepositERC1155Sig:
			event := backendabi.ERC1155MessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ERC1155GatewayABI, &event, "DepositERC1155", vlog); err != nil {
				log.Error("Failed to unpack DepositERC1155 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l1DepositMessages[len(l1DepositMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeERC1155)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = event.TokenID.String()
			lastMessage.TokenAmounts = event.Amount.String()
		case backendabi.L1BatchDepositERC1155Sig:
			event := backendabi.BatchERC1155MessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ERC1155GatewayABI, &event, "BatchDepositERC1155", vlog); err != nil {
				log.Error("Failed to unpack BatchDepositERC1155 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l1DepositMessages[len(l1DepositMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeERC1155)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = utils.ConvertBigIntArrayToString(event.TokenIDs)
			lastMessage.TokenAmounts = utils.ConvertBigIntArrayToString(event.TokenAmounts)
		case backendabi.L1SentMessageEventSig:
			event := backendabi.L1SentMessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ScrollMessengerABI, &event, "SentMessage", vlog); err != nil {
				log.Error("Failed to unpack SentMessage event", "err", err)
				return nil, nil, err
			}
			from := event.Sender.String()
			if from == e.cfg.GatewayRouterAddr {
				tx, isPending, rpcErr := e.client.TransactionByHash(ctx, vlog.TxHash)
				if rpcErr != nil || isPending {
					log.Error("Failed to get tx or the tx is still pending", "rpcErr", rpcErr, "isPending", isPending)
					return nil, nil, rpcErr
				}
				// EOA -> multisig -> gateway router.
				if tx.To() != nil {
					from = (*tx.To()).String()
				}
			}
			l1DepositMessages = append(l1DepositMessages, &orm.CrossMessage{
				L1BlockNumber:  vlog.BlockNumber,
				Sender:         from,
				Receiver:       event.Target.String(),
				TokenType:      int(orm.TokenTypeETH),
				L1TxHash:       vlog.TxHash.String(),
				TokenAmounts:   event.Value.String(),
				MessageNonce:   event.MessageNonce.Uint64(),
				MessageType:    int(orm.MessageTypeL1SentMessage),
				TxStatus:       int(orm.TxStatusTypeSent),
				BlockTimestamp: blockTimestampsMap[vlog.BlockNumber],
				MessageHash:    utils.ComputeMessageHash(event.Sender, event.Target, event.Value, event.MessageNonce, event.Message).String(),
			})
		case backendabi.L1RelayedMessageEventSig:
			event := backendabi.L1RelayedMessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ScrollMessengerABI, &event, "RelayedMessage", vlog); err != nil {
				log.Error("Failed to unpack RelayedMessage event", "err", err)
				return nil, nil, err
			}
			l1RelayedMessages = append(l1RelayedMessages, &orm.CrossMessage{
				MessageHash:   event.MessageHash.String(),
				L1BlockNumber: vlog.BlockNumber,
				L1TxHash:      vlog.TxHash.String(),
				TxStatus:      int(orm.TxStatusTypeRelayed),
				MessageType:   int(orm.MessageTypeL2SentMessage),
			})
		case backendabi.L1FailedRelayedMessageEventSig:
			event := backendabi.L1FailedRelayedMessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ScrollMessengerABI, &event, "FailedRelayedMessage", vlog); err != nil {
				log.Error("Failed to unpack FailedRelayedMessage event", "err", err)
				return nil, nil, err
			}
			l1RelayedMessages = append(l1RelayedMessages, &orm.CrossMessage{
				MessageHash:   event.MessageHash.String(),
				L1BlockNumber: vlog.BlockNumber,
				L1TxHash:      vlog.TxHash.String(),
				TxStatus:      int(orm.TxStatusTypeFailedRelayed),
				MessageType:   int(orm.MessageTypeL2SentMessage),
			})
		}
	}
	return l1DepositMessages, l1RelayedMessages, nil
}

// ParseL1BatchEventLogs parses L1 watched batch events.
func (e *L1EventParser) ParseL1BatchEventLogs(ctx context.Context, logs []types.Log, client *ethclient.Client) ([]*orm.BatchEvent, error) {
	var l1BatchEvents []*orm.BatchEvent
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L1CommitBatchEventSig:
			event := backendabi.L1CommitBatchEvent{}
			if err := utils.UnpackLog(backendabi.IScrollChainABI, &event, "CommitBatch", vlog); err != nil {
				log.Error("Failed to unpack CommitBatch event", "err", err)
				return nil, err
			}
			commitTx, isPending, err := client.TransactionByHash(ctx, vlog.TxHash)
			if err != nil || isPending {
				log.Error("Failed to get commit batch tx or the tx is still pending", "err", err, "isPending", isPending)
				return nil, err
			}
			startBlock, endBlock, err := utils.GetBatchRangeFromCalldata(commitTx.Data())
			if err != nil {
				log.Error("Failed to get batch range from calldata", "hash", commitTx.Hash().String(), "height", vlog.BlockNumber)
				return nil, err
			}
			l1BatchEvents = append(l1BatchEvents, &orm.BatchEvent{
				BatchStatus:      int(orm.BatchStatusTypeCommitted),
				BatchIndex:       event.BatchIndex.Uint64(),
				BatchHash:        event.BatchHash.String(),
				StartBlockNumber: startBlock,
				EndBlockNumber:   endBlock,
				L1BlockNumber:    vlog.BlockNumber,
			})
		case backendabi.L1RevertBatchEventSig:
			event := backendabi.L1RevertBatchEvent{}
			if err := utils.UnpackLog(backendabi.IScrollChainABI, &event, "RevertBatch", vlog); err != nil {
				log.Error("Failed to unpack RevertBatch event", "err", err)
				return nil, err
			}
			l1BatchEvents = append(l1BatchEvents, &orm.BatchEvent{
				BatchStatus:   int(orm.BatchStatusTypeReverted),
				BatchIndex:    event.BatchIndex.Uint64(),
				BatchHash:     event.BatchHash.String(),
				L1BlockNumber: vlog.BlockNumber,
			})
		case backendabi.L1FinalizeBatchEventSig:
			event := backendabi.L1FinalizeBatchEvent{}
			if err := utils.UnpackLog(backendabi.IScrollChainABI, &event, "FinalizeBatch", vlog); err != nil {
				log.Error("Failed to unpack FinalizeBatch event", "err", err)
				return nil, err
			}
			l1BatchEvents = append(l1BatchEvents, &orm.BatchEvent{
				BatchStatus:   int(orm.BatchStatusTypeFinalized),
				BatchIndex:    event.BatchIndex.Uint64(),
				BatchHash:     event.BatchHash.String(),
				L1BlockNumber: vlog.BlockNumber,
			})
		}
	}
	return l1BatchEvents, nil
}

// ParseL1MessageQueueEventLogs parses L1 watched message queue events.
func (e *L1EventParser) ParseL1MessageQueueEventLogs(logs []types.Log, l1DepositMessages []*orm.CrossMessage) ([]*orm.MessageQueueEvent, error) {
	messageHashes := make(map[common.Hash]struct{})
	for _, msg := range l1DepositMessages {
		messageHashes[common.HexToHash(msg.MessageHash)] = struct{}{}
	}

	var l1MessageQueueEvents []*orm.MessageQueueEvent
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L1QueueTransactionEventSig:
			event := backendabi.L1QueueTransactionEvent{}
			if err := utils.UnpackLog(backendabi.IL1MessageQueueABI, &event, "QueueTransaction", vlog); err != nil {
				log.Error("Failed to unpack QueueTransaction event", "err", err)
				return nil, err
			}
			messageHash := common.BytesToHash(crypto.Keccak256(event.Data))
			// If the message hash is not found in the map, it's not a replayMessage or enforced tx (omitted); add it to the events.
			if _, exists := messageHashes[messageHash]; !exists {
				l1MessageQueueEvents = append(l1MessageQueueEvents, &orm.MessageQueueEvent{
					EventType:   orm.MessageQueueEventTypeQueueTransaction,
					QueueIndex:  event.QueueIndex,
					MessageHash: messageHash,
					TxHash:      vlog.TxHash,
				})
			}
		case backendabi.L1DequeueTransactionEventSig:
			event := backendabi.L1DequeueTransactionEvent{}
			if err := utils.UnpackLog(backendabi.IL1MessageQueueABI, &event, "DequeueTransaction", vlog); err != nil {
				log.Error("Failed to unpack DequeueTransaction event", "err", err)
				return nil, err
			}
			skippedIndices := utils.GetSkippedQueueIndices(event.StartIndex.Uint64(), event.SkippedBitmap)
			for _, index := range skippedIndices {
				l1MessageQueueEvents = append(l1MessageQueueEvents, &orm.MessageQueueEvent{
					EventType:  orm.MessageQueueEventTypeDequeueTransaction,
					QueueIndex: index,
				})
			}
		case backendabi.L1DropTransactionEventSig:
			event := backendabi.L1DropTransactionEvent{}
			if err := utils.UnpackLog(backendabi.IL1MessageQueueABI, &event, "DropTransaction", vlog); err != nil {
				log.Error("Failed to unpack DropTransaction event", "err", err)
				return nil, err
			}
			l1MessageQueueEvents = append(l1MessageQueueEvents, &orm.MessageQueueEvent{
				EventType:  orm.MessageQueueEventTypeDropTransaction,
				QueueIndex: event.Index.Uint64(),
				TxHash:     vlog.TxHash,
			})
		}
	}
	return l1MessageQueueEvents, nil
}
