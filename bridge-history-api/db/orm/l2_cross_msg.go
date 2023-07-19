package orm

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

// L2CrossMsg specify the orm operations for layer2 cross messages
type L2CrossMsg struct {
	*CrossMsg
}

// NewL2CrossMsg create an NewL2CrossMsg instance
func NewL2CrossMsg(db *gorm.DB) *L2CrossMsg {
	return &L2CrossMsg{&CrossMsg{db: db}}
}

// GetL2CrossMsgByHash returns layer2 cross message by given hash
func (l *L2CrossMsg) GetL2CrossMsgByHash(l2Hash common.Hash) (*CrossMsg, error) {
	result := &CrossMsg{}
	err := l.db.Table("cross_message").Where("layer2_hash = ? AND msg_type = ? AND deleted_at IS NULL", l2Hash.String(), Layer1Msg).First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
	}
	return result, err
}

// GetL2CrossMsgByAddress returns all layer2 cross messages under given address
// Warning: return empty slice if no data found
func (l *L2CrossMsg) GetL2CrossMsgByAddress(sender common.Address) ([]*CrossMsg, error) {
	var results []*CrossMsg
	err := l.db.Table("cross_message").Where("sender = ? AND msg_type = ? AND deleted_at IS NULL", sender.String(), Layer2Msg).
		Find(&results).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
	}
	return results, err

}

// DeleteL2CrossMsgFromHeightDBTx delete layer2 cross messages from given height
func (l *L2CrossMsg) DeleteL2CrossMsgFromHeightDBTx(dbTx *gorm.DB, height int64) (*gorm.DB, error) {
	err := dbTx.Table("cross_message").
		Where("height > ? AND msg_type = ?", height, Layer2Msg).
		Update("deleted_at", gorm.Expr("current_timestamp")).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return dbTx, nil
		}
	}
	return dbTx, err
}

// BatchInsertL2CrossMsgDBTx batch insert layer2 cross messages
func (l *L2CrossMsg) BatchInsertL2CrossMsgDBTx(dbTx *gorm.DB, messages []*CrossMsg) (*gorm.DB, error) {
	if len(messages) == 0 {
		return dbTx, nil
	}
	err := dbTx.Model(&CrossMsg{}).Table("cross_message").Create(&messages).Error
	if err != nil {
		l2hashes := make([]string, 0, len(messages))
		heights := make([]uint64, 0, len(messages))
		for _, msg := range messages {
			l2hashes = append(l2hashes, msg.Layer2Hash)
			heights = append(heights, msg.Height)
		}
		log.Error("failed to insert l2 cross messages", "l2hashes", l2hashes, "heights", heights, "err", err)
	}
	return dbTx, err
}

// UpdateL2CrossMsgHashDBTx update layer2 cross message hash
func (l *L2CrossMsg) UpdateL2CrossMsgHashDBTx(ctx context.Context, dbTx *gorm.DB, l2Hash, msgHash common.Hash) error {
	err := dbTx.Table("cross_message").
		Where("layer2_hash = ? AND deleted_at IS NULL", l2Hash.String()).
		Update("msg_hash", msgHash.String()).
		Error
	return err
}

// UpdateL2CrossMsgHash update layer2 cross message hash
func (l *L2CrossMsg) UpdateL2CrossMsgHash(ctx context.Context, l2Hash, msgHash common.Hash) error {
	err := l.db.Table("cross_message").
		Where("layer2_hash = ? AND deleted_at IS NULL", l2Hash.String()).
		UpdateColumn("msg_hash", msgHash.String()).
		Error
	return err
}

// GetLatestL2ProcessedHeight returns the latest processed height of layer2 cross messages
func (l *L2CrossMsg) GetLatestL2ProcessedHeight() (int64, error) {
	var height int64
	err := l.db.Table("cross_message").
		Where("msg_type = ? AND deleted_at IS NULL", Layer2Msg).
		Order("id DESC").
		Limit(1).
		Select("height").
		Scan(&height).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return height, err
}

// UpdateL2BlockTimestamp update layer2 cross message block timestamp
func (l *L2CrossMsg) UpdateL2BlockTimestamp(height uint64, timestamp time.Time) error {
	err := l.db.Table("cross_message").
		Where("height = ? AND msg_type = ? AND deleted_at IS NULL", height, Layer2Msg).
		Update("block_timestamp", timestamp).Error
	return err
}

// GetL2EarliestNoBlockTimestampHeight returns the earliest layer2 cross message height which has no block timestamp
func (l *L2CrossMsg) GetL2EarliestNoBlockTimestampHeight() (uint64, error) {
	var height int64
	err := l.db.Table("cross_message").
		Where("block_timestamp IS NULL AND msg_type = ? AND deleted_at IS NULL", Layer2Msg).
		Order("height ASC").
		Select("height").
		Limit(1).
		Scan(&height).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return uint64(height), err
}

// GetL2CrossMsgByMsgHashList returns layer2 cross messages under given msg hashes
func (l *L2CrossMsg) GetL2CrossMsgByMsgHashList(msgHashList []string) ([]*CrossMsg, error) {
	var results []*CrossMsg
	err := l.db.Table("cross_message").
		Where("msg_hash IN (?) AND msg_type = ? AND deleted_at IS NULL", msgHashList, Layer2Msg).
		Find(&results).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
	}
	if len(results) == 0 {
		log.Debug("no L2CrossMsg under given msg hashes", "msg hash list", msgHashList)
	}
	return results, err
}
