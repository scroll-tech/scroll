package orm

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

// L1CrossMsg specify the orm operations for layer1 cross messages
type L1CrossMsg struct {
	*CrossMsg
}

// NewL1CrossMsg create an L1CrossMsg instance
func NewL1CrossMsg(db *gorm.DB) *L1CrossMsg {
	return &L1CrossMsg{&CrossMsg{db: db}}
}

// GetL1CrossMsgByHash returns layer1 cross message by given hash
func (l *L1CrossMsg) GetL1CrossMsgByHash(l1Hash common.Hash) (*CrossMsg, error) {
	result := &CrossMsg{}
	err := l.db.Model(&CrossMsg{}).Where("layer1_hash = ? AND msg_type = ?", l1Hash.String(), Layer1Msg).First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
	}
	return result, err
}

// GetLatestL1ProcessedHeight returns the latest processed height of layer1 cross messages
func (l *L1CrossMsg) GetLatestL1ProcessedHeight() (uint64, error) {
	result := &CrossMsg{}
	err := l.db.Model(&CrossMsg{}).Where("msg_type = ?", Layer1Msg).
		Order("id DESC").
		Limit(1).
		Select("height").
		First(&result).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return result.Height, err
}

// GetL1EarliestNoBlockTimestampHeight returns the earliest layer1 cross message height which has no block timestamp
func (l *L1CrossMsg) GetL1EarliestNoBlockTimestampHeight() (uint64, error) {
	result := &CrossMsg{}
	err := l.db.Model(&CrossMsg{}).
		Where("block_timestamp IS NULL AND msg_type = ?", Layer1Msg).
		Order("height ASC").
		Limit(1).
		Select("height").
		First(&result).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return result.Height, err
}

// BatchInsertL1CrossMsg batch insert layer1 cross messages into db
func (l *L1CrossMsg) BatchInsertL1CrossMsg(ctx context.Context, messages []*CrossMsg, dbTx ...*gorm.DB) error {
	if len(messages) == 0 {
		return nil
	}
	db := l.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	db.WithContext(ctx)
	err := db.Model(&CrossMsg{}).Model(&CrossMsg{}).Create(&messages).Error
	if err != nil {
		l1hashes := make([]string, 0, len(messages))
		heights := make([]uint64, 0, len(messages))
		for _, msg := range messages {
			l1hashes = append(l1hashes, msg.Layer1Hash)
			heights = append(heights, msg.Height)
		}
		log.Error("failed to insert l1 cross messages", "l1hashes", l1hashes, "heights", heights, "err", err)
	}
	return err
}

// UpdateL1CrossMsgHash update l1 cross msg hash in db, no need to check msg_type since layer1_hash wont be empty if its layer1 msg
func (l *L1CrossMsg) UpdateL1CrossMsgHash(ctx context.Context, l1Hash, msgHash common.Hash, dbTx ...*gorm.DB) error {
	db := l.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	db.WithContext(ctx)
	err := l.db.Model(&CrossMsg{}).Model(&CrossMsg{}).Where("layer1_hash = ?", l1Hash.Hex()).Update("msg_hash", msgHash.Hex()).Error
	return err

}

// UpdateL1BlockTimestamp update layer1 block timestamp
func (l *L1CrossMsg) UpdateL1BlockTimestamp(height uint64, timestamp time.Time) error {
	err := l.db.Model(&CrossMsg{}).
		Where("height = ? AND msg_type = ?", height, Layer1Msg).
		Update("block_timestamp", timestamp).Error
	return err
}

// DeleteL1CrossMsgAfterHeight soft delete layer1 cross messages after given height
func (l *L1CrossMsg) DeleteL1CrossMsgAfterHeight(ctx context.Context, height uint64, dbTx ...*gorm.DB) error {
	db := l.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	db.WithContext(ctx)
	result := db.Delete(&CrossMsg{}, "height > ? AND msg_type = ?", height, Layer1Msg)
	return result.Error
}
