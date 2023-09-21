package orm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

// L2SentMsg defines the struct for l2_sent_msg table record
type L2SentMsg struct {
	db *gorm.DB `gorm:"column:-"`

	ID             uint64         `json:"id" gorm:"column:id"`
	OriginalSender string         `json:"original_sender" gorm:"column:original_sender;default:''"`
	TxHash         string         `json:"tx_hash" gorm:"column:tx_hash"`
	MsgHash        string         `json:"msg_hash" gorm:"column:msg_hash"`
	Sender         string         `json:"sender" gorm:"column:sender"`
	Target         string         `json:"target" gorm:"column:target"`
	Value          string         `json:"value" gorm:"column:value"`
	Height         uint64         `json:"height" gorm:"column:height"`
	Nonce          uint64         `json:"nonce" gorm:"column:nonce"`
	BatchIndex     uint64         `json:"batch_index" gorm:"column:batch_index;default:0"`
	MsgProof       string         `json:"msg_proof" gorm:"column:msg_proof;default:''"`
	MsgData        string         `json:"msg_data" gorm:"column:msg_data;default:''"`
	CreatedAt      *time.Time     `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      *time.Time     `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewL2SentMsg create an NewL2SentMsg instance
func NewL2SentMsg(db *gorm.DB) *L2SentMsg {
	return &L2SentMsg{db: db}
}

// TableName returns the table name for the L2SentMsg model.
func (*L2SentMsg) TableName() string {
	return "l2_sent_msg"
}

// GetL2SentMsgByHash get l2 sent msg by hash
func (l *L2SentMsg) GetL2SentMsgByHash(ctx context.Context, msgHash string) (*L2SentMsg, error) {
	var result L2SentMsg
	err := l.db.WithContext(ctx).Model(&L2SentMsg{}).
		Where("msg_hash = ?", msgHash).
		First(&result).
		Error
	if err != nil {
		return nil, fmt.Errorf("L2SentMsg.GetL2SentMsgByHash error: %w", err)
	}
	return &result, nil
}

// GetLatestSentMsgHeightOnL2 get latest sent msg height on l2
func (l *L2SentMsg) GetLatestSentMsgHeightOnL2(ctx context.Context) (uint64, error) {
	var result L2SentMsg
	err := l.db.WithContext(ctx).Model(&L2SentMsg{}).
		Select("height").
		Order("nonce DESC").
		First(&result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, fmt.Errorf("L2SentMsg.GetLatestSentMsgHeightOnL2 error: %w", err)

	}
	return result.Height, nil
}

// GetClaimableL2SentMsgByAddressWithOffset returns both the total number of unclaimed messages and a paginated list of those messages.
// TODO: Add metrics about the result set sizes (total/claimed/unclaimed messages).
func (l *L2SentMsg) GetClaimableL2SentMsgByAddressWithOffset(ctx context.Context, address string, offset int, limit int) (uint64, []*L2SentMsg, error) {
	var totalMsgs []*L2SentMsg
	db := l.db.WithContext(ctx)
	db = db.Table("l2_sent_msg")
	db = db.Where("original_sender = ? OR sender = ?", address, address)
	db = db.Where("msg_proof != ''")
	db = db.Where("deleted_at IS NULL")
	db = db.Order("id DESC")
	if err := db.Find(&totalMsgs).Error; err != nil {
		return 0, nil, err
	}

	msgHashes := []string{}
	for _, msg := range totalMsgs {
		msgHashes = append(msgHashes, msg.MsgHash)
	}

	var claimedMsgHashes []string
	db = l.db.WithContext(ctx)
	db = db.Table("relayed_msg")
	db = db.Where("msg_hash IN (?)", msgHashes)
	db = db.Where("deleted_at IS NULL")
	if err := db.Pluck("msg_hash", &claimedMsgHashes).Error; err != nil {
		return 0, nil, err
	}

	claimedMsgHashSet := make(map[string]struct{})
	for _, hash := range claimedMsgHashes {
		claimedMsgHashSet[hash] = struct{}{}
	}
	var unclaimedL2Msgs []*L2SentMsg
	for _, msg := range totalMsgs {
		if _, found := claimedMsgHashSet[msg.MsgHash]; !found {
			unclaimedL2Msgs = append(unclaimedL2Msgs, msg)
		}
	}

	// pagination
	start := offset
	end := offset + limit
	if start > len(unclaimedL2Msgs) {
		start = len(unclaimedL2Msgs)
	}
	if end > len(unclaimedL2Msgs) {
		end = len(unclaimedL2Msgs)
	}
	return uint64(len(unclaimedL2Msgs)), unclaimedL2Msgs[start:end], nil
}

// GetLatestL2SentMsgBatchIndex get latest l2 sent msg batch index
func (l *L2SentMsg) GetLatestL2SentMsgBatchIndex(ctx context.Context) (int64, error) {
	var result L2SentMsg
	err := l.db.WithContext(ctx).Model(&L2SentMsg{}).
		Where("batch_index != 0").
		Order("batch_index DESC").
		Select("batch_index").
		First(&result).
		Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return -1, nil
	}
	if err != nil {
		return -1, fmt.Errorf("L2SentMsg.GetLatestL2SentMsgBatchIndex error: %w", err)
	}
	// Watch for overflow, tho its not likely to happen
	return int64(result.BatchIndex), nil
}

