package orm

import (
	"context"
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
	result := &L2SentMsg{}
	err := l.db.WithContext(ctx).Model(&L2SentMsg{}).
		Where("msg_hash = ?", msgHash).
		First(result).
		Error
	return result, err
}

// GetLatestSentMsgHeightOnL2 get latest sent msg height on l2
func (l *L2SentMsg) GetLatestSentMsgHeightOnL2(ctx context.Context) (uint64, error) {
	result := &L2SentMsg{}
	err := l.db.WithContext(ctx).Model(&L2SentMsg{}).
		Select("height").
		Order("nonce DESC").
		First(result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return result.Height, err
}

// GetClaimableL2SentMsgByAddressWithOffset get claimable l2 sent msg by address with offset
func (l *L2SentMsg) GetClaimableL2SentMsgByAddressWithOffset(ctx context.Context, address string, offset int, limit int) ([]*L2SentMsg, error) {
	var results []*L2SentMsg
	err := l.db.WithContext(ctx).Raw(`SELECT * FROM l2_sent_msg WHERE id NOT IN (SELECT l2_sent_msg.id FROM l2_sent_msg INNER JOIN relayed_msg ON l2_sent_msg.msg_hash = relayed_msg.msg_hash WHERE l2_sent_msg.deleted_at IS NULL AND relayed_msg.deleted_at IS NULL) AND (original_sender=$1 OR sender = $1) AND msg_proof !='' ORDER BY id DESC LIMIT $2 OFFSET $3;`, address, limit, offset).
		Scan(&results).Error
	return results, err
}

// GetClaimableL2SentMsgByAddressTotalNum get claimable l2 sent msg by address total num
func (l *L2SentMsg) GetClaimableL2SentMsgByAddressTotalNum(ctx context.Context, address string) (uint64, error) {
	var count uint64
	err := l.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM l2_sent_msg WHERE id NOT IN (SELECT l2_sent_msg.id FROM l2_sent_msg INNER JOIN relayed_msg ON l2_sent_msg.msg_hash = relayed_msg.msg_hash WHERE l2_sent_msg.deleted_at IS NULL AND relayed_msg.deleted_at IS NULL) AND (original_sender=$1 OR sender = $1) AND msg_proof !='';`, address).
		Scan(&count).Error
	return count, err
}

// GetLatestL2SentMsgBatchIndex get latest l2 sent msg batch index
func (l *L2SentMsg) GetLatestL2SentMsgBatchIndex(ctx context.Context) (int64, error) {
	result := &L2SentMsg{}
	err := l.db.WithContext(ctx).Model(&L2SentMsg{}).
		Where("batch_index != 0").
		Order("batch_index DESC").
		Select("batch_index").
		First(result).
		Error
	if err != nil {
		return -1, err
	}
	// Watch for overflow, tho its not likely to happen
	return int64(result.Height), nil
}

// GetL2SentMsgMsgHashByHeightRange get l2 sent msg msg hash by height range
func (l *L2SentMsg) GetL2SentMsgMsgHashByHeightRange(ctx context.Context, startHeight, endHeight uint64) ([]*L2SentMsg, error) {
	var results []*L2SentMsg
	err := l.db.WithContext(ctx).Model(&L2SentMsg{}).
		Where("height >= ? AND height <= ?", startHeight, endHeight).
		Order("nonce ASC").
		Find(&results).
		Error
	return results, err
}

// GetL2SentMessageByNonce get l2 sent message by nonce
func (l *L2SentMsg) GetL2SentMessageByNonce(ctx context.Context, nonce uint64) (*L2SentMsg, error) {
	result := &L2SentMsg{}
	err := l.db.WithContext(ctx).Model(&L2SentMsg{}).
		Where("nonce = ?", nonce).
		First(result).
		Error
	return result, err
}

// GetLatestL2SentMsgLEHeight get latest l2 sent msg less than or equal to end block number
func (l *L2SentMsg) GetLatestL2SentMsgLEHeight(ctx context.Context, endBlockNumber uint64) (*L2SentMsg, error) {
	result := &L2SentMsg{}
	err := l.db.WithContext(ctx).Model(&L2SentMsg{}).
		Where("height <= ?", endBlockNumber).
		Order("nonce DESC").
		First(result).
		Error
	return result, err
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
	}
	return err
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
	return err
}

// DeleteL2SentMsgAfterHeight delete l2 sent msg after height
func (l *L2SentMsg) DeleteL2SentMsgAfterHeight(ctx context.Context, height uint64, dbTx ...*gorm.DB) error {
	db := l.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	err := db.WithContext(ctx).Model(&L2SentMsg{}).
		Where("height > ?", height).
		Updates(map[string]interface{}{
			"deleted_at": gorm.Expr("current_timestamp"),
		}).Error
	return err
}
