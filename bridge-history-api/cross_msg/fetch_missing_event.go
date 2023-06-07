package cross_msg

import (
	"context"
	"math/big"

	geth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/jmoiron/sqlx"

	backendabi "bridge-history-api/abi"
	"bridge-history-api/db"
	"bridge-history-api/db/orm"
	"bridge-history-api/utils"
)

// Todo : read from config
var (
	// the number of blocks fetch per round
	FETCH_LIMIT = int64(3000)
)

// FetchAndSave is a function type that fetches events from blockchain and saves them to database
type FetchAndSave func(ctx context.Context, client *ethclient.Client, database db.OrmFactory, from int64, to int64, addressList []common.Address) error

// GetLatestProcessed is a function type that gets the latest processed block height from database
type GetLatestProcessed func(db db.OrmFactory) (int64, error)
type UpdateXHash func(ctx context.Context)

type FetchEventWorker struct {
	F    FetchAndSave
	G    GetLatestProcessed
	Name string
}

type msgHashWrapper struct {
	msgHash common.Hash
	txHash  common.Hash
}

func GetLatestL1ProcessedHeight(db db.OrmFactory) (int64, error) {
	crossHeight, err := db.GetLatestL1ProcessedHeight()
	if err != nil {
		log.Error("failed to get L1 cross message processed height: ", "err", err)
		return 0, err
	}
	relayedHeight, err := db.GetLatestRelayedHeightOnL1()
	if err != nil {
		log.Error("failed to get L1 relayed message processed height: ", "err", err)
		return 0, err
	}
	if crossHeight > relayedHeight {
		return crossHeight, nil
	} else {
		return relayedHeight, nil
	}
}

func GetLatestL2ProcessedHeight(db db.OrmFactory) (int64, error) {
	crossHeight, err := db.GetLatestL2ProcessedHeight()
	if err != nil {
		log.Error("failed to get L2 cross message processed height", "err", err)
		return 0, err
	}
	relayedHeight, err := db.GetLatestRelayedHeightOnL2()
	if err != nil {
		log.Error("failed to get L2 relayed message processed height", "err", err)
		return 0, err
	}
	if crossHeight > relayedHeight {
		return crossHeight, nil
	} else {
		return relayedHeight, nil
	}
}

func L1FetchAndSaveEvents(ctx context.Context, client *ethclient.Client, database db.OrmFactory, from int64, to int64, addrList []common.Address) error {

	query := geth.FilterQuery{
		FromBlock: big.NewInt(from), // inclusive
		ToBlock:   big.NewInt(to),   // inclusive
		Addresses: addrList,
		Topics:    make([][]common.Hash, 1),
	}
	query.Topics[0] = make([]common.Hash, 7)
	query.Topics[0][0] = backendabi.L1DepositETHSig
	query.Topics[0][1] = backendabi.L1DepositERC20Sig
	query.Topics[0][2] = backendabi.L1RelayedMessageEventSignature
	query.Topics[0][3] = backendabi.L1SentMessageEventSignature
	query.Topics[0][4] = backendabi.L1DepositERC721Sig
	query.Topics[0][5] = backendabi.L1DepositERC1155Sig
	query.Topics[0][6] = backendabi.L1DepositWETHSig

	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		log.Warn("Failed to get l1 event logs", "err", err)
		return err
	}
	depositL1CrossMsgs, msgHashes, relayedMsg, err := parseBackendL1EventLogs(logs)
	if err != nil {
		log.Error("l1FetchAndSaveEvents: Failed to parse cross msg event logs", "err", err)
		return err
	}
	dbTx, err := database.Beginx()
	if err != nil {
		log.Error("l2FetchAndSaveEvents: Failed to begin db transaction", "err", err)
		return err
	}
	err = database.BatchInsertL1CrossMsgDBTx(dbTx, depositL1CrossMsgs)
	if err != nil {
		dbTx.Rollback()
		log.Crit("l1FetchAndSaveEvents: Failed to insert cross msg event logs", "err", err)
	}

	err = database.BatchInsertRelayedMsgDBTx(dbTx, relayedMsg)
	if err != nil {
		dbTx.Rollback()
		log.Crit("l1FetchAndSaveEvents: Failed to insert relayed message event logs", "err", err)
	}
	err = updateL1CrossMsgMsgHash(ctx, dbTx, database, msgHashes)
	if err != nil {
		dbTx.Rollback()
		log.Crit("l1FetchAndSaveEvents: Failed to update msgHash in L1 cross msg", "err", err)
	}
	err = dbTx.Commit()
	if err != nil {
		// if we can not insert into DB, there must something wrong, need a on-call member handle the dababase manually
		dbTx.Rollback()
		log.Error("l1FetchAndSaveEvents: Failed to commit db transaction", "err", err)
		return err
	}

	return nil
}

