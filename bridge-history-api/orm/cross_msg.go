package orm

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

// AssetType can be ETH/ERC20/ERC1155/ERC721
type AssetType int

// MsgType can be layer1/layer2 msg
type MsgType int

func (a AssetType) String() string {
	switch a {
	case ETH:
		return "ETH"
	case ERC20:
		return "ERC20"
	case ERC1155:
		return "ERC1155"
	case ERC721:
		return "ERC721"
	}
	return "Unknown Asset Type"
}

const (
	// ETH = 0
	ETH AssetType = iota
	// ERC20 = 1
	ERC20
	// ERC721 = 2
	ERC721
	// ERC1155 = 3
	ERC1155
)

const (
	// UnknownMsg = 0
	UnknownMsg MsgType = iota
	// Layer1Msg = 1
	Layer1Msg
	// Layer2Msg = 2
	Layer2Msg
)

// CrossMsg represents a cross message from layer 1 to layer 2
type CrossMsg struct {
	db *gorm.DB `gorm:"column:-"`

	ID           uint64         `json:"id" gorm:"column:id"`
	MsgHash      string         `json:"msg_hash" gorm:"column:msg_hash"`
	Height       uint64         `json:"height" gorm:"column:height"`
	Sender       string         `json:"sender" gorm:"column:sender"`
	Target       string         `json:"target" gorm:"column:target"`
	Amount       string         `json:"amount" gorm:"column:amount"`
	Layer1Hash   string         `json:"layer1_hash" gorm:"column:layer1_hash;default:''"`
	Layer2Hash   string         `json:"layer2_hash" gorm:"column:layer2_hash;default:''"`
	Layer1Token  string         `json:"layer1_token" gorm:"column:layer1_token;default:''"`
	Layer2Token  string         `json:"layer2_token" gorm:"column:layer2_token;default:''"`
	TokenIDs     string         `json:"token_ids" gorm:"column:token_ids;default:''"`
	TokenAmounts string         `json:"token_amounts" gorm:"column:token_amounts;default:''"`
	Asset        int            `json:"asset" gorm:"column:asset"`
	MsgType      int            `json:"msg_type" gorm:"column:msg_type"`
	Timestamp    *time.Time     `json:"timestamp" gorm:"column:block_timestamp;default;NULL"`
	CreatedAt    *time.Time     `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    *time.Time     `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// TableName returns the table name for the CrossMsg model.
func (*CrossMsg) TableName() string {
	return "cross_message"
}

// NewCrossMsg returns a new instance of CrossMsg.
func NewCrossMsg(db *gorm.DB) *CrossMsg {
	return &CrossMsg{db: db}
}

// L1 Cross Msgs Operations

// GetL1CrossMsgByHash returns layer1 cross message by given hash
func (l *CrossMsg) GetL1CrossMsgByHash(l1Hash common.Hash) (*CrossMsg, error) {
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
func (l *CrossMsg) GetLatestL1ProcessedHeight() (uint64, error) {
	result := &CrossMsg{}
	err := l.db.Model(&CrossMsg{}).Where("msg_type = ?", Layer1Msg).
		Order("id DESC").
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
func (l *CrossMsg) GetL1EarliestNoBlockTimestampHeight() (uint64, error) {
	result := &CrossMsg{}
	err := l.db.Model(&CrossMsg{}).
		Where("block_timestamp IS NULL AND msg_type = ?", Layer1Msg).
		Order("height ASC").
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

// InsertL1CrossMsg batch insert layer1 cross messages into db
func (l *CrossMsg) InsertL1CrossMsg(ctx context.Context, messages []*CrossMsg, dbTx ...*gorm.DB) error {
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
func (l *CrossMsg) UpdateL1CrossMsgHash(ctx context.Context, l1Hash, msgHash common.Hash, dbTx ...*gorm.DB) error {
	db := l.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	db.WithContext(ctx)
	err := l.db.Model(&CrossMsg{}).Model(&CrossMsg{}).Where("layer1_hash = ?", l1Hash.Hex()).Update("msg_hash", msgHash.Hex()).Error
	return err

}

// UpdateL1BlockTimestamp update layer1 block timestamp
func (l *CrossMsg) UpdateL1BlockTimestamp(height uint64, timestamp time.Time) error {
	err := l.db.Model(&CrossMsg{}).
		Where("height = ? AND msg_type = ?", height, Layer1Msg).
		Update("block_timestamp", timestamp).Error
	return err
}

// DeleteL1CrossMsgAfterHeight soft delete layer1 cross messages after given height
func (l *CrossMsg) DeleteL1CrossMsgAfterHeight(ctx context.Context, height uint64, dbTx ...*gorm.DB) error {
	db := l.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	db.WithContext(ctx)
	result := db.Delete(&CrossMsg{}, "height > ? AND msg_type = ?", height, Layer1Msg)
	return result.Error
}

// L2 Cross Msgs Operations

// GetL2CrossMsgByHash returns layer2 cross message by given hash
func (l *CrossMsg) GetL2CrossMsgByHash(l2Hash common.Hash) (*CrossMsg, error) {
	result := &CrossMsg{}
	err := l.db.Model(&CrossMsg{}).Where("layer2_hash = ? AND msg_type = ?", l2Hash.String(), Layer1Msg).First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
	}
	return result, err
}

// GetLatestL2ProcessedHeight returns the latest processed height of layer2 cross messages
func (l *CrossMsg) GetLatestL2ProcessedHeight() (uint64, error) {
	result := &CrossMsg{}
	err := l.db.Model(&CrossMsg{}).
		Where("msg_type = ?", Layer2Msg).
		Order("id DESC").
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

// GetL2CrossMsgByMsgHashList returns layer2 cross messages under given msg hashes
func (l *CrossMsg) GetL2CrossMsgByMsgHashList(msgHashList []string) ([]*CrossMsg, error) {
	var results []*CrossMsg
	err := l.db.Model(&CrossMsg{}).
		Where("msg_hash IN (?) AND msg_type = ?", msgHashList, Layer2Msg).
		Find(&results).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
	}
	if len(results) == 0 {
		log.Debug("no CrossMsg under given msg hashes", "msg hash list", msgHashList)
	}
	return results, err
}

// GetL2EarliestNoBlockTimestampHeight returns the earliest layer2 cross message height which has no block timestamp
func (l *CrossMsg) GetL2EarliestNoBlockTimestampHeight() (uint64, error) {
	result := &CrossMsg{}
	err := l.db.Model(&CrossMsg{}).
		Where("block_timestamp IS NULL AND msg_type = ?", Layer2Msg).
		Order("height ASC").
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

// GetL2CrossMsgByAddress returns all layer2 cross messages under given address
// Warning: return empty slice if no data found
func (l *CrossMsg) GetL2CrossMsgByAddress(sender common.Address) ([]*CrossMsg, error) {
	var results []*CrossMsg
	err := l.db.Model(&CrossMsg{}).Where("sender = ? AND msg_type = ?", sender.String(), Layer2Msg).
		Find(&results).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
	}
	return results, err

}

// BatchInsertL2CrossMsg batch insert layer2 cross messages
func (l *CrossMsg) BatchInsertL2CrossMsg(ctx context.Context, messages []*CrossMsg, dbTx ...*gorm.DB) error {
	if len(messages) == 0 {
		return nil
	}
	db := l.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	db.WithContext(ctx)
	err := db.Model(&CrossMsg{}).Create(&messages).Error
	if err != nil {
		l2hashes := make([]string, 0, len(messages))
		heights := make([]uint64, 0, len(messages))
		for _, msg := range messages {
			l2hashes = append(l2hashes, msg.Layer2Hash)
			heights = append(heights, msg.Height)
		}
		log.Error("failed to insert l2 cross messages", "l2hashes", l2hashes, "heights", heights, "err", err)
	}
	return err
}

// UpdateL2CrossMsgHash update layer2 cross message hash
func (l *CrossMsg) UpdateL2CrossMsgHash(ctx context.Context, l2Hash, msgHash common.Hash, dbTx ...*gorm.DB) error {
	db := l.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	db.WithContext(ctx)
	err := db.Model(&CrossMsg{}).
		Where("layer2_hash = ?", l2Hash.String()).
		Update("msg_hash", msgHash.String()).
		Error
	return err
}

// UpdateL2BlockTimestamp update layer2 cross message block timestamp
func (l *CrossMsg) UpdateL2BlockTimestamp(height uint64, timestamp time.Time) error {
	err := l.db.Model(&CrossMsg{}).
		Where("height = ? AND msg_type = ?", height, Layer2Msg).
		Update("block_timestamp", timestamp).Error
	return err
}

// DeleteL2CrossMsgFromHeight delete layer2 cross messages from given height
func (l *CrossMsg) DeleteL2CrossMsgFromHeight(ctx context.Context, height uint64, dbTx ...*gorm.DB) error {
	db := l.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	db.WithContext(ctx)
	err := db.Model(&CrossMsg{}).Delete("height > ? AND msg_type = ?", height, Layer2Msg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
	}
	return err
}
