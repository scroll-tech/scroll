package cross_msg

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"bridge-history-api/db"
)

type ReorgHandling func(ctx context.Context, reorgHeight int64, db db.OrmFactory) error

func reverseArray(arr []*types.Header) []*types.Header {
	for i := 0; i < len(arr)/2; i++ {
		j := len(arr) - i - 1
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}

func IsParentAndChild(parentHeader *types.Header, header *types.Header) bool {
	return header.ParentHash == parentHeader.Hash()
}

func MergeAddIntoHeaderList(baseArr, extraArr []*types.Header, maxLength int) []*types.Header {
	mergedArr := append(baseArr, extraArr...)
	if len(mergedArr) <= maxLength {
		return mergedArr
	}

	startIndex := len(mergedArr) - maxLength
	return mergedArr[startIndex:]
}

func BackwardFindReorgBlock(ctx context.Context, headers []*types.Header, client *ethclient.Client, header *types.Header) (int, bool, []*types.Header) {
	maxStep := len(headers)
	backwardHeaderList := []*types.Header{header}
	for iterRound := 0; iterRound < maxStep; iterRound++ {
		header, err := client.HeaderByHash(ctx, header.ParentHash)
		if err != nil {
			log.Error("BackwardFindReorgBlock failed", "error", err)
			return -1, false, nil
		}
		backwardHeaderList = append(backwardHeaderList, header)
		for j := len(headers) - 1; j >= 0; j-- {
			if IsParentAndChild(headers[j], header) {
				backwardHeaderList = reverseArray(backwardHeaderList)
				return j, true, backwardHeaderList
			}
		}
	}
	return -1, false, nil
}

func L1ReorgHandling(ctx context.Context, reorgHeight int64, db db.OrmFactory) error {
	dbTx, err := db.Beginx()
	if err != nil {
		log.Crit("begin db tx failed", "err", err)
	}
	err = db.DeleteL1CrossMsgAfterHeightDBTx(dbTx, reorgHeight)
	if err != nil {
		dbTx.Rollback()
		log.Crit("delete l1 cross msg from height", "height", reorgHeight, "err", err)
	}
	err = db.DeleteL1RelayedHashAfterHeightDBTx(dbTx, reorgHeight)
	if err != nil {
		dbTx.Rollback()
		log.Crit("delete l1 relayed hash from height", "height", reorgHeight, "err", err)
	}
	err = dbTx.Commit()
	if err != nil {
		dbTx.Rollback()
		log.Error("commit tx failed", "err", err)
		return err
	}
	return nil
}

func L2ReorgHandling(ctx context.Context, reorgHeight int64, db db.OrmFactory) error {
	dbTx, err := db.Beginx()
	if err != nil {
		dbTx.Rollback()
		log.Crit("begin db tx failed", "err", err)
	}
	err = db.DeleteL2CrossMsgFromHeightDBTx(dbTx, reorgHeight)
	if err != nil {
		dbTx.Rollback()
		log.Crit("delete l2 cross msg from height", "height", reorgHeight, "err", err)
	}
	err = db.DeleteL2RelayedHashAfterHeightDBTx(dbTx, reorgHeight)
	if err != nil {
		dbTx.Rollback()
		log.Crit("delete l2 relayed hash from height", "height", reorgHeight, "err", err)
	}
	err = db.DeleteL2SentMsgAfterHeightDBTx(dbTx, reorgHeight)
	if err != nil {
		dbTx.Rollback()
		log.Crit("delete l2 sent msg from height", "height", reorgHeight, "err", err)
	}
	err = dbTx.Commit()
	if err != nil {
		dbTx.Rollback()
		log.Error("commit tx failed", "err", err)
		return err
	}
	return nil
}