func L2FetchAndSaveEvents(ctx context.Context, client *ethclient.Client, database db.OrmFactory, from int64, to int64, addrList []common.Address) error {
	query := geth.FilterQuery{
		FromBlock: big.NewInt(from), // inclusive
		ToBlock:   big.NewInt(to),   // inclusive
		Addresses: addrList,
		Topics:    make([][]common.Hash, 1),
	}
	query.Topics[0] = make([]common.Hash, 7)
	query.Topics[0][0] = backendabi.L2WithdrawETHSig
	query.Topics[0][1] = backendabi.L2WithdrawERC20Sig
	query.Topics[0][2] = backendabi.L2RelayedMessageEventSignature
	query.Topics[0][3] = backendabi.L2SentMessageEventSignature
	query.Topics[0][4] = backendabi.L2WithdrawERC721Sig
	query.Topics[0][5] = backendabi.L2WithdrawERC1155Sig
	query.Topics[0][6] = backendabi.L2WithdrawWETHSig

	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		log.Warn("Failed to get l2 event logs", "err", err)
		return err
	}
	depositL2CrossMsgs, msgHashes, relayedMsg, err := parseBackendL2EventLogs(logs)
	if err != nil {
		log.Error("l2FetchAndSaveEvents: Failed to parse cross msg event logs", "err", err)
		return err
	}
	dbTx, err := database.Beginx()
	if err != nil {
		log.Error("l2FetchAndSaveEvents: Failed to begin db transaction", "err", err)
		return err
	}
	err = database.BatchInsertL2CrossMsgDBTx(dbTx, depositL2CrossMsgs)
	if err != nil {
		dbTx.Rollback()
		log.Crit("l2FetchAndSaveEvents: Failed to insert cross msg event logs", "err", err)
	}

	err = database.BatchInsertRelayedMsgDBTx(dbTx, relayedMsg)
	if err != nil {
		dbTx.Rollback()
		log.Crit("l2FetchAndSaveEvents: Failed to insert relayed message event logs", "err", err)
	}
	err = updateL2CrossMsgMsgHash(ctx, dbTx, database, msgHashes)
	if err != nil {
		dbTx.Rollback()
		log.Crit("l2FetchAndSaveEvents: Failed to update msgHash in L2 cross msg", "err", err)
	}
	err = dbTx.Commit()
	if err != nil {
		// if we can not insert into DB, there must something wrong, need a on-call member handle the dababase manually
		dbTx.Rollback()
		log.Error("l2FetchAndSaveEvents: Failed to commit db transaction", "err", err)
		return err
	}

	return nil
}

