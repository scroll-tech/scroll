package logic

import (
	"context"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	backendabi "scroll-tech/bridge-history-api/abi"
	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/orm"
	btypes "scroll-tech/bridge-history-api/internal/types"
	"scroll-tech/bridge-history-api/internal/utils"
)

// L2EventParser the L2 event parser
type L2EventParser struct {
	cfg    *config.FetcherConfig
	client *ethclient.Client
}

// NewL2EventParser creates the L2 event parser
func NewL2EventParser(cfg *config.FetcherConfig, client *ethclient.Client) *L2EventParser {
	return &L2EventParser{
		cfg:    cfg,
		client: client,
	}
}

// ParseL2EventLogs parses L2 watchedevents
func (e *L2EventParser) ParseL2EventLogs(ctx context.Context, logs []types.Log, blockTimestampsMap map[uint64]uint64) ([]*orm.CrossMessage, []*orm.CrossMessage, []*orm.BridgeBatchDepositEvent, error) {
	l2WithdrawMessages, l2RelayedMessages, err := e.ParseL2SingleCrossChainEventLogs(ctx, logs, blockTimestampsMap)
	if err != nil {
		return nil, nil, nil, err
	}

	l2BridgeBatchDepositMessages, err := e.ParseL2BridgeBatchDepositCrossChainEventLogs(logs, blockTimestampsMap)
	if err != nil {
		return nil, nil, nil, err
	}
	return l2WithdrawMessages, l2RelayedMessages, l2BridgeBatchDepositMessages, nil
}

// ParseL2BridgeBatchDepositCrossChainEventLogs parses L2 watched bridge batch deposit events
func (e *L2EventParser) ParseL2BridgeBatchDepositCrossChainEventLogs(logs []types.Log, blockTimestampsMap map[uint64]uint64) ([]*orm.BridgeBatchDepositEvent, error) {
	var l2BridgeBatchDepositEvents []*orm.BridgeBatchDepositEvent
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L2BridgeBatchDistributeSig:
			event := backendabi.L2BatchBridgeGatewayBatchDistribute{}
			err := utils.UnpackLog(backendabi.L2BatchBridgeGatewayABI, &event, "BatchDistribute", vlog)
			if err != nil {
				log.Error("Failed to unpack BatchDistribute event", "err", err)
				return nil, err
			}

			var tokenType btypes.TokenType
			if event.L1Token == common.HexToAddress("0") {
				tokenType = btypes.TokenTypeETH
			} else {
				tokenType = btypes.TokenTypeERC20
			}

			l2BridgeBatchDepositEvents = append(l2BridgeBatchDepositEvents, &orm.BridgeBatchDepositEvent{
				TokenType:      int(tokenType),
				BatchIndex:     event.BatchIndex.Uint64(),
				L2TokenAddress: event.L2Token.String(),
				L2BlockNumber:  vlog.BlockNumber,
				L2TxHash:       vlog.TxHash.String(),
				TxStatus:       int(btypes.TxStatusBridgeBatchDistribute),
				BlockTimestamp: blockTimestampsMap[vlog.BlockNumber],
				L2LogIndex:     vlog.Index,
			})
		case backendabi.L2BridgeBatchDistributeFailedSig:
			event := backendabi.L2BatchBridgeGatewayDistributeFailed{}
			err := utils.UnpackLog(backendabi.L2BatchBridgeGatewayABI, &event, "DistributeFailed", vlog)
			if err != nil {
				log.Error("Failed to unpack DistributeFailed event", "err", err)
				return nil, err
			}
			l2BridgeBatchDepositEvents = append(l2BridgeBatchDepositEvents, &orm.BridgeBatchDepositEvent{
				BatchIndex:     event.BatchIndex.Uint64(),
				L2TokenAddress: event.L2Token.String(),
				L2BlockNumber:  vlog.BlockNumber,
				L2TxHash:       vlog.TxHash.String(),
				TxStatus:       int(btypes.TxStatusBridgeBatchDistributeFailed),
				BlockTimestamp: blockTimestampsMap[vlog.BlockNumber],
				Sender:         event.Receiver.String(),
				L2LogIndex:     vlog.Index,
			})
		}
	}
	return l2BridgeBatchDepositEvents, nil
}

