package logic

import (
	"context"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	backendabi "scroll-tech/bridge-history-api/abi"
	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/orm"
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

// ParseL2EventLogs parses L2 watched events
func (e *L2EventParser) ParseL2EventLogs(ctx context.Context, logs []types.Log, blockTimestampsMap map[uint64]uint64) ([]*orm.CrossMessage, []*orm.CrossMessage, error) {
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
			lastMessage.TokenIDs = utils.ConvertBigIntArrayToString(event.TokenIDs)
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
			lastMessage.TokenIDs = utils.ConvertBigIntArrayToString(event.TokenIDs)
			lastMessage.TokenAmounts = utils.ConvertBigIntArrayToString(event.TokenAmounts)
		case backendabi.L2SentMessageEventSig:
			event := backendabi.L2SentMessageEvent{}
			err := utils.UnpackLog(backendabi.IL2ScrollMessengerABI, &event, "SentMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack SentMessage event", "err", err)
				return nil, nil, err
			}
			from := event.Sender.String()
			if event.Sender.String() == e.cfg.GatewayRouterAddr {
				tx, isPending, rpcErr := e.client.TransactionByHash(ctx, vlog.TxHash)
				if err != nil || isPending {
					log.Warn("Failed to get tx or the tx is still pending", "rpcErr", rpcErr, "isPending", isPending)
					return nil, nil, rpcErr
				}
				signer := types.LatestSignerForChainID(new(big.Int).SetUint64(tx.ChainId().Uint64()))
				sender, senderErr := signer.Sender(tx)
				if senderErr != nil {
					log.Error("get sender failed", "chain id", tx.ChainId().Uint64(), "tx hash", tx.Hash().String(), "err", senderErr)
					return nil, nil, senderErr
				}
				from = sender.String()
			}
			l2WithdrawMessages = append(l2WithdrawMessages, &orm.CrossMessage{
				MessageHash:    utils.ComputeMessageHash(event.Sender, event.Target, event.Value, event.MessageNonce, event.Message).String(),
				Sender:         from,
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
				MessageType:   int(orm.MessageTypeL1SentMessage),
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
				TxStatus:      int(orm.TxStatusTypeFailedRelayed),
				MessageType:   int(orm.MessageTypeL1SentMessage),
			})
		}
	}
	return l2WithdrawMessages, l2RelayedMessages, nil
}