func parseBackendL1EventLogs(logs []types.Log) ([]*orm.CrossMsg, []msgHashWrapper, []*orm.RelayedMsg, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l1CrossMsg []*orm.CrossMsg
	var relayedMsgs []*orm.RelayedMsg
	var msgHashes []msgHashWrapper
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L1DepositETHSig:
			event := backendabi.DepositETH{}
			err := utils.UnpackLog(backendabi.L1ETHGatewayABI, &event, "DepositETH", vlog)
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
			err := utils.UnpackLog(backendabi.L1StandardERC20GatewayABI, &event, "DepositERC20", vlog)
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
			err := utils.UnpackLog(backendabi.L1ERC721GatewayABI, &event, "DepositERC721", vlog)
			if err != nil {
				log.Warn("Failed to unpack DepositERC721 event", "err", err)
				return l1CrossMsg, msgHashes, relayedMsgs, err
			}
			l1CrossMsg = append(l1CrossMsg, &orm.CrossMsg{
				Height:       vlog.BlockNumber,
				Sender:       event.From.String(),
				Target:       event.To.String(),
				Asset:        int(orm.ERC721),
				Layer1Hash:   vlog.TxHash.Hex(),
				Layer1Token:  event.L1Token.Hex(),
				Layer2Token:  event.L2Token.Hex(),
				TokenIDs:     []int64{event.TokenID.Int64()},
				TokenAmounts: []int64{1},
			})
		case backendabi.L1BatchDepositERC721Sig:
			event := backendabi.BatchERC721MessageEvent{}
			err := utils.UnpackLog(backendabi.L1ERC721GatewayABI, &event, "BatchDepositERC721", vlog)
			if err != nil {
				log.Warn("Failed to unpack BatchDepositERC721 event", "err", err)
				return l1CrossMsg, msgHashes, relayedMsgs, err
			}
			l1CrossMsg = append(l1CrossMsg, &orm.CrossMsg{
				Height:       vlog.BlockNumber,
				Sender:       event.From.String(),
				Target:       event.To.String(),
				Asset:        int(orm.ERC721),
				Layer1Hash:   vlog.TxHash.Hex(),
				Layer1Token:  event.L1Token.Hex(),
				Layer2Token:  event.L2Token.Hex(),
				TokenIDs:     bigIntToInt64Array(event.TokenIDs),
				TokenAmounts: bigIntToIntOnes(event.TokenAmounts),
			})
		case backendabi.L1DepositERC1155Sig:
			event := backendabi.ERC1155MessageEvent{}
			err := utils.UnpackLog(backendabi.L1ERC1155GatewayABI, &event, "DepositERC1155", vlog)
			if err != nil {
				log.Warn("Failed to unpack DepositERC1155 event", "err", err)
				return l1CrossMsg, msgHashes, relayedMsgs, err
			}
			l1CrossMsg = append(l1CrossMsg, &orm.CrossMsg{
				Height:       vlog.BlockNumber,
				Sender:       event.From.String(),
				Target:       event.To.String(),
				Asset:        int(orm.ERC1155),
				Layer1Hash:   vlog.TxHash.Hex(),
				Layer1Token:  event.L1Token.Hex(),
				Layer2Token:  event.L2Token.Hex(),
				TokenIDs:     []int64{event.TokenID.Int64()},
				TokenAmounts: []int64{event.Amount.Int64()},
			})
		case backendabi.L1BatchDepositERC1155Sig:
			event := backendabi.BatchERC1155MessageEvent{}
			err := utils.UnpackLog(backendabi.L1ERC1155GatewayABI, &event, "BatchDepositERC1155", vlog)
			if err != nil {
				log.Warn("Failed to unpack BatchDepositERC1155 event", "err", err)
				return l1CrossMsg, msgHashes, relayedMsgs, err
			}
			l1CrossMsg = append(l1CrossMsg, &orm.CrossMsg{
				Height:       vlog.BlockNumber,
				Sender:       event.From.String(),
				Target:       event.To.String(),
				Asset:        int(orm.ERC1155),
				Layer1Hash:   vlog.TxHash.Hex(),
				Layer1Token:  event.L1Token.Hex(),
				Layer2Token:  event.L2Token.Hex(),
				TokenIDs:     bigIntToInt64Array(event.TokenIDs),
				TokenAmounts: bigIntToInt64Array(event.TokenAmounts),
			})
		case backendabi.L1SentMessageEventSignature:
			event := backendabi.L1SentMessageEvent{}
			err := utils.UnpackLog(backendabi.L1ScrollMessengerABI, &event, "SentMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack SentMessage event", "err", err)
				return l1CrossMsg, msgHashes, relayedMsgs, err
			}
			msgHash := utils.ComputeMessageHash(event.Sender, event.Target, event.Value, event.MessageNonce, event.Message)
			msgHashes = append(msgHashes, msgHashWrapper{
				msgHash: msgHash,
				txHash:  vlog.TxHash})
		case backendabi.L1RelayedMessageEventSignature:
			event := backendabi.L1RelayedMessageEvent{}
			err := utils.UnpackLog(backendabi.L1ScrollMessengerABI, &event, "RelayedMessage", vlog)
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

