package orm

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

type L1CrossMsg struct {
	*CrossMsg
}

// NewL1CrossMsg create an L1CrossMsg instance
func NewL1CrossMsg(db *gorm.DB) *L1CrossMsg {
	return &L1CrossMsg{&CrossMsg{db: db}}
}

func (l *L1CrossMsg) GetL1CrossMsgByHash(l1Hash common.Hash) (*CrossMsg, error) {
	result := &CrossMsg{}
	err := l.db.Where("layer1_hash = ? AND msg_type = ? AND deleted_at IS NULL", l1Hash.String(), Layer1Msg).First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

func (l *L1CrossMsg) BatchInsertL1CrossMsgDBTx(dbTx *gorm.DB, messages []*CrossMsg) error {
	if len(messages) == 0 {
		return nil
	}
	err := dbTx.Model(&CrossMsg{}).Create(&messages).Error
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

// UpdateL1CrossMsgHashDBTx update l1 cross msg hash in db, no need to check msg_type since layer1_hash wont be empty if its layer1 msg
func (l *L1CrossMsg) UpdateL1CrossMsgHashDBTx(ctx context.Context, dbTx *gorm.DB, l1Hash, msgHash common.Hash) error {
	err := l.db.Model(&CrossMsg{}).Where("layer1_hash = ? AND deleted_at IS NULL", l1Hash.Hex()).Update("msg_hash", msgHash.Hex()).Error
	return err

}

func (l *L1CrossMsg) GetLatestL1ProcessedHeight() (int64, error) {
	var height int64
	err := l.db.Table("cross_message").Where("msg_type = ? AND deleted_at IS NULL", Layer1Msg).Order("id DESC").Limit(1).Select("height").Scan(&height).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return height, err
}

func (l *L1CrossMsg) DeleteL1CrossMsgAfterHeightDBTx(dbTx *gorm.DB, height int64) error {
	err := dbTx.Table("cross_message").
		Where("height > ? AND msg_type = ?", height, Layer1Msg).
		Updates(map[string]interface{}{"deleted_at": gorm.Expr("current_timestamp")}).Error
	return err
}

func (l *L1CrossMsg) UpdateL1BlockTimestamp(height uint64, timestamp time.Time) error {
	err := l.db.Table("cross_message").
		Where("height = ? AND msg_type = ? AND deleted_at IS NULL", height, Layer1Msg).
		Update("block_timestamp", timestamp).Error
	return err
}

func (l *L1CrossMsg) GetL1EarliestNoBlockTimestampHeight() (uint64, error) {
	var height int64
	err := l.db.Table("cross_message").
		Where("block_timestamp IS NULL AND msg_type = ? AND deleted_at IS NULL", Layer1Msg).
		Order("height ASC").
		Limit(1).
		Select("height").
		Scan(&height).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return uint64(height), err
}