// GetL2SentMsgMsgHashByHeightRange get l2 sent msg msg hash by height range
func (l *L2SentMsg) GetL2SentMsgMsgHashByHeightRange(ctx context.Context, startHeight, endHeight uint64) ([]*L2SentMsg, error) {
	var results []*L2SentMsg
	err := l.db.WithContext(ctx).Model(&L2SentMsg{}).
		Where("height >= ? AND height <= ?", startHeight, endHeight).
		Order("nonce ASC").
		Find(&results).
		Error
	if err != nil {
		return nil, fmt.Errorf("L2SentMsg.GetL2SentMsgMsgHashByHeightRange error: %w", err)
	}
	return results, nil
}

// GetL2SentMessageByNonce get l2 sent message by nonce
func (l *L2SentMsg) GetL2SentMessageByNonce(ctx context.Context, nonce uint64) (*L2SentMsg, error) {
	var result L2SentMsg
	err := l.db.WithContext(ctx).Model(&L2SentMsg{}).
		Where("nonce = ?", nonce).
		First(&result).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("L2SentMsg.GetL2SentMessageByNonce error: %w", err)

	}
	return &result, nil
}

// GetLatestL2SentMsgLEHeight get latest l2 sent msg less than or equal to end block number
func (l *L2SentMsg) GetLatestL2SentMsgLEHeight(ctx context.Context, endBlockNumber uint64) (*L2SentMsg, error) {
	var result L2SentMsg
	err := l.db.WithContext(ctx).Model(&L2SentMsg{}).
		Where("height <= ?", endBlockNumber).
		Order("nonce DESC").
		First(&result).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("L2SentMsg.GetLatestL2SentMsgLEHeight error: %w", err)

	}
	return &result, nil
}

// InsertL2SentMsg batch insert l2 sent msg
func (l *L2SentMsg) InsertL2SentMsg(ctx context.Context, messages []*L2SentMsg, dbTx ...*gorm.DB) error {
	if len(messages) == 0 {
		return nil
	}
	db := l.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	db.WithContext(ctx)
	err := db.Model(&L2SentMsg{}).Create(&messages).Error
	if err != nil {
		l2hashes := make([]string, 0, len(messages))
		heights := make([]uint64, 0, len(messages))
		for _, msg := range messages {
			l2hashes = append(l2hashes, msg.TxHash)
			heights = append(heights, msg.Height)
		}
		log.Error("failed to insert l2 sent messages", "l2hashes", l2hashes, "heights", heights, "err", err)
		return fmt.Errorf("L2SentMsg.InsertL2SentMsg error: %w", err)
	}
	return nil
}

// UpdateL2MessageProof update l2 message proof in db tx
func (l *L2SentMsg) UpdateL2MessageProof(ctx context.Context, msgHash string, proof string, batchIndex uint64, dbTx ...*gorm.DB) error {
	db := l.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	db.WithContext(ctx)
	err := db.Model(&L2SentMsg{}).
		Where("msg_hash = ?", msgHash).
		Updates(map[string]interface{}{
			"msg_proof":   proof,
			"batch_index": batchIndex,
		}).Error
	if err != nil {
		return fmt.Errorf("L2SentMsg.UpdateL2MessageProof error: %w", err)
	}
	return nil
}

// DeleteL2SentMsgAfterHeight delete l2 sent msg after height
func (l *L2SentMsg) DeleteL2SentMsgAfterHeight(ctx context.Context, height uint64, dbTx ...*gorm.DB) error {
	db := l.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	err := db.WithContext(ctx).Model(&L2SentMsg{}).Delete("height > ?", height).Error
	if err != nil {
		return fmt.Errorf("L2SentMsg.DeleteL2SentMsgAfterHeight error: %w", err)
	}
	return nil
}
