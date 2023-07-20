package orm

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

// RelayedMsg is the struct for relayed_msg table
type RelayedMsg struct {
	db *gorm.DB `gorm:"column:-"`

	ID         uint64         `json:"id" gorm:"column:id"`
	MsgHash    string         `json:"msg_hash" gorm:"column:msg_hash"`
	Height     uint64         `json:"height" gorm:"column:height"`
	Layer1Hash string         `json:"layer1_hash" gorm:"column:layer1_hash;default:''"`
	Layer2Hash string         `json:"layer2_hash" gorm:"column:layer2_hash;default:''"`
	CreatedAt  *time.Time     `json:"created_at" gorm:"column:created_at"`
	UpdatedAt  *time.Time     `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewRelayedMsg create an NewRelayedMsg instance
func NewRelayedMsg(db *gorm.DB) *RelayedMsg {
	return &RelayedMsg{db: db}
}

// TableName returns the table name for the RelayedMsg model.
func (*RelayedMsg) TableName() string {
	return "relayed_msg"
}

// BatchInsertRelayedMsgDBTx batch insert relayed msg into db and return the transaction
func (r *RelayedMsg) BatchInsertRelayedMsg(ctx context.Context, messages []*RelayedMsg, dbTx ...*gorm.DB) error {
	if len(messages) == 0 {
		return nil
	}
	db := r.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	db.WithContext(ctx)
	err := db.Model(&RelayedMsg{}).Create(&messages).Error
	if err != nil {
		l2hashes := make([]string, 0, len(messages))
		l1hashes := make([]string, 0, len(messages))
		heights := make([]uint64, 0, len(messages))
		for _, msg := range messages {
			l2hashes = append(l2hashes, msg.Layer2Hash)
			l1hashes = append(l1hashes, msg.Layer1Hash)
			heights = append(heights, msg.Height)
		}
		log.Error("failed to insert l2 sent messages", "l2hashes", l2hashes, "l1hashes", l1hashes, "heights", heights, "err", err)
	}
	return err
}

// GetRelayedMsgByHash get relayed msg by hash
func (r *RelayedMsg) GetRelayedMsgByHash(msgHash string) (*RelayedMsg, error) {
	result := &RelayedMsg{}
	err := r.db.Model(&RelayedMsg{}).
		Where("msg_hash = ?", msgHash).
		First(&result).
		Error
	return result, err
}

// GetLatestRelayedHeightOnL1 get latest relayed height on l1
func (r *RelayedMsg) GetLatestRelayedHeightOnL1() (uint64, error) {
	result := &RelayedMsg{}
	err := r.db.Model(&RelayedMsg{}).
		Select("height").
		Where("layer1_hash != ''").
		Order("height DESC").
		Limit(1).
		Scan(&result).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return result.Height, err
}

// GetLatestRelayedHeightOnL2 get latest relayed height on l2
func (r *RelayedMsg) GetLatestRelayedHeightOnL2() (uint64, error) {
	result := &RelayedMsg{}
	err := r.db.Model(&RelayedMsg{}).
		Select("height").
		Where("layer2_hash != ''").
		Order("height DESC").
		Limit(1).
		Scan(&result).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return result.Height, nil
}

// DeleteL1RelayedHashAfterHeight delete l1 relayed hash after height
func (r *RelayedMsg) DeleteL1RelayedHashAfterHeight(ctx context.Context, height uint64, dbTx ...*gorm.DB) error {
	db := r.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	db.WithContext(ctx)
	err := db.Model(&RelayedMsg{}).
		Where("height > ? AND layer1_hash != ''", height).
		Update("deleted_at", gorm.Expr("current_timestamp")).Error
	return err
}

// DeleteL2RelayedHashAfterHeight delete l2 relayed hash after heights
func (r *RelayedMsg) DeleteL2RelayedHashAfterHeight(ctx context.Context, height uint64, dbTx ...*gorm.DB) error {
	db := r.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	db.WithContext(ctx)
	err := db.Model(&RelayedMsg{}).
		Where("height > ? AND layer2_hash != ''", height).
		Update("deleted_at", gorm.Expr("current_timestamp")).Error
	return err
}
