package logic

import (
	"context"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	backendabi "scroll-tech/bridge-history-api/abi"
	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/orm"
	btypes "scroll-tech/bridge-history-api/internal/types"
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

// ParseL1CrossChainEventLogs parse l1 cross chain event logs
func (e *L1EventParser) ParseL1CrossChainEventLogs(ctx context.Context, logs []types.Log, blockTimestampsMap map[uint64]uint64) ([]*orm.CrossMessage, []*orm.CrossMessage, []*orm.BridgeBatchDepositEvent, error) {
	l1CrossChainDepositMessages, l1CrossChainRelayedMessages, err := e.ParseL1SingleCrossChainEventLogs(ctx, logs, blockTimestampsMap)
	if err != nil {
		return nil, nil, nil, err
	}

	l1BridgeBatchDepositMessages, err := e.ParseL1BridgeBatchDepositCrossChainEventLogs(logs, blockTimestampsMap)
	if err != nil {
		return nil, nil, nil, err
	}

	return l1CrossChainDepositMessages, l1CrossChainRelayedMessages, l1BridgeBatchDepositMessages, nil
}

// ParseL1BridgeBatchDepositCrossChainEventLogs parse L1 watched batch bridge cross chain events.
func (e *L1EventParser) ParseL1BridgeBatchDepositCrossChainEventLogs(logs []types.Log, blockTimestampsMap map[uint64]uint64) ([]*orm.BridgeBatchDepositEvent, error) {
	var l1BridgeBatchDepositMessages []*orm.BridgeBatchDepositEvent
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L1BridgeBatchDepositSig:
			event := backendabi.L1BatchBridgeGatewayDeposit{}
			if err := utils.UnpackLog(backendabi.L1BatchBridgeGatewayABI, &event, "Deposit", vlog); err != nil {
				log.Error("Failed to unpack batch bridge gateway deposit event", "err", err)
				return nil, err
			}

			var tokenType btypes.TokenType
			if event.Token == common.HexToAddress("0") {
				tokenType = btypes.TokenTypeETH
			} else {
				tokenType = btypes.TokenTypeERC20
			}

			l1BridgeBatchDepositMessages = append(l1BridgeBatchDepositMessages, &orm.BridgeBatchDepositEvent{
				TokenType:      int(tokenType),
				Sender:         event.Sender.String(),
				BatchIndex:     event.BatchIndex.Uint64(),
				TokenAmount:    event.Amount.String(),
				Fee:            event.Fee.String(),
				L1TokenAddress: event.Token.String(),
				L1BlockNumber:  vlog.BlockNumber,
				L1TxHash:       vlog.TxHash.String(),
				TxStatus:       int(btypes.TxStatusBridgeBatchDeposit),
				BlockTimestamp: blockTimestampsMap[vlog.BlockNumber],
				L1LogIndex:     vlog.Index,
			})
		}
	}
	return l1BridgeBatchDepositMessages, nil
}