// ParseL2SingleCrossChainEventLogs parses L2 watched events
func (e *L2EventParser) ParseL2SingleCrossChainEventLogs(ctx context.Context, logs []types.Log, blockTimestampsMap map[uint64]uint64) ([]*orm.CrossMessage, []*orm.CrossMessage, error) {
	var l2WithdrawMessages []*orm.CrossMessage
	var l2RelayedMessages []*orm.CrossMessage
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L2WithdrawETHSig:
			event := backendabi.ETHMessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ETHGatewayABI, &event, "WithdrawETH", vlog)
			if err != nil {
				log.Error("Failed to unpack WithdrawETH event", "err", err)
				return nil, nil, err
			}
			lastMessage := l2WithdrawMessages[len(l2WithdrawMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(btypes.TokenTypeETH)
			lastMessage.TokenAmounts = event.Amount.String()
		case backendabi.L2WithdrawERC20Sig:
			event := backendabi.ERC20MessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ERC20GatewayABI, &event, "WithdrawERC20", vlog)
			if err != nil {
				log.Error("Failed to unpack WithdrawERC20 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l2WithdrawMessages[len(l2WithdrawMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(btypes.TokenTypeERC20)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenAmounts = event.Amount.String()
		case backendabi.L2WithdrawERC721Sig:
			event := backendabi.ERC721MessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ERC721GatewayABI, &event, "WithdrawERC721", vlog)
			if err != nil {
				log.Error("Failed to unpack WithdrawERC721 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l2WithdrawMessages[len(l2WithdrawMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(btypes.TokenTypeERC721)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = event.TokenID.String()
		case backendabi.L2BatchWithdrawERC721Sig:
			event := backendabi.BatchERC721MessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ERC721GatewayABI, &event, "BatchWithdrawERC721", vlog)
			if err != nil {
				log.Error("Failed to unpack BatchWithdrawERC721 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l2WithdrawMessages[len(l2WithdrawMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(btypes.TokenTypeERC721)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = utils.ConvertBigIntArrayToString(event.TokenIDs)
		case backendabi.L2WithdrawERC1155Sig:
			event := backendabi.ERC1155MessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ERC1155GatewayABI, &event, "WithdrawERC1155", vlog)
			if err != nil {
				log.Error("Failed to unpack WithdrawERC1155 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l2WithdrawMessages[len(l2WithdrawMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(btypes.TokenTypeERC1155)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = event.TokenID.String()
			lastMessage.TokenAmounts = event.Amount.String()
		case backendabi.L2BatchWithdrawERC1155Sig:
			event := backendabi.BatchERC1155MessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ERC1155GatewayABI, &event, "BatchWithdrawERC1155", vlog)
			if err != nil {
				log.Error("Failed to unpack BatchWithdrawERC1155 event", "err", err)
				return nil, nil, err
			}
			lastMessage := l2WithdrawMessages[len(l2WithdrawMessages)-1]
			lastMessage.Sender = event.From.String()
			lastMessage.Receiver = event.To.String()
			lastMessage.TokenType = int(btypes.TokenTypeERC1155)
			lastMessage.L1TokenAddress = event.L1Token.String()
			lastMessage.L2TokenAddress = event.L2Token.String()
			lastMessage.TokenIDs = utils.ConvertBigIntArrayToString(event.TokenIDs)
			lastMessage.TokenAmounts = utils.ConvertBigIntArrayToString(event.TokenAmounts)
		case backendabi.L2SentMessageEventSig:
			event := backendabi.L2SentMessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ScrollMessengerABI, &event, "SentMessage", vlog)
			if err != nil {
				log.Error("Failed to unpack SentMessage event", "err", err)
				return nil, nil, err
			}
			from, err := getRealFromAddress(ctx, event.Sender, event.Message, e.client, vlog.TxHash, e.cfg.GatewayRouterAddr)
			if err != nil {
				log.Error("Failed to get real 'from' address", "err", err)
				return nil, nil, err
			}
			l2WithdrawMessages = append(l2WithdrawMessages, &orm.CrossMessage{
				MessageHash:    utils.ComputeMessageHash(event.Sender, event.Target, event.Value, event.MessageNonce, event.Message).String(),
				Sender:         from,
				Receiver:       event.Target.String(),
				TokenType:      int(btypes.TokenTypeETH),
				L2TxHash:       vlog.TxHash.String(),
				TokenAmounts:   event.Value.String(),
				MessageFrom:    event.Sender.String(),
				MessageTo:      event.Target.String(),
				MessageValue:   event.Value.String(),
				MessageNonce:   event.MessageNonce.Uint64(),
				MessageData:    hexutil.Encode(event.Message),
				MessageType:    int(btypes.MessageTypeL2SentMessage),
				TxStatus:       int(btypes.TxStatusTypeSent),
				BlockTimestamp: blockTimestampsMap[vlog.BlockNumber],
				L2BlockNumber:  vlog.BlockNumber,
			})
		case backendabi.L2RelayedMessageEventSig:
			event := backendabi.L2RelayedMessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ScrollMessengerABI, &event, "RelayedMessage", vlog)
			if err != nil {
				log.Error("Failed to unpack RelayedMessage event", "err", err)
				return nil, nil, err
			}
			l2RelayedMessages = append(l2RelayedMessages, &orm.CrossMessage{
				MessageHash:   event.MessageHash.String(),
				L2BlockNumber: vlog.BlockNumber,
				L2TxHash:      vlog.TxHash.String(),
				TxStatus:      int(btypes.TxStatusTypeRelayed),
				MessageType:   int(btypes.MessageTypeL1SentMessage),
			})
		case backendabi.L2FailedRelayedMessageEventSig:
			event := backendabi.L2RelayedMessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ScrollMessengerABI, &event, "FailedRelayedMessage", vlog)
			if err != nil {
				log.Error("Failed to unpack FailedRelayedMessage event", "err", err)
				return nil, nil, err
			}
			l2RelayedMessages = append(l2RelayedMessages, &orm.CrossMessage{
				MessageHash:   event.MessageHash.String(),
				L2BlockNumber: vlog.BlockNumber,
				L2TxHash:      vlog.TxHash.String(),
				TxStatus:      int(btypes.TxStatusTypeFailedRelayed),
				MessageType:   int(btypes.MessageTypeL1SentMessage),
			})
		}
	}
	return l2WithdrawMessages, l2RelayedMessages, nil
}
