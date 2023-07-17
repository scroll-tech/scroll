package crossmsg

import (
	"context"
	"math/big"

	geth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	backendabi "bridge-history-api/abi"
	"bridge-history-api/db"
	"bridge-history-api/utils"
)

// Todo : read from config
var (
	// the number of blocks fetch per round
	fetchLimit = int64(3000)
)

// FetchAndSave is a function type that fetches events from blockchain and saves them to database
type FetchAndSave func(ctx context.Context, client *ethclient.Client, database db.OrmFactory, from int64, to int64, addressList []common.Address) error

// GetLatestProcessed is a function type that gets the latest processed block height from database
type GetLatestProcessed func(db db.OrmFactory) (int64, error)

// FetchEventWorker defines worker with fetch and save function, processed number getter, and name
type FetchEventWorker struct {
	F    FetchAndSave
	G    GetLatestProcessed
	Name string
}

// GetLatestL1ProcessedHeight get L1 the latest processed height
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
	}
	return relayedHeight, nil
}

// GetLatestL2ProcessedHeight get L2 latest processed height
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
	l2SentHeight, err := db.GetLatestSentMsgHeightOnL2()
	if err != nil {
		log.Error("failed to get L2 sent message processed height", "err", err)
		return 0, err
	}
	maxHeight := crossHeight
	if maxHeight < relayedHeight {
		maxHeight = relayedHeight
	}
	if maxHeight < l2SentHeight {
		maxHeight = l2SentHeight
	}
	return maxHeight, nil
}

// L1FetchAndSaveEvents fetch and save events on L1
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
	depositL1CrossMsgs, relayedMsg, err := utils.ParseBackendL1EventLogs(logs)
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
		if rollBackErr := dbTx.Rollback(); rollBackErr != nil {
			log.Error("dbTx Rollback failed", "err", rollBackErr)
		}
		log.Crit("l1FetchAndSaveEvents: Failed to insert cross msg event logs", "err", err)
	}

	err = database.BatchInsertRelayedMsgDBTx(dbTx, relayedMsg)
	if err != nil {
		if rollBackErr := dbTx.Rollback(); rollBackErr != nil {
			log.Error("dbTx Rollback failed", "err", rollBackErr)
		}
		log.Crit("l1FetchAndSaveEvents: Failed to insert relayed message event logs", "err", err)
	}
	err = dbTx.Commit()
	if err != nil {
		// if we can not insert into DB, there must something wrong, need a on-call member handle the dababase manually
		if rollBackErr := dbTx.Rollback(); rollBackErr != nil {
			log.Error("dbTx Rollback failed", "err", rollBackErr)
		}
		log.Error("l1FetchAndSaveEvents: Failed to commit db transaction", "err", err)
		return err
	}

	return nil
}

// L2FetchAndSaveEvents fetche and save events on L2
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
	depositL2CrossMsgs, relayedMsg, l2SentMsgs, err := utils.ParseBackendL2EventLogs(logs)
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
		if rollBackErr := dbTx.Rollback(); rollBackErr != nil {
			log.Error("dbTx Rollback failed", "err", rollBackErr)
		}
		log.Crit("l2FetchAndSaveEvents: Failed to insert cross msg event logs", "err", err)
	}

	err = database.BatchInsertRelayedMsgDBTx(dbTx, relayedMsg)
	if err != nil {
		if rollBackErr := dbTx.Rollback(); rollBackErr != nil {
			log.Error("dbTx Rollback failed", "err", rollBackErr)
		}
		log.Crit("l2FetchAndSaveEvents: Failed to insert relayed message event logs", "err", err)
	}

	err = database.BatchInsertL2SentMsgDBTx(dbTx, l2SentMsgs)
	if err != nil {
		if rollBackErr := dbTx.Rollback(); rollBackErr != nil {
			log.Error("dbTx Rollback failed", "err", rollBackErr)
		}
		log.Crit("l2FetchAndSaveEvents: Failed to insert l2 sent message", "err", err)
	}

	err = dbTx.Commit()
	if err != nil {
		// if we can not insert into DB, there must something wrong, need a on-call member handle the dababase manually
		if rollBackErr := dbTx.Rollback(); rollBackErr != nil {
			log.Error("dbTx Rollback failed", "err", rollBackErr)
		}
		log.Error("l2FetchAndSaveEvents: Failed to commit db transaction", "err", err)
		return err
	}

	return nil
}

// FetchAndSaveBatchIndex fetche and save batch index
func FetchAndSaveBatchIndex(ctx context.Context, client *ethclient.Client, database db.OrmFactory, from int64, to int64, scrollChainAddr common.Address) error {
	query := geth.FilterQuery{
		FromBlock: big.NewInt(from), // inclusive
		ToBlock:   big.NewInt(to),   // inclusive
		Addresses: []common.Address{scrollChainAddr},
		Topics:    make([][]common.Hash, 1),
	}
	query.Topics[0] = make([]common.Hash, 1)
	query.Topics[0][0] = backendabi.L1CommitBatchEventSignature
	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		log.Warn("Failed to get batch commit event logs", "err", err)
		return err
	}
	rollupBatches, err := utils.ParseBatchInfoFromScrollChain(ctx, client, logs)
	if err != nil {
		log.Error("FetchAndSaveBatchIndex: Failed to parse batch commit msg event logs", "err", err)
		return err
	}
	dbTx, err := database.Beginx()
	if err != nil {
		log.Error("FetchAndSaveBatchIndex: Failed to begin db transaction", "err", err)
		return err
	}
	err = database.BatchInsertRollupBatchDBTx(dbTx, rollupBatches)
	if err != nil {
		if rollBackErr := dbTx.Rollback(); rollBackErr != nil {
			log.Error("dbTx Rollback failed", "err", rollBackErr)
		}
		log.Crit("FetchAndSaveBatchIndex: Failed to insert batch commit msg event logs", "err", err)
	}
	err = dbTx.Commit()
	if err != nil {
		// if we can not insert into DB, there must something wrong, need a on-call member handle the dababase manually
		if rollBackErr := dbTx.Rollback(); rollBackErr != nil {
			log.Error("dbTx Rollback failed", "err", rollBackErr)
		}
		log.Error("FetchAndSaveBatchIndex: Failed to commit db transaction", "err", err)
		return err
	}
	return nil
}
