package crossmsg

import (
	"bridge-history-api/db/orm"
	"context"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

// ReorgHandling handles reorg function type
type ReorgHandling func(ctx context.Context, reorgHeight uint64, db *gorm.DB) error

func reverseArray(arr []*types.Header) []*types.Header {
	for i := 0; i < len(arr)/2; i++ {
		j := len(arr) - i - 1
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}

// IsParentAndChild match the child header ParentHash with parent header Hash
func IsParentAndChild(parentHeader *types.Header, header *types.Header) bool {
	return header.ParentHash == parentHeader.Hash()
}

// MergeAddIntoHeaderList merges two header lists, if exceed the max length then drop the oldest entries
func MergeAddIntoHeaderList(baseArr, extraArr []*types.Header, maxLength int) []*types.Header {
	mergedArr := append(baseArr, extraArr...)
	if len(mergedArr) <= maxLength {
		return mergedArr
	}

	startIndex := len(mergedArr) - maxLength
	return mergedArr[startIndex:]
}

// BackwardFindReorgBlock finds the reorg block by backward search
func BackwardFindReorgBlock(ctx context.Context, headers []*types.Header, client *ethclient.Client, lastHeader *types.Header) (int, bool, []*types.Header) {
	maxStep := len(headers)
	backwardHeaderList := []*types.Header{lastHeader}
	for iterRound := 0; iterRound < maxStep; iterRound++ {
		header, err := client.HeaderByHash(ctx, lastHeader.ParentHash)
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
		lastHeader = header
	}
	return -1, false, nil
}

// L1ReorgHandling handles l1 reorg
func L1ReorgHandling(ctx context.Context, reorgHeight uint64, db *gorm.DB) error {
	l1CrossMsgOrm := orm.NewL1CrossMsg(db)
	relayedOrm := orm.NewRelayedMsg(db)
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := l1CrossMsgOrm.DeleteL1CrossMsgAfterHeight(ctx, reorgHeight, tx); err != nil {
			log.Error("delete l1 cross msg from height", "height", reorgHeight, "err", err)
			return err
		}
		if err := relayedOrm.DeleteL1RelayedHashAfterHeight(ctx, reorgHeight, tx); err != nil {
			log.Error("delete l1 relayed msg from height", "height", reorgHeight, "err", err)
			return err
		}
		return nil
	})
	if err != nil {
		log.Crit("l1 reorg handling failed", "err", err)
	}
	return err
}

// L2ReorgHandling handles l2 reorg
func L2ReorgHandling(ctx context.Context, reorgHeight uint64, db *gorm.DB) error {
	l2CrossMsgOrm := orm.NewL2CrossMsg(db)
	relayedOrm := orm.NewRelayedMsg(db)
	l2SentMsgOrm := orm.NewL2SentMsg(db)
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := l2CrossMsgOrm.DeleteL2CrossMsgFromHeight(ctx, reorgHeight, tx); err != nil {
			log.Error("delete l2 cross msg from height", "height", reorgHeight, "err", err)
			return err
		}
		if err := relayedOrm.DeleteL2RelayedHashAfterHeight(ctx, reorgHeight, tx); err != nil {
			log.Error("delete l2 relayed msg from height", "height", reorgHeight, "err", err)
			return err
		}
		if err := l2SentMsgOrm.DeleteL2SentMsgAfterHeight(ctx, reorgHeight, tx); err != nil {
			log.Error("delete l2 sent msg from height", "height", reorgHeight, "err", err)
			return err
		}
		return nil
	})
	if err != nil {
		log.Crit("l2 reorg handling failed", "err", err)
	}
	return err
}
