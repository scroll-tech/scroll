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

	ID             uint64     `json:"id" gorm:"column:id"`
	OriginalSender string     `json:"original_sender" gorm:"column:original_sender;default:''"`
	TxHash         string     `json:"tx_hash" gorm:"column:tx_hash"`
	MsgHash        string     `json:"msg_hash" gorm:"column:msg_hash"`
	Sender         string     `json:"sender" gorm:"column:sender"`
	Target         string     `json:"target" gorm:"column:target"`
	Value          string     `json:"value" gorm:"column:value"`
	Height         uint64     `json:"height" gorm:"column:height"`
	Nonce          uint64     `json:"nonce" gorm:"column:nonce"`
	BatchIndex     uint64     `json:"batch_index" gorm:"column:batch_index;default:0"`
	MsgProof       string     `json:"msg_proof" gorm:"column:msg_proof;default:''"`
	MsgData        string     `json:"msg_data" gorm:"column:msg_data;default:''"`
	CreatedAt      *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      *time.Time `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt      *time.Time `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewL2SentMsg create an NewL2SentMsg instance
func NewL2SentMsg(db *gorm.DB) *L2SentMsg {
	return &L2SentMsg{db: db}
}

// GetL2SentMsgByHash get l2 sent msg by hash
func (l *L2SentMsg) GetL2SentMsgByHash(msgHash string) (*L2SentMsg, error) {
	result := &L2SentMsg{}
	err := l.db.Table("l2_sent_msg").
		Where("msg_hash = ? AND deleted_at IS NULL", msgHash).
		First(&result).
		Error
	return result, err
}

// BatchInsertL2SentMsgDBTx batch insert l2 sent msg
func (l *L2SentMsg) BatchInsertL2SentMsgDBTx(dbTx *gorm.DB, messages []*L2SentMsg) (*gorm.DB, error) {
	if len(messages) == 0 {
		return dbTx, nil
	}

	err := dbTx.Model(&L2SentMsg{}).Table("l2_sent_msg").Create(&messages).Error
	if err != nil {
		l2hashes := make([]string, 0, len(messages))
		heights := make([]uint64, 0, len(messages))
		for _, msg := range messages {
			l2hashes = append(l2hashes, msg.TxHash)
			heights = append(heights, msg.Height)
		}
		log.Error("failed to insert l2 sent messages", "l2hashes", l2hashes, "heights", heights, "err", err)
	}
	return dbTx, err
}

// GetLatestSentMsgHeightOnL2 get latest sent msg height on l2
func (l *L2SentMsg) GetLatestSentMsgHeightOnL2() (int64, error) {
	var height int64
	err := l.db.Table("l2_sent_msg").
		Where("deleted_at IS NULL").
		Order("nonce DESC").
		Limit(1).
		Select("height").
		Scan(&height).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return height, err
}

// UpdateL2MessageProofInDBTx update l2 message proof in db tx
func (l *L2SentMsg) UpdateL2MessageProofInDBTx(ctx context.Context, dbTx *gorm.DB, msgHash string, proof string, batchIndex uint64) (*gorm.DB, error) {
	err := dbTx.Table("l2_sent_msg").
		Where("msg_hash = ? AND deleted_at IS NULL", msgHash).
		Updates(map[string]interface{}{
			"msg_proof":   proof,
			"batch_index": batchIndex,
		}).Error
	return dbTx, err
}

// GetLatestL2SentMsgBatchIndex get latest l2 sent msg batch index
func (l *L2SentMsg) GetLatestL2SentMsgBatchIndex() (int64, error) {
	var batchIndex int64
	err := l.db.Table("l2_sent_msg").
		Where("batch_index != 0 AND deleted_at IS NULL").
		Order("batch_index DESC").
		Select("batch_index").
		Limit(1).
		Scan(&batchIndex).Error
	if err != nil {
		return -1, err
	}
	return batchIndex, nil
}

// GetL2SentMsgMsgHashByHeightRange get l2 sent msg msg hash by height range
func (l *L2SentMsg) GetL2SentMsgMsgHashByHeightRange(startHeight, endHeight uint64) ([]*L2SentMsg, error) {
	var results []*L2SentMsg
	err := l.db.Table("l2_sent_msg").
		Where("height >= ? AND height <= ? AND deleted_at IS NULL", startHeight, endHeight).
		Order("nonce ASC").
		Find(&results).
		Error
	return results, err
}

// GetL2SentMessageByNonce get l2 sent message by nonce
func (l *L2SentMsg) GetL2SentMessageByNonce(nonce uint64) (*L2SentMsg, error) {
	result := &L2SentMsg{}
	err := l.db.Table("l2_sent_msg").
		Where("nonce = ? AND deleted_at IS NULL", nonce).
		First(&result).
		Error
	return result, err
}

// GetLatestL2SentMsgLEHeight get latest l2 sent msg less than or equal to end block number
func (l *L2SentMsg) GetLatestL2SentMsgLEHeight(endBlockNumber uint64) (*L2SentMsg, error) {
	result := &L2SentMsg{}
	err := l.db.Table("l2_sent_msg").
		Where("height <= ? AND deleted_at IS NULL", endBlockNumber).
		Order("nonce DESC").
		First(&result).
		Error
	return result, err
}

// DeleteL2SentMsgAfterHeightDBTx delete l2 sent msg after height
func (l *L2SentMsg) DeleteL2SentMsgAfterHeightDBTx(dbTx *gorm.DB, height int64) (*gorm.DB, error) {
	err := dbTx.Table("l2_sent_msg").
		Where("height > ?", height).
		Updates(map[string]interface{}{
			"deleted_at": gorm.Expr("current_timestamp"),
		}).Error
	return dbTx, err
}

// GetClaimableL2SentMsgByAddressWithOffset get claimable l2 sent msg by address with offset
func (l *L2SentMsg) GetClaimableL2SentMsgByAddressWithOffset(address string, offset int, limit int) ([]*L2SentMsg, error) {
	var results []*L2SentMsg
	err := l.db.Raw(`SELECT * FROM l2_sent_msg WHERE id NOT IN (SELECT l2_sent_msg.id FROM l2_sent_msg INNER JOIN relayed_msg ON l2_sent_msg.msg_hash = relayed_msg.msg_hash WHERE l2_sent_msg.deleted_at IS NULL AND relayed_msg.deleted_at IS NULL) AND (original_sender=$1 OR sender = $1) AND msg_proof !='' ORDER BY id DESC LIMIT $2 OFFSET $3;`, address, limit, offset).
		Scan(&results).Error
	return results, err
}

// GetClaimableL2SentMsgByAddressTotalNum get claimable l2 sent msg by address total num
func (l *L2SentMsg) GetClaimableL2SentMsgByAddressTotalNum(address string) (uint64, error) {
	var count uint64
	err := l.db.Raw(`SELECT COUNT(*) FROM l2_sent_msg WHERE id NOT IN (SELECT l2_sent_msg.id FROM l2_sent_msg INNER JOIN relayed_msg ON l2_sent_msg.msg_hash = relayed_msg.msg_hash WHERE l2_sent_msg.deleted_at IS NULL AND relayed_msg.deleted_at IS NULL) AND (original_sender=$1 OR sender = $1) AND msg_proof !='';`, address).
		Scan(&count).Error
	return count, err
}
