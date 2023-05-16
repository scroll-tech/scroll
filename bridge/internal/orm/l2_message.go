package orm

import (
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
)

type L2Message struct {
	db *gorm.DB `gorm:"-"`

	Nonce       uint64    `json:"nonce" gorm:"nonce"`
	MsgHash     string    `json:"msg_hash" gorm:"msg_hash"`
	Height      uint64    `json:"height" gorm:"height"`
	Sender      string    `json:"sender" gorm:"sender"`
	Value       string    `json:"value" gorm:"value"`
	Target      string    `json:"target" gorm:"target"`
	Calldata    string    `json:"calldata" gorm:"calldata"`
	Layer2Hash  string    `json:"layer2_hash" gorm:"layer2_hash"`
	Layer1Hash  string    `json:"layer1_hash" gorm:"layer1_hash"`
	Proof       string    `json:"proof" gorm:"proof"`
	Status      int       `json:"status" gorm:"status"`
	CreatedTime time.Time `json:"created_time" gorm:"created_time"`
	UpdatedTime time.Time `json:"updated_time" gorm:"updated_time"`
}

// NewL2Message create an L2Message instance
func NewL2Message(db *gorm.DB) *L2Message {
	return &L2Message{db: db}
}

// TableName define the L1Message table name
func (*L2Message) TableName() string {
	return "l1_message"
}

// GetL2Messages fetch list of messages given msg status
func (m *L2Message) GetL2Messages(fields map[string]interface{}) ([]L2Message, error) {
	var l2MsgList []L2Message
	selectFields := "nonce, msg_hash, height, sender, target, value, calldata, layer2_hash"
	db := m.db.Select(selectFields)
	for key, value := range fields {
		db.Where(key, value)
	}
	if err := db.Find(&l2MsgList).Error; err != nil {
		return nil, err
	}
	return l2MsgList, nil
}

// SaveL2Messages batch save a list of layer2 messages
func (m *L2Message) SaveL2Messages(messages []L2Message) error {
	if len(messages) == 0 {
		return nil
	}

	err := m.db.Create(&messages).Error
	if err != nil {
		nonces := make([]uint64, 0, len(messages))
		heights := make([]uint64, 0, len(messages))
		for _, msg := range messages {
			nonces = append(nonces, msg.Nonce)
			heights = append(heights, msg.Height)
		}
		log.Error("failed to insert layer2Messages", "nonces", nonces, "heights", heights, "err", err)
	}
	return err
}

// UpdateLayer2Status updates message stauts, given message hash
func (m *L2Message) UpdateLayer2Status(msgHash string, status types.MsgStatus) error {
	err := m.db.Model(&L2Message{}).Where("msg_hash", msgHash).Update("status", status).Error
	if err != nil {
		return err
	}
	return nil
}

// UpdateLayer2StatusAndLayer1Hash updates message stauts and layer1 transaction hash, given message hash
func (m *L2Message) UpdateLayer2StatusAndLayer1Hash(msgHash string, status types.MsgStatus, layer1Hash string) error {
	updateFields := map[string]interface{}{
		"status":      status,
		"layer1_hash": layer1Hash,
	}
	err := m.db.Model(&L2Message{}).Where("msg_hash", msgHash).Updates(updateFields).Error
	if err != nil {
		return err
	}
	return nil
}
