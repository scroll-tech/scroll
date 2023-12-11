package logic

import (
	"context"
	"math/big"
	"strings"

	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	backendabi "scroll-tech/bridge-history-api/abi"
	"scroll-tech/bridge-history-api/internal/orm"
	"scroll-tech/bridge-history-api/internal/utils"
)

// ParseL1CrossChainEventLogs parses L1 watched cross chain events.
func ParseL1CrossChainEventLogs(ctx context.Context, logs []types.Log, blockTimestampsMap map[uint64]uint64, client *ethclient.Client) ([]*orm.CrossMessage, []*orm.CrossMessage, error) {
	var l1DepositMessages []*orm.CrossMessage
	var l1RelayedMessages []*orm.CrossMessage
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L1DepositETHSig:
			event := backendabi.ETHMessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ETHGatewayABI, &event, "DepositETH", vlog); err != nil {
				log.Warn("Failed to unpack DepositETH event", "err", err)
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
				log.Warn("Failed to unpack DepositERC20 event", "err", err)
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
				log.Warn("Failed to unpack DepositERC721 event", "err", err)
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
				log.Warn("Failed to unpack BatchDepositERC721 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l1DepositMessages[len(l1DepositMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeERC721)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = convertBigIntArrayToString(event.TokenIDs)
		case backendabi.L1DepositERC1155Sig:
			event := backendabi.ERC1155MessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ERC1155GatewayABI, &event, "DepositERC1155", vlog); err != nil {
				log.Warn("Failed to unpack DepositERC1155 event", "err", err)
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
				log.Warn("Failed to unpack BatchDepositERC1155 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l1DepositMessages[len(l1DepositMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeERC1155)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = convertBigIntArrayToString(event.TokenIDs)
			lastMessage.TokenAmounts = convertBigIntArrayToString(event.TokenAmounts)
		case backendabi.L1SentMessageEventSig:
			event := backendabi.L1SentMessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ScrollMessengerABI, &event, "SentMessage", vlog); err != nil {
				log.Warn("Failed to unpack SentMessage event", "err", err)
				return nil, nil, err
			}
			// Use this messageHash as next deposit event's messageHash
			messageHash := utils.ComputeMessageHash(event.Sender, event.Target, event.Value, event.MessageNonce, event.Message).String()
			l1DepositMessages = append(l1DepositMessages, &orm.CrossMessage{
				L1BlockNumber:  vlog.BlockNumber,
				Sender:         event.Sender.String(),
				Receiver:       event.Target.String(),
				TokenType:      int(orm.TokenTypeETH),
				L1TxHash:       vlog.TxHash.String(),
				TokenAmounts:   event.Value.String(),
				MessageNonce:   event.MessageNonce.Uint64(),
				MessageType:    int(orm.MessageTypeL1SentMessage),
				TxStatus:       int(orm.TxStatusTypeSent),
				BlockTimestamp: blockTimestampsMap[vlog.BlockNumber],
				MessageHash:    messageHash,
			})
		case backendabi.L1RelayedMessageEventSig:
			event := backendabi.L1RelayedMessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ScrollMessengerABI, &event, "RelayedMessage", vlog); err != nil {
				log.Warn("Failed to unpack RelayedMessage event", "err", err)
				return nil, nil, err
			}
			l1RelayedMessages = append(l1RelayedMessages, &orm.CrossMessage{
				MessageHash:   event.MessageHash.String(),
				L1BlockNumber: vlog.BlockNumber,
				L1TxHash:      vlog.TxHash.String(),
				TxStatus:      int(orm.TxStatusTypeRelayed),
			})
		case backendabi.L1FailedRelayedMessageEventSig:
			event := backendabi.L1FailedRelayedMessageEvent{}
			if err := utils.UnpackLog(backendabi.IL1ScrollMessengerABI, &event, "FailedRelayedMessage", vlog); err != nil {
				log.Warn("Failed to unpack FailedRelayedMessage event", "err", err)
				return nil, nil, err
			}
			l1RelayedMessages = append(l1RelayedMessages, &orm.CrossMessage{
				MessageHash:   event.MessageHash.String(),
				L1BlockNumber: vlog.BlockNumber,
				L1TxHash:      vlog.TxHash.String(),
				TxStatus:      int(orm.TxStatusTypeRelayedFailed),
			})
		}
	}
	return l1DepositMessages, l1RelayedMessages, nil
}

