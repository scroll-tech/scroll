package orm

import (
	"time"

	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

// RelayedMsg is the struct for relayed_msg table
type RelayedMsg struct {
	db *gorm.DB `gorm:"column:-"`

	ID         uint64     `json:"id" gorm:"column:id"`
	MsgHash    string     `json:"msg_hash" gorm:"column:msg_hash"`
	Height     uint64     `json:"height" gorm:"column:height"`
	Layer1Hash string     `json:"layer1_hash" gorm:"column:layer1_hash;default:''"`
	Layer2Hash string     `json:"layer2_hash" gorm:"column:layer2_hash;default:''"`
	CreatedAt  *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt  *time.Time `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt  *time.Time `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewRelayedMsg create an NewRelayedMsg instance
func NewRelayedMsg(db *gorm.DB) *RelayedMsg {
	return &RelayedMsg{db: db}
}

func (r *RelayedMsg) BatchInsertRelayedMsgDBTx(dbTx *gorm.DB, messages []*RelayedMsg) (*gorm.DB, error) {
	if len(messages) == 0 {
		return dbTx, nil
	}

	err := dbTx.Model(&RelayedMsg{}).Create(&messages).Error
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
	return dbTx, err
}

func (r *RelayedMsg) GetRelayedMsgByHash(msgHash string) (*RelayedMsg, error) {
	result := &RelayedMsg{}
	err := r.db.Table("relayed_msg").
		Select("msg_hash, height, layer1_hash, layer2_hash").
		Where("msg_hash = ? AND deleted_at IS NULL", msgHash).
		First(&result).
		Error
	return result, err
}

func (r *RelayedMsg) GetLatestRelayedHeightOnL1() (int64, error) {
	var height int64
	err := r.db.Table("relayed_msg").
		Select("height").
		Where("layer1_hash != '' AND deleted_at IS NULL").
		Order("height DESC").
		Limit(1).
		Scan(&height).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return height, err
}

func (r *RelayedMsg) GetLatestRelayedHeightOnL2() (int64, error) {
	var height int64
	err := r.db.Table("relayed_msg").
		Select("height").
		Where("layer2_hash != '' AND deleted_at IS NULL").
		Order("height DESC").
		Limit(1).
		Scan(&height).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return height, nil
}

func (r *RelayedMsg) DeleteL1RelayedHashAfterHeightDBTx(dbTx *gorm.DB, height int64) (*gorm.DB, error) {
	err := dbTx.Table("relayed_msg").
		Where("height > ? AND layer1_hash != ''", height).
		Update("deleted_at", gorm.Expr("current_timestamp")).Error
	return dbTx, err

}

func (r *RelayedMsg) DeleteL2RelayedHashAfterHeightDBTx(dbTx *gorm.DB, height int64) (*gorm.DB, error) {
	err := dbTx.Table("relayed_msg").
		Where("height > ? AND layer2_hash != ''", height).
		Update("deleted_at", gorm.Expr("current_timestamp")).Error
	return dbTx, err
}
