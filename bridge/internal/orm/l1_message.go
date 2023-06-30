package orm

import (
	"context"
	"database/sql"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
)

// L1Message is structure of stored layer1 bridge message
type L1Message struct {
	db *gorm.DB `gorm:"column:-"`

	QueueIndex uint64 `json:"queue_index" gorm:"column:queue_index"`
	MsgHash    string `json:"msg_hash" gorm:"column:msg_hash"`
	Height     uint64 `json:"height" gorm:"column:height"`
	GasLimit   uint64 `json:"gas_limit" gorm:"column:gas_limit"`
	Sender     string `json:"sender" gorm:"column:sender"`
	Target     string `json:"target" gorm:"column:target"`
	Value      string `json:"value" gorm:"column:value"`
	Calldata   string `json:"calldata" gorm:"column:calldata"`
	Layer1Hash string `json:"layer1_hash" gorm:"column:layer1_hash"`
	Layer2Hash string `json:"layer2_hash" gorm:"column:layer2_hash;default:NULL"`
	Status     int    `json:"status" gorm:"column:status;default:1"`
}

// NewL1Message create an L1MessageOrm instance
func NewL1Message(db *gorm.DB) *L1Message {
	return &L1Message{db: db}
}

// TableName define the L1Message table name
func (*L1Message) TableName() string {
	return "l1_message"
}

// GetLayer1LatestWatchedHeight returns latest height stored in the table
func (m *L1Message) GetLayer1LatestWatchedHeight() (int64, error) {
	// @note It's not correct, since we may don't have message in some blocks.
	// But it will only be called at start, some redundancy is acceptable.
	var maxHeight sql.NullInt64
	result := m.db.Model(&L1Message{}).Select("MAX(height)").Scan(&maxHeight)
	if result.Error != nil {
		return -1, result.Error
	}
	if maxHeight.Valid {
		return maxHeight.Int64, nil
	}
	return -1, nil
}

// GetLayer1LatestMessageWithLayer2Hash returns latest l1 message with layer2 hash
func (m *L1Message) GetLayer1LatestMessageWithLayer2Hash() (*L1Message, error) {
	var msg *L1Message
	err := m.db.Where("layer2_hash IS NOT NULL").Order("queue_index DESC").First(&msg).Error
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// GetL1MessagesByStatus fetch list of unprocessed messages given msg status
func (m *L1Message) GetL1MessagesByStatus(status types.MsgStatus, limit uint64) ([]L1Message, error) {
	var msgs []L1Message
	err := m.db.Where("status", int(status)).Order("queue_index ASC").Limit(int(limit)).Find(&msgs).Error
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

// GetL1MessageByQueueIndex fetch message by queue_index
// for unit test
func (m *L1Message) GetL1MessageByQueueIndex(queueIndex uint64) (*L1Message, error) {
	var msg L1Message
	err := m.db.Where("queue_index", queueIndex).First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetL1MessageByMsgHash fetch message by queue_index
// for unit test
func (m *L1Message) GetL1MessageByMsgHash(msgHash string) (*L1Message, error) {
	var msg L1Message
	err := m.db.Where("msg_hash", msgHash).First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// SaveL1Messages batch save a list of layer1 messages
func (m *L1Message) SaveL1Messages(ctx context.Context, messages []*L1Message) error {
	if len(messages) == 0 {
		return nil
	}

	err := m.db.WithContext(ctx).Create(&messages).Error
	if err != nil {
		queueIndices := make([]uint64, 0, len(messages))
		heights := make([]uint64, 0, len(messages))
		for _, msg := range messages {
			queueIndices = append(queueIndices, msg.QueueIndex)
			heights = append(heights, msg.Height)
		}
		log.Error("failed to insert l1Messages", "queueIndices", queueIndices, "heights", heights, "err", err)
	}
	return err
}

// UpdateLayer1Status updates message stauts, given message hash
func (m *L1Message) UpdateLayer1Status(ctx context.Context, msgHash string, status types.MsgStatus) error {
	if err := m.db.Model(&L1Message{}).WithContext(ctx).Where("msg_hash", msgHash).Update("status", int(status)).Error; err != nil {
		return err
	}
	return nil
}

// UpdateLayer1StatusAndLayer2Hash updates message status and layer2 transaction hash, given message hash
func (m *L1Message) UpdateLayer1StatusAndLayer2Hash(ctx context.Context, msgHash string, status types.MsgStatus, layer2Hash string) error {
	updateFields := map[string]interface{}{
		"status":      int(status),
		"layer2_hash": layer2Hash,
	}
	if err := m.db.Model(&L1Message{}).WithContext(ctx).Where("msg_hash", msgHash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}