func parseBackendL2EventLogs(logs []types.Log) ([]*orm.CrossMsg, []msgHashWrapper, []*orm.RelayedMsg, error) {
	// Need use contract abi to parse event Log
	// Can only be tested after we have our contracts set up

	var l2CrossMsg []*orm.CrossMsg
	var relayedMsgs []*orm.RelayedMsg
	var msgHashes []msgHashWrapper
	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case backendabi.L2WithdrawETHSig:
			event := backendabi.DepositETH{}
			err := utils.UnpackLog(backendabi.L2ETHGatewayABI, &event, "WithdrawETH", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawETH event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, err
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
			err := utils.UnpackLog(backendabi.L2StandardERC20GatewayABI, &event, "WithdrawERC20", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawERC20 event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, err
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
			err := utils.UnpackLog(backendabi.L2ERC721GatewayABI, &event, "WithdrawERC721", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawERC721 event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, err
			}
			l2CrossMsg = append(l2CrossMsg, &orm.CrossMsg{
				Height:      vlog.BlockNumber,
				Sender:      event.From.String(),
				Target:      event.To.String(),
				Asset:       int(orm.ERC721),
				Layer2Hash:  vlog.TxHash.Hex(),
				Layer1Token: event.L1Token.Hex(),
				Layer2Token: event.L2Token.Hex(),
				TokenIDs:    []int64{event.TokenID.Int64()},
				// can only be one for a single WithdrawERC721 tx
				TokenAmounts: []int64{1},
			})
		case backendabi.L2BatchWithdrawERC721Sig:
			event := backendabi.BatchERC721MessageEvent{}
			err := utils.UnpackLog(backendabi.L2ERC721GatewayABI, &event, "BatchWithdrawERC721", vlog)
			if err != nil {
				log.Warn("Failed to unpack BatchWithdrawERC721 event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, err
			}
			l2CrossMsg = append(l2CrossMsg, &orm.CrossMsg{
				Height:       vlog.BlockNumber,
				Sender:       event.From.String(),
				Target:       event.To.String(),
				Asset:        int(orm.ERC721),
				Layer2Hash:   vlog.TxHash.Hex(),
				Layer1Token:  event.L1Token.Hex(),
				Layer2Token:  event.L2Token.Hex(),
				TokenIDs:     bigIntToInt64Array(event.TokenIDs),
				TokenAmounts: bigIntToIntOnes(event.TokenIDs),
			})
		case backendabi.L2WithdrawERC1155Sig:
			event := backendabi.ERC1155MessageEvent{}
			err := utils.UnpackLog(backendabi.L2ERC1155GatewayABI, &event, "WithdrawERC1155", vlog)
			if err != nil {
				log.Warn("Failed to unpack WithdrawERC1155 event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, err
			}
			l2CrossMsg = append(l2CrossMsg, &orm.CrossMsg{
				Height:       vlog.BlockNumber,
				Sender:       event.From.String(),
				Target:       event.To.String(),
				Asset:        int(orm.ERC1155),
				Layer2Hash:   vlog.TxHash.Hex(),
				Layer1Token:  event.L1Token.Hex(),
				Layer2Token:  event.L2Token.Hex(),
				TokenIDs:     []int64{event.TokenID.Int64()},
				TokenAmounts: []int64{event.Amount.Int64()},
			})
		case backendabi.L2BatchWithdrawERC1155Sig:
			event := backendabi.BatchERC1155MessageEvent{}
			err := utils.UnpackLog(backendabi.L2ERC1155GatewayABI, &event, "BatchWithdrawERC1155", vlog)
			if err != nil {
				log.Warn("Failed to unpack BatchWithdrawERC1155 event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, err
			}
			l2CrossMsg = append(l2CrossMsg, &orm.CrossMsg{
				Height:       vlog.BlockNumber,
				Sender:       event.From.String(),
				Target:       event.To.String(),
				Asset:        int(orm.ERC1155),
				Layer2Hash:   vlog.TxHash.Hex(),
				Layer1Token:  event.L1Token.Hex(),
				Layer2Token:  event.L2Token.Hex(),
				TokenIDs:     bigIntToInt64Array(event.TokenIDs),
				TokenAmounts: bigIntToInt64Array(event.TokenAmounts),
			})
		case backendabi.L2SentMessageEventSignature:
			event := backendabi.L2SentMessageEvent{}
			err := utils.UnpackLog(backendabi.L2ScrollMessengerABI, &event, "SentMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack SentMessage event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, err
			}
			msgHash := utils.ComputeMessageHash(event.Sender, event.Target, event.Value, event.MessageNonce, event.Message)
			msgHashes = append(msgHashes, msgHashWrapper{
				msgHash: msgHash,
				txHash:  vlog.TxHash})
		case backendabi.L2RelayedMessageEventSignature:
			event := backendabi.L2RelayedMessageEvent{}
			err := utils.UnpackLog(backendabi.L2ScrollMessengerABI, &event, "RelayedMessage", vlog)
			if err != nil {
				log.Warn("Failed to unpack RelayedMessage event", "err", err)
				return l2CrossMsg, msgHashes, relayedMsgs, err
			}
			relayedMsgs = append(relayedMsgs, &orm.RelayedMsg{
				MsgHash:    event.MessageHash.String(),
				Height:     vlog.BlockNumber,
				Layer2Hash: vlog.TxHash.Hex(),
			})

		}

	}
	return l2CrossMsg, msgHashes, relayedMsgs, nil
}

func updateL1CrossMsgMsgHash(ctx context.Context, dbTx *sqlx.Tx, database db.OrmFactory, msgHashes []msgHashWrapper) error {
	for _, msgHash := range msgHashes {
		err := database.UpdateL1CrossMsgHashDBTx(ctx, dbTx, msgHash.txHash, msgHash.msgHash)
		if err != nil {
			log.Error("updateL1CrossMsgMsgHash: can not update layer1 cross msg MsgHash", "layer1 hash", msgHash.txHash, "err", err)
			continue
		}
	}
	return nil
}

func updateL2CrossMsgMsgHash(ctx context.Context, dbTx *sqlx.Tx, database db.OrmFactory, msgHashes []msgHashWrapper) error {
	for _, msgHash := range msgHashes {
		err := database.UpdateL2CrossMsgHashDBTx(ctx, dbTx, msgHash.txHash, msgHash.msgHash)
		if err != nil {
			log.Error("updateL2CrossMsgMsgHash: can not update layer2 cross msg MsgHash", "layer2 hash", msgHash.txHash, "err", err)
			continue
		}
	}
	return nil
}

func bigIntToIntOnes(bigIntSlice []*big.Int) []int64 {
	int64Slice := make([]int64, len(bigIntSlice))
	for i, _ := range bigIntSlice {
		int64Slice[i] = 1
	}
	return int64Slice
}

func bigIntToInt64Array(bigIntSlice []*big.Int) []int64 {
	int64Slice := make([]int64, len(bigIntSlice))
	for i, bi := range bigIntSlice {
		int64Slice[i] = bi.Int64()
	}
	return int64Slice
}