// ParseL1BatchEventLogs parses L1 watched batch events.
func ParseL1BatchEventLogs(ctx context.Context, logs []types.Log, blockTimestampsMap map[uint64]uint64, client *ethclient.Client) ([]*orm.BatchEvent, error) {
	var l1BatchEvents []*orm.BatchEvent
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L1CommitBatchEventSig:
			event := backendabi.L1CommitBatchEvent{}
			if err := utils.UnpackLog(backendabi.IScrollChainABI, &event, "CommitBatch", vlog); err != nil {
				log.Warn("Failed to unpack CommitBatch event", "err", err)
				return nil, err
			}
			commitTx, isPending, err := client.TransactionByHash(ctx, vlog.TxHash)
			if err != nil || isPending {
				log.Warn("Failed to get commit Batch tx receipt or the tx is still pending", "err", err)
				return nil, err
			}
			startBlock, endBlock, err := utils.GetBatchRangeFromCalldata(commitTx.Data())
			if err != nil {
				log.Warn("Failed to get batch range from calldata", "hash", commitTx.Hash().String(), "height", vlog.BlockNumber)
				return nil, err
			}
			l1BatchEvents = append(l1BatchEvents, &orm.BatchEvent{
				BatchStatus:      int(orm.BatchStatusTypeCommitted),
				BatchIndex:       event.BatchIndex.Uint64(),
				BatchHash:        event.BatchHash.String(),
				StartBlockNumber: startBlock,
				EndBlockNumber:   endBlock,
			})
		case backendabi.L1RevertBatchEventSig:
			event := backendabi.L1RevertBatchEvent{}
			if err := utils.UnpackLog(backendabi.IScrollChainABI, &event, "RevertBatch", vlog); err != nil {
				log.Warn("Failed to unpack RevertBatch event", "err", err)
				return nil, err
			}
			l1BatchEvents = append(l1BatchEvents, &orm.BatchEvent{
				BatchStatus: int(orm.BatchStatusTypeReverted),
				BatchIndex:  event.BatchIndex.Uint64(),
				BatchHash:   event.BatchHash.String(),
			})
		case backendabi.L1FinalizeBatchEventSig:
			event := backendabi.L1FinalizeBatchEvent{}
			if err := utils.UnpackLog(backendabi.IScrollChainABI, &event, "FinalizeBatch", vlog); err != nil {
				log.Warn("Failed to unpack FinalizeBatch event", "err", err)
				return nil, err
			}
			l1BatchEvents = append(l1BatchEvents, &orm.BatchEvent{
				BatchStatus: int(orm.BatchStatusTypeFinalized),
				BatchIndex:  event.BatchIndex.Uint64(),
				BatchHash:   event.BatchHash.String(),
			})
		}
	}
	return l1BatchEvents, nil
}

// ParseL1MessageQueueEventLogs parses L1 watched message queue events.
func ParseL1MessageQueueEventLogs(ctx context.Context, logs []types.Log, blockTimestampsMap map[uint64]uint64, client *ethclient.Client) ([]*orm.MessageQueueEvent, error) {
	var l1MessageQueueEvents []*orm.MessageQueueEvent
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L1QueueTransactionEventSig:
			event := backendabi.L1QueueTransactionEvent{}
			if err := utils.UnpackLog(backendabi.IL1MessageQueueABI, &event, "QueueTransaction", vlog); err != nil {
				log.Warn("Failed to unpack QueueTransaction event", "err", err)
				return nil, err
			}
			// 1. Update queue index of both sent message and replay message.
			// 2. Update tx hash of replay message.
			l1MessageQueueEvents = append(l1MessageQueueEvents, &orm.MessageQueueEvent{
				EventType:  orm.MessageQueueEventTypeQueueTransaction,
				QueueIndex: event.QueueIndex,
				TxHash:     vlog.TxHash,
			})
		case backendabi.L1DequeueTransactionEventSig:
			event := backendabi.L1DequeueTransactionEvent{}
			if err := utils.UnpackLog(backendabi.IL1MessageQueueABI, &event, "DequeueTransaction", vlog); err != nil {
				log.Warn("Failed to unpack DequeueTransaction event", "err", err)
				return nil, err
			}
			skippedIndices := getSkippedQueueIndices(event.StartIndex.Uint64(), event.SkippedBitmap)
			for _, index := range skippedIndices {
				l1MessageQueueEvents = append(l1MessageQueueEvents, &orm.MessageQueueEvent{
					EventType:  orm.MessageQueueEventTypeDequeueTransaction,
					QueueIndex: index,
				})
			}
		case backendabi.L1DropTransactionEventSig:
			event := backendabi.L1DropTransactionEvent{}
			if err := utils.UnpackLog(backendabi.IL1MessageQueueABI, &event, "DropTransaction", vlog); err != nil {
				log.Warn("Failed to unpack DropTransaction event", "err", err)
				return nil, err
			}
			l1MessageQueueEvents = append(l1MessageQueueEvents, &orm.MessageQueueEvent{
				EventType:  orm.MessageQueueEventTypeDropTransaction,
				QueueIndex: event.Index.Uint64(),
			})
		}
	}
	return l1MessageQueueEvents, nil
}

