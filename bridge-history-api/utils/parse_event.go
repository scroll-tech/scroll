package utils

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	backendabi "bridge-history-api/abi"
	"bridge-history-api/db/orm"
)

type CachedParsedTxCalldata struct {
	CallDataIndex uint64
	BatchIndices  []uint64
	StartBlocks   []uint64
	EndBlocks     []uint64
}

func ParseBackendL1EventLogs(logs []types.Log) ([]*orm.CrossMsg, []*orm.RelayedMsg, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l1CrossMsg []*orm.CrossMsg
	var relayedMsgs []*orm.RelayedMsg
	var msgHash string
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L1DepositETHSig:
			event := backendabi.DepositETH{}
			err := UnpackLog(backendabi.L1ETHGatewayABI, &event, "DepositETH", vlog)
			if err != nil {
				log.Warn("Failed to unpack DepositETH event", "err", err)
				return l1CrossMsg, relayedMsgs, err
			}
			l1CrossMsg = append(l1CrossMsg, &orm.CrossMsg{
				Height:     vlog.BlockNumber,
				Sender:     event.From.String(),
				Target:     event.To.String(),
				Amount:     event.Amount.String(),
				Asset:      int(orm.ETH),
				Layer1Hash: vlog.TxHash.Hex(),
				MsgHash:    msgHash,
			})
		case backendabi.L1DepositERC20Sig:
			event := backendabi.ERC20MessageEvent{}
			err := UnpackLog(backendabi.L1StandardERC20GatewayABI, &event, "DepositERC20", vlog)
			if err != nil {
				log.Warn("Failed to unpack DepositERC20 event", "err", err)
				return l1CrossMsg, relayedMsgs, err
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
				MsgHash:     msgHash,
			})
		case backendabi.L1DepositERC721Sig:
			event := backendabi.ERC721MessageEvent{}
			err := UnpackLog(backendabi.L1ERC721GatewayABI, &event, "DepositERC721", vlog)
			if err != nil {
				log.Warn("Failed to unpack DepositERC721 event", "err", err)
				return l1CrossMsg, relayedMsgs, err
			}
			l1CrossMsg = append(l1CrossMsg, &orm.CrossMsg{
				Height:      vlog.BlockNumber,
				Sender:      event.From.String(),
				Target:      event.To.String(),
				Asset:       int(orm.ERC721),
				Layer1Hash:  vlog.TxHash.Hex(),
				Layer1Token: event.L1Token.Hex(),
				Layer2Token: event.L2Token.Hex(),
				TokenIDs:    event.TokenID.String(),
				MsgHash:     msgHash,
			})
		case backendabi.L1DepositERC1155Sig:
			event := backendabi.ERC1155MessageEvent{}
			err := UnpackLog(backendabi.L1ERC1155GatewayABI, &event, "DepositERC1155", vlog)
			if err != nil {
				log.Warn("Failed to unpack DepositERC1155 event", "err", err)
				return l1CrossMsg, relayedMsgs, err
			}
			l1CrossMsg = append(l1CrossMsg, &orm.CrossMsg{
				Height:      vlog.BlockNumber,
				Sender:      event.From.String(),
				Target:      event.To.String(),
				Asset:       int(orm.ERC1155),
				Layer1Hash:  vlog.TxHash.Hex(),
				Layer1Token: event.L1Token.Hex(),
				Layer2Token: event.L2Token.Hex(),
				TokenIDs:    event.TokenID.String(),
				Amount:      event.Amount.String(),
				MsgHash:     msgHash,
			})
		case backendabi.L1SentMessageEventSignature:
			event := backendabi.L1SentMessageEvent{}
			err := UnpackLog(backendabi.L1ScrollMessengerABI, &event, "SentMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack SentMessage event", "err", err)
				return l1CrossMsg, relayedMsgs, err
			}
			// since every deposit event will emit after a sent event, so can use this msg_hash as next withdraw event's msg_hash
			msgHash = ComputeMessageHash(event.Sender, event.Target, event.Value, event.MessageNonce, event.Message).Hex()

		case backendabi.L1RelayedMessageEventSignature:
			event := backendabi.L1RelayedMessageEvent{}
			err := UnpackLog(backendabi.L1ScrollMessengerABI, &event, "RelayedMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack RelayedMessage event", "err", err)
				return l1CrossMsg, relayedMsgs, err
			}
			relayedMsgs = append(relayedMsgs, &orm.RelayedMsg{
				MsgHash:    event.MessageHash.String(),
				Height:     vlog.BlockNumber,
				Layer1Hash: vlog.TxHash.Hex(),
			})

		}

	}
	return l1CrossMsg, relayedMsgs, nil
}

