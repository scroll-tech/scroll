package orm

import (
	"errors"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
)

// L1Message is structure of stored layer1 bridge message
type L1Message struct {
	db *gorm.DB `gorm:"-"`

	QueueIndex  uint64    `json:"queue_index" gorm:"queue_index"`
	MsgHash     string    `json:"msg_hash" gorm:"msg_hash"`
	Height      uint64    `json:"height" gorm:"height"`
	GasLimit    uint64    `json:"gas_limit" gorm:"gas_limit"`
	Sender      string    `json:"sender" gorm:"sender"`
	Target      string    `json:"target" gorm:"target"`
	Value       string    `json:"value" gorm:"value"`
	Calldata    string    `json:"calldata" gorm:"calldata"`
	Layer1Hash  string    `json:"layer1_hash" gorm:"layer1_hash"`
	Layer2Hash  string    `json:"layer2_hash" gorm:"layer2_hash"`
	Status      int       `json:"status" gorm:"status"`
	CreatedTime time.Time `json:"created_time" gorm:"created_time"`
	UpdatedTime time.Time `json:"updated_time" gorm:"updated_time"`
}

// NewL1Message create an L1MessageOrm instance
func NewL1Message(db *gorm.DB) *L1Message {
	return &L1Message{db: db}
}

// TableName define the L1Message table name
func (*L1Message) TableName() string {
	return "l2_message"
}

// GetL1MessagesByStatus fetch list of unprocessed messages given msg status
func (m *L1Message) GetL1MessagesByStatus(status types.MsgStatus, limit uint64) ([]L1Message, error) {
	var msgs []L1Message
	fields := "queue_index, msg_hash, height, sender, target, value, calldata, layer1_hash, status"
	err := m.db.Select(fields).Where("status", status).Order("queue_index ASC").Limit(int(limit)).Find(&msgs).Error
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

// GetLayer1LatestWatchedHeight returns latest height stored in the table
func (m *L1Message) GetLayer1LatestWatchedHeight() (uint64, error) {
	// @note It's not correct, since we may don't have message in some blocks.
	// But it will only be called at start, some redundancy is acceptable.
	var msg L1Message
	err := m.db.Select("MAX(height)").First(&msg).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	return msg.Height, nil
}

// SaveL1Messages batch save a list of layer1 messages
func (m *L1Message) SaveL1Messages(messages []*L1Message) error {
	if len(messages) == 0 {
		return nil
	}

	err := m.db.Create(&messages).Error
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
func (m *L1Message) UpdateLayer1Status(msgHash string, status types.MsgStatus) error {
	if err := m.db.Model(&L1Message{}).Where("msg_hash", msgHash).Update("status", status).Error; err != nil {
		return err
	}
	return nil
}

// UpdateLayer1StatusAndLayer2Hash updates message status and layer2 transaction hash, given message hash
func (m *L1Message) UpdateLayer1StatusAndLayer2Hash(msgHash string, status types.MsgStatus, layer2Hash string) error {
	updateFields := map[string]interface{}{
		"status":      status,
		"layer2_hash": layer2Hash,
	}
	if err := m.db.Model(&L1Message{}).Where("msg_hash", msgHash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}
