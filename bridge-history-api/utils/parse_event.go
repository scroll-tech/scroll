package utils

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	backendabi "bridge-history-api/abi"
	"bridge-history-api/db/orm"
)

type MsgHashWrapper struct {
	MsgHash common.Hash
	TxHash  common.Hash
}

func ParseBackendL1EventLogs(logs []types.Log) ([]*orm.CrossMsg, []MsgHashWrapper, []*orm.RelayedMsg, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l1CrossMsg []*orm.CrossMsg
	var relayedMsgs []*orm.RelayedMsg
	var msgHashes []MsgHashWrapper
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L1DepositETHSig:
			event := backendabi.DepositETH{}
			err := UnpackLog(backendabi.L1ETHGatewayABI, &event, "DepositETH", vlog)
			if err != nil {
				log.Warn("Failed to unpack DepositETH event", "err", err)
				return l1CrossMsg, msgHashes, relayedMsgs, err
			}
			l1CrossMsg = append(l1CrossMsg, &orm.CrossMsg{
				Height:     vlog.BlockNumber,
				Sender:     event.From.String(),
				Target:     event.To.String(),
				Amount:     event.Amount.String(),
				Asset:      int(orm.ETH),
				Layer1Hash: vlog.TxHash.Hex(),
			})
		case backendabi.L1DepositERC20Sig:
			event := backendabi.ERC20MessageEvent{}
			err := UnpackLog(backendabi.L1StandardERC20GatewayABI, &event, "DepositERC20", vlog)
			if err != nil {
				log.Warn("Failed to unpack DepositERC20 event", "err", err)
				return l1CrossMsg, msgHashes, relayedMsgs, err
			}
			l1CrossMsg = append(l1CrossMsg, &orm.CrossMsg{
				Height:      vlog.BlockNumber,
				Sender:      event.From.String(),
				Target:      event.To.String(),
				Amount:      event.Amount.String(),
				Asset:       int(orm.ERC20),
				Layer1Hash:  vlog.TxHash.Hex(),
				Layer1Token: event.L1Token.Hex(),
				Layer2Token: event.L2Token.Hex(),
			})
		case backendabi.L1DepositERC721Sig:
			event := backendabi.ERC721MessageEvent{}
			err := UnpackLog(backendabi.L1ERC721GatewayABI, &event, "DepositERC721", vlog)
			if err != nil {
				log.Warn("Failed to unpack DepositERC721 event", "err", err)
				return l1CrossMsg, msgHashes, relayedMsgs, err
			}
			l1CrossMsg = append(l1CrossMsg, &orm.CrossMsg{
				Height:      vlog.BlockNumber,
				Sender:      event.From.String(),
				Target:      event.To.String(),
				Asset:       int(orm.ERC721),
				Layer1Hash:  vlog.TxHash.Hex(),
				Layer1Token: event.L1Token.Hex(),
				Layer2Token: event.L2Token.Hex(),
				TokenID:     event.TokenID.Uint64(),
			})
		case backendabi.L1DepositERC1155Sig:
			event := backendabi.ERC1155MessageEvent{}
			err := UnpackLog(backendabi.L1ERC1155GatewayABI, &event, "DepositERC1155", vlog)
			if err != nil {
				log.Warn("Failed to unpack DepositERC1155 event", "err", err)
				return l1CrossMsg, msgHashes, relayedMsgs, err
			}
			l1CrossMsg = append(l1CrossMsg, &orm.CrossMsg{
				Height:      vlog.BlockNumber,
				Sender:      event.From.String(),
				Target:      event.To.String(),
				Asset:       int(orm.ERC1155),
				Layer1Hash:  vlog.TxHash.Hex(),
				Layer1Token: event.L1Token.Hex(),
				Layer2Token: event.L2Token.Hex(),
				TokenID:     event.TokenID.Uint64(),
				Amount:      event.Amount.String(),
			})
		case backendabi.L1SentMessageEventSignature:
			event := backendabi.L1SentMessageEvent{}
			err := UnpackLog(backendabi.L1ScrollMessengerABI, &event, "SentMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack SentMessage event", "err", err)
				return l1CrossMsg, msgHashes, relayedMsgs, err
			}
			msgHash := ComputeMessageHash(event.Sender, event.Target, event.Value, event.MessageNonce, event.Message)
			msgHashes = append(msgHashes, MsgHashWrapper{
				MsgHash: msgHash,
				TxHash:  vlog.TxHash})
		case backendabi.L1RelayedMessageEventSignature:
			event := backendabi.L1RelayedMessageEvent{}
			err := UnpackLog(backendabi.L1ScrollMessengerABI, &event, "RelayedMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack RelayedMessage event", "err", err)
				return l1CrossMsg, msgHashes, relayedMsgs, err
			}
			relayedMsgs = append(relayedMsgs, &orm.RelayedMsg{
				MsgHash:    event.MessageHash.String(),
				Height:     vlog.BlockNumber,
				Layer1Hash: vlog.TxHash.Hex(),
			})

		}

	}
	return l1CrossMsg, msgHashes, relayedMsgs, nil
}