// ParseL1SingleCrossChainEventLogs parses L1 watched single cross chain events.
func (e *L1EventParser) ParseL1SingleCrossChainEventLogs(ctx context.Context, logs []types.Log, blockTimestampsMap map[uint64]uint64) ([]*orm.CrossMessage, []*orm.CrossMessage, error) {
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
			lastMessage.TokenType = int(btypes.TokenTypeETH)
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
			lastMessage.TokenType = int(btypes.TokenTypeERC20)
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
			lastMessage.TokenType = int(btypes.TokenTypeERC721)
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
			lastMessage.TokenType = int(btypes.TokenTypeERC721)
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
			lastMessage.TokenType = int(btypes.TokenTypeERC1155)
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
			lastMessage.TokenType = int(btypes.TokenTypeERC1155)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = utils.ConvertBigIntArrayToString(event.TokenIDs)
			lastMessage.TokenAmounts = utils.ConvertBigIntArrayToString(event.TokenAmounts)
		case backendabi.L1DepositWrappedTokenSig:
			event := backendabi.WrappedTokenMessageEvent{}
			if err := utils.UnpackLog(backendabi.L1WrappedTokenGatewayABI, &event, "DepositWrappedToken", vlog); err != nil {
				log.Error("Failed to unpack DepositWrappedToken event", "err", err)
				return nil, nil, err
			}
			lastMessage := l1DepositMessages[len(l1DepositMessages)-1]
			lastMessage.Sender = event.From.String()
		case backendabi.L1SentMessageEventSig:
			event := backendabi.L1SentMessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ScrollMessengerABI, &event, "SentMessage", vlog); err != nil {
				log.Error("Failed to unpack SentMessage event", "err", err)
				return nil, nil, err
			}
			from, err := getRealFromAddress(ctx, event.Sender, event.Message, e.client, vlog.TxHash, e.cfg.GatewayRouterAddr)
			if err != nil {
				log.Error("Failed to get real 'from' address", "err", err)
				return nil, nil, err
			}
			l1DepositMessages = append(l1DepositMessages, &orm.CrossMessage{
				L1BlockNumber:  vlog.BlockNumber,
				Sender:         from,
				Receiver:       event.Target.String(),
				TokenType:      int(btypes.TokenTypeETH),
				L1TxHash:       vlog.TxHash.String(),
				TokenAmounts:   event.Value.String(),
				MessageNonce:   event.MessageNonce.Uint64(),
				MessageType:    int(btypes.MessageTypeL1SentMessage),
				TxStatus:       int(btypes.TxStatusTypeSent),
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
				TxStatus:      int(btypes.TxStatusTypeRelayed),
				MessageType:   int(btypes.MessageTypeL2SentMessage),
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
				TxStatus:      int(btypes.TxStatusTypeFailedRelayed),
				MessageType:   int(btypes.MessageTypeL2SentMessage),
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
				BatchStatus:      int(btypes.BatchStatusTypeCommitted),
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
				BatchStatus:   int(btypes.BatchStatusTypeReverted),
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
				BatchStatus:   int(btypes.BatchStatusTypeFinalized),
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
					EventType:   btypes.MessageQueueEventTypeQueueTransaction,
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
					EventType:  btypes.MessageQueueEventTypeDequeueTransaction,
					QueueIndex: index,
				})
			}
		case backendabi.L1ResetDequeuedTransactionEventSig:
			event := backendabi.L1ResetDequeuedTransactionEvent{}
			if err := utils.UnpackLog(backendabi.IL1MessageQueueABI, &event, "ResetDequeuedTransaction", vlog); err != nil {
				log.Error("Failed to unpack ResetDequeuedTransaction event", "err", err)
				return nil, err
			}
			l1MessageQueueEvents = append(l1MessageQueueEvents, &orm.MessageQueueEvent{
				EventType:  btypes.MessageQueueEventTypeResetDequeuedTransaction,
				QueueIndex: event.StartIndex.Uint64(),
			})
		case backendabi.L1DropTransactionEventSig:
			event := backendabi.L1DropTransactionEvent{}
			if err := utils.UnpackLog(backendabi.IL1MessageQueueABI, &event, "DropTransaction", vlog); err != nil {
				log.Error("Failed to unpack DropTransaction event", "err", err)
				return nil, err
			}
			l1MessageQueueEvents = append(l1MessageQueueEvents, &orm.MessageQueueEvent{
				EventType:  btypes.MessageQueueEventTypeDropTransaction,
				QueueIndex: event.Index.Uint64(),
				TxHash:     vlog.TxHash,
			})
		}
	}
	return l1MessageQueueEvents, nil
}

func getRealFromAddress(ctx context.Context, eventSender common.Address, eventMessage []byte, client *ethclient.Client, txHash common.Hash, gatewayRouterAddr string) (string, error) {
	if eventSender != common.HexToAddress(gatewayRouterAddr) {
		return eventSender.String(), nil
	}

	// deposit/withdraw ETH: EOA -> contract 1 -> ... -> contract n -> gateway router -> messenger.
	if len(eventMessage) >= 32 {
		addressBytes := eventMessage[32-common.AddressLength : 32]
		var address common.Address
		address.SetBytes(addressBytes)

		return address.Hex(), nil
	}

	log.Warn("event message data too short to contain an address", "length", len(eventMessage))

	// Legacy handling logic if length of message < 32, for backward compatibility before the next contract upgrade.
	tx, isPending, rpcErr := client.TransactionByHash(ctx, txHash)
	if rpcErr != nil || isPending {
		log.Error("Failed to get transaction or the transaction is still pending", "rpcErr", rpcErr, "isPending", isPending)
		return "", rpcErr
	}
	// Case 1: deposit/withdraw ETH: EOA -> multisig -> gateway router -> messenger.
	if tx.To() != nil && (*tx.To()).String() != gatewayRouterAddr {
		return (*tx.To()).String(), nil
	}
	// Case 2: deposit/withdraw ETH: EOA -> gateway router -> messenger.
	signer := types.LatestSignerForChainID(new(big.Int).SetUint64(tx.ChainId().Uint64()))
	sender, err := signer.Sender(tx)
	if err != nil {
		log.Error("Get sender failed", "chain id", tx.ChainId().Uint64(), "tx hash", tx.Hash().String(), "err", err)
		return "", err
	}
	return sender.String(), nil
}