// ParseL2EventLogs parses L2 watched events
func ParseL2EventLogs(logs []types.Log, blockTimestampsMap map[uint64]uint64) ([]*orm.CrossMessage, []*orm.CrossMessage, error) {
	var l2WithdrawMessages []*orm.CrossMessage
	var l2RelayedMessages []*orm.CrossMessage
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L2WithdrawETHSig:
			event := backendabi.ETHMessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ETHGatewayABI, &event, "WithdrawETH", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawETH event", "err", err)
				return nil, nil, err
			}
			lastMessage := l2WithdrawMessages[len(l2WithdrawMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeETH)
			lastMessage.TokenAmounts = event.Amount.String()
		case backendabi.L2WithdrawERC20Sig:
			event := backendabi.ERC20MessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ERC20GatewayABI, &event, "WithdrawERC20", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawERC20 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l2WithdrawMessages[len(l2WithdrawMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeERC20)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenAmounts = event.Amount.String()
		case backendabi.L2WithdrawERC721Sig:
			event := backendabi.ERC721MessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ERC721GatewayABI, &event, "WithdrawERC721", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawERC721 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l2WithdrawMessages[len(l2WithdrawMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeERC721)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = event.TokenID.String()
		case backendabi.L2BatchWithdrawERC721Sig:
			event := backendabi.BatchERC721MessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ERC721GatewayABI, &event, "BatchWithdrawERC721", vlog)
			if err != nil {
				log.Warn("Failed to unpack BatchWithdrawERC721 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l2WithdrawMessages[len(l2WithdrawMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeERC721)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = convertBigIntArrayToString(event.TokenIDs)
		case backendabi.L2WithdrawERC1155Sig:
			event := backendabi.ERC1155MessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ERC1155GatewayABI, &event, "WithdrawERC1155", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawERC1155 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l2WithdrawMessages[len(l2WithdrawMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeERC1155)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = event.TokenID.String()
			lastMessage.TokenAmounts = event.Amount.String()
		case backendabi.L2BatchWithdrawERC1155Sig:
			event := backendabi.BatchERC1155MessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ERC1155GatewayABI, &event, "BatchWithdrawERC1155", vlog)
			if err != nil {
				log.Warn("Failed to unpack BatchWithdrawERC1155 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l2WithdrawMessages[len(l2WithdrawMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(orm.TokenTypeERC1155)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = convertBigIntArrayToString(event.TokenIDs)
			lastMessage.TokenAmounts = convertBigIntArrayToString(event.TokenAmounts)
		case backendabi.L2SentMessageEventSig:
			event := backendabi.L2SentMessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ScrollMessengerABI, &event, "SentMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack SentMessage event", "err", err)
				return nil, nil, err
			}
			// Use this messageHash as next deposit event's messageHash
			messageHash := utils.ComputeMessageHash(event.Sender, event.Target, event.Value, event.MessageNonce, event.Message)
			l2WithdrawMessages = append(l2WithdrawMessages, &orm.CrossMessage{
				MessageHash:    messageHash.String(),
				Sender:         event.Sender.String(),
				Receiver:       event.Target.String(),
				TokenType:      int(orm.TokenTypeETH),
				L2TxHash:       vlog.TxHash.String(),
				TokenAmounts:   event.Value.String(),
				MessageFrom:    event.Sender.String(),
				MessageTo:      event.Target.String(),
				MessageValue:   event.Value.String(),
				MessageNonce:   event.MessageNonce.Uint64(),
				MessageData:    hexutil.Encode(event.Message),
				MessageType:    int(orm.MessageTypeL2SentMessage),
				TxStatus:       int(orm.TxStatusTypeSent),
				BlockTimestamp: blockTimestampsMap[vlog.BlockNumber],
				L2BlockNumber:  vlog.BlockNumber,
			})
		case backendabi.L2RelayedMessageEventSig:
			event := backendabi.L2RelayedMessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ScrollMessengerABI, &event, "RelayedMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack RelayedMessage event", "err", err)
				return nil, nil, err
			}
			l2RelayedMessages = append(l2RelayedMessages, &orm.CrossMessage{
				MessageHash:   event.MessageHash.String(),
				L2BlockNumber: vlog.BlockNumber,
				L2TxHash:      vlog.TxHash.String(),
				TxStatus:      int(orm.TxStatusTypeRelayed),
			})
		case backendabi.L2FailedRelayedMessageEventSig:
			event := backendabi.L2RelayedMessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ScrollMessengerABI, &event, "FailedRelayedMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack FailedRelayedMessage event", "err", err)
				return nil, nil, err
			}
			l2RelayedMessages = append(l2RelayedMessages, &orm.CrossMessage{
				MessageHash:   event.MessageHash.String(),
				L2BlockNumber: vlog.BlockNumber,
				L2TxHash:      vlog.TxHash.String(),
				TxStatus:      int(orm.TxStatusTypeRelayedFailed),
			})
		}
	}
	return l2WithdrawMessages, l2RelayedMessages, nil
}

func convertBigIntArrayToString(array []*big.Int) string {
	stringArray := make([]string, len(array))
	for i, num := range array {
		stringArray[i] = num.String()
	}

	result := strings.Join(stringArray, ", ")
	return result
}

func getSkippedQueueIndices(startIndex uint64, skippedBitmap *big.Int) []uint64 {
	var indices []uint64
	for i := 0; i < 256; i++ {
		index := startIndex + uint64(i)
		bit := new(big.Int).Rsh(skippedBitmap, uint(i))
		if bit.Bit(0) == 0 {
			continue
		}
		indices = append(indices, index)
	}
	return indices
}