func ParseBackendL2EventLogs(logs []types.Log) ([]*orm.CrossMsg, []*orm.RelayedMsg, []*orm.L2SentMsg, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l2CrossMsg []*orm.CrossMsg
	// this is use to confirm finalized l1 msg
	var relayedMsgs []*orm.RelayedMsg
	var l2SentMsgs []*orm.L2SentMsg
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L2WithdrawETHSig:
			event := backendabi.DepositETH{}
			err := UnpackLog(backendabi.L2ETHGatewayABI, &event, "WithdrawETH", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawETH event", "err", err)
				return l2CrossMsg, relayedMsgs, l2SentMsgs, err
			}
			l2SentMsgs[len(l2SentMsgs)-1].OriginalSender = event.From.Hex()
			l2CrossMsg = append(l2CrossMsg, &orm.CrossMsg{
				Height:     vlog.BlockNumber,
				Sender:     event.From.String(),
				Target:     event.To.String(),
				Amount:     event.Amount.String(),
				Asset:      int(orm.ETH),
				Layer2Hash: vlog.TxHash.Hex(),
				MsgHash:    l2SentMsgs[len(l2SentMsgs)-1].MsgHash,
			})
		case backendabi.L2WithdrawERC20Sig:
			event := backendabi.ERC20MessageEvent{}
			err := UnpackLog(backendabi.L2StandardERC20GatewayABI, &event, "WithdrawERC20", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawERC20 event", "err", err)
				return l2CrossMsg, relayedMsgs, l2SentMsgs, err
			}
			l2SentMsgs[len(l2SentMsgs)-1].OriginalSender = event.From.Hex()
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
				return l2CrossMsg, relayedMsgs, l2SentMsgs, err
			}
			l2SentMsgs[len(l2SentMsgs)-1].OriginalSender = event.From.Hex()
			l2CrossMsg = append(l2CrossMsg, &orm.CrossMsg{
				Height:      vlog.BlockNumber,
				Sender:      event.From.String(),
				Target:      event.To.String(),
				Asset:       int(orm.ERC721),
				Layer2Hash:  vlog.TxHash.Hex(),
				Layer1Token: event.L1Token.Hex(),
				Layer2Token: event.L2Token.Hex(),
				TokenIDs:    event.TokenID.String(),
			})
		case backendabi.L2WithdrawERC1155Sig:
			event := backendabi.ERC1155MessageEvent{}
			err := UnpackLog(backendabi.L2ERC1155GatewayABI, &event, "WithdrawERC1155", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawERC1155 event", "err", err)
				return l2CrossMsg, relayedMsgs, l2SentMsgs, err
			}
			l2SentMsgs[len(l2SentMsgs)-1].OriginalSender = event.From.Hex()
			l2CrossMsg = append(l2CrossMsg, &orm.CrossMsg{
				Height:      vlog.BlockNumber,
				Sender:      event.From.String(),
				Target:      event.To.String(),
				Asset:       int(orm.ERC1155),
				Layer2Hash:  vlog.TxHash.Hex(),
				Layer1Token: event.L1Token.Hex(),
				Layer2Token: event.L2Token.Hex(),
				TokenIDs:    event.TokenID.String(),
				Amount:      event.Amount.String(),
			})
		case backendabi.L2SentMessageEventSignature:
			event := backendabi.L2SentMessageEvent{}
			err := UnpackLog(backendabi.L2ScrollMessengerABI, &event, "SentMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack SentMessage event", "err", err)
				return l2CrossMsg, relayedMsgs, l2SentMsgs, err
			}
			// since every withdraw event will emit after a sent event, so can use this msg_hash as next withdraw event's msg_hash
			msgHash := ComputeMessageHash(event.Sender, event.Target, event.Value, event.MessageNonce, event.Message)
			l2SentMsgs = append(l2SentMsgs,
				&orm.L2SentMsg{
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
				return l2CrossMsg, relayedMsgs, l2SentMsgs, err
			}
			relayedMsgs = append(relayedMsgs, &orm.RelayedMsg{
				MsgHash:    event.MessageHash.String(),
				Height:     vlog.BlockNumber,
				Layer2Hash: vlog.TxHash.Hex(),
			})

		}
	}
	return l2CrossMsg, relayedMsgs, l2SentMsgs, nil
}

func ParseBatchInfoFromScrollChain(ctx context.Context, client *ethclient.Client, logs []types.Log) ([]*orm.RollupBatch, error) {
	var rollupBatches []*orm.RollupBatch
	cache := make(map[string]CachedParsedTxCalldata)
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L1CommitBatchEventSignature:
			event := backendabi.L1CommitBatchEvent{}
			err := UnpackLog(backendabi.ScrollChainABI, &event, "CommitBatch", vlog)
			if err != nil {
				log.Warn("Failed to unpack CommitBatch event", "err", err)
				return rollupBatches, err
			}
			if _, ok := cache[vlog.TxHash.Hex()]; ok {
				c := cache[vlog.TxHash.Hex()]
				c.CallDataIndex++
				rollupBatches = append(rollupBatches, &orm.RollupBatch{
					CommitHeight:     vlog.BlockNumber,
					BatchIndex:       c.BatchIndices[c.CallDataIndex],
					BatchHash:        event.BatchHash.Hex(),
					StartBlockNumber: c.StartBlocks[c.CallDataIndex],
					EndBlockNumber:   c.EndBlocks[c.CallDataIndex],
				})
				cache[vlog.TxHash.Hex()] = c
				continue
			}

			commitTx, isPending, err := client.TransactionByHash(ctx, vlog.TxHash)
			if err != nil || isPending {
				log.Warn("Failed to get commit Batch tx receipt or the tx is still pending", "err", err)
				return rollupBatches, err
			}
			indices, startBlocks, endBlocks, err := GetBatchRangeFromCalldataV1(commitTx.Data())
			if err != nil {
				log.Warn("Failed to get batch range from calldata", "hash", commitTx.Hash().Hex(), "height", vlog.BlockNumber)
				return rollupBatches, err
			}
			cache[vlog.TxHash.Hex()] = CachedParsedTxCalldata{
				CallDataIndex: 0,
				BatchIndices:  indices,
				StartBlocks:   startBlocks,
				EndBlocks:     endBlocks,
			}
			rollupBatches = append(rollupBatches, &orm.RollupBatch{
				CommitHeight:     vlog.BlockNumber,
				BatchIndex:       indices[0],
				BatchHash:        event.BatchHash.Hex(),
				StartBlockNumber: startBlocks[0],
				EndBlockNumber:   endBlocks[0],
			})

		default:
			continue
		}
	}
	return rollupBatches, nil
}