func ParseBackendL2EventLogs(logs []types.Log) ([]*orm.CrossMsg, []MsgHashWrapper, []*orm.RelayedMsg, []*orm.L2SentMsg, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l2CrossMsg []*orm.CrossMsg
	// this is use to confirm finalized l1 msg
	var relayedMsgs []*orm.RelayedMsg
	var l2SentMsg []*orm.L2SentMsg
	var msgHashes []MsgHashWrapper
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L2WithdrawETHSig:
			event := backendabi.DepositETH{}
			err := UnpackLog(backendabi.L2ETHGatewayABI, &event, "WithdrawETH", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawETH event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, l2SentMsg, err
			}
			l2CrossMsg = append(l2CrossMsg, &orm.CrossMsg{
				Height:     vlog.BlockNumber,
				Sender:     event.From.String(),
				Target:     event.To.String(),
				Amount:     event.Amount.String(),
				Asset:      int(orm.ETH),
				Layer2Hash: vlog.TxHash.Hex(),
			})
		case backendabi.L2WithdrawERC20Sig:
			event := backendabi.ERC20MessageEvent{}
			err := UnpackLog(backendabi.L2StandardERC20GatewayABI, &event, "WithdrawERC20", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawERC20 event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, l2SentMsg, err
			}
			l2CrossMsg = append(l2CrossMsg, &orm.CrossMsg{
				Height:      vlog.BlockNumber,
				Sender:      event.From.String(),
				Target:      event.To.String(),
				Amount:      event.Amount.String(),
				Asset:       int(orm.ERC20),
				Layer2Hash:  vlog.TxHash.Hex(),
				Layer1Token: event.L1Token.Hex(),
				Layer2Token: event.L2Token.Hex(),
			})
		case backendabi.L2WithdrawERC721Sig:
			event := backendabi.ERC721MessageEvent{}
			err := UnpackLog(backendabi.L2ERC721GatewayABI, &event, "WithdrawERC721", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawERC721 event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, l2SentMsg, err
			}
			l2CrossMsg = append(l2CrossMsg, &orm.CrossMsg{
				Height:      vlog.BlockNumber,
				Sender:      event.From.String(),
				Target:      event.To.String(),
				Asset:       int(orm.ERC721),
				Layer2Hash:  vlog.TxHash.Hex(),
				Layer1Token: event.L1Token.Hex(),
				Layer2Token: event.L2Token.Hex(),
				TokenID:     event.TokenID.Uint64(),
			})
		case backendabi.L2WithdrawERC1155Sig:
			event := backendabi.ERC1155MessageEvent{}
			err := UnpackLog(backendabi.L2ERC1155GatewayABI, &event, "WithdrawERC1155", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawERC1155 event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, l2SentMsg, err
			}
			l2CrossMsg = append(l2CrossMsg, &orm.CrossMsg{
				Height:      vlog.BlockNumber,
				Sender:      event.From.String(),
				Target:      event.To.String(),
				Asset:       int(orm.ERC1155),
				Layer2Hash:  vlog.TxHash.Hex(),
				Layer1Token: event.L1Token.Hex(),
				Layer2Token: event.L2Token.Hex(),
				TokenID:     event.TokenID.Uint64(),
				Amount:      event.Amount.String(),
			})
		case backendabi.L2SentMessageEventSignature:
			event := backendabi.L2SentMessageEvent{}
			err := UnpackLog(backendabi.L2ScrollMessengerABI, &event, "SentMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack SentMessage event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, l2SentMsg, err
			}
			msgHash := ComputeMessageHash(event.Sender, event.Target, event.Value, event.MessageNonce, event.Message)
			l2SentMsg = append(l2SentMsg, &orm.L2SentMsg{
				Sender:  event.Sender.Hex(),
				Target:  event.Target.Hex(),
				Value:   event.Value.String(),
				MsgHash: msgHash.Hex(),
				Height:  vlog.BlockNumber,
				Nonce:   event.MessageNonce.Uint64(),
				MsgData: hexutil.Encode(event.Message),
			})
		case backendabi.L2RelayedMessageEventSignature:
			event := backendabi.L2RelayedMessageEvent{}
			err := UnpackLog(backendabi.L2ScrollMessengerABI, &event, "RelayedMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack RelayedMessage event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, l2SentMsg, err
			}
			relayedMsgs = append(relayedMsgs, &orm.RelayedMsg{
				MsgHash:    event.MessageHash.String(),
				Height:     vlog.BlockNumber,
				Layer2Hash: vlog.TxHash.Hex(),
			})

		}
	}
	return l2CrossMsg, msgHashes, relayedMsgs, l2SentMsg, nil
}

func ParseBatchInfoFromScrollChain(ctx context.Context, client *ethclient.Client, logs []types.Log) ([]*orm.L2SentMsg, error) {
	var l2SentMsg []*orm.L2SentMsg
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L1CommitBatchEventSignature:
			event := backendabi.L1CommitBatchEvent{}
			err := UnpackLog(backendabi.L1ScrollMessengerABI, &event, "CommitBatch", vlog)
			if err != nil {
				log.Warn("Failed to unpack CommitBatch event", "err", err)
				return l2SentMsg, err
			}
			commitTx, is_pending, err := client.TransactionByHash(ctx, vlog.TxHash)
			if err != nil || is_pending {
				log.Warn("Failed to get commit Batch tx receipt or the tx is still pending", "err", err)
				return l2SentMsg, err
			}
			startBlcock, endBlockNumber := GetBatchRangeFromCalldata(commitTx.Data())
		default:
			continue
		}
	}
	return l2SentMsg, nil
}
