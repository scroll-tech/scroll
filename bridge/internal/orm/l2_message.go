package orm

import (
	"context"
	"errors"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
)

type L2Message struct {
	db *gorm.DB `gorm:"column:-"`

	Nonce      uint64 `json:"nonce" gorm:"column:nonce"`
	MsgHash    string `json:"msg_hash" gorm:"column:msg_hash"`
	Height     uint64 `json:"height" gorm:"column:height"`
	Sender     string `json:"sender" gorm:"column:sender"`
	Value      string `json:"value" gorm:"column:value"`
	Target     string `json:"target" gorm:"column:target"`
	Calldata   string `json:"calldata" gorm:"column:calldata"`
	Layer2Hash string `json:"layer2_hash" gorm:"column:layer2_hash"`
	Layer1Hash string `json:"layer1_hash" gorm:"column:layer1_hash"`
	Proof      string `json:"proof" gorm:"column:proof"`
	Status     int    `json:"status" gorm:"column:status;default:1"`
}

// NewL2Message create an L2Message instance
func NewL2Message(db *gorm.DB) *L2Message {
	return &L2Message{db: db}
}

// TableName define the L1Message table name
func (*L2Message) TableName() string {
	return "l2_message"
}

// GetL2Messages fetch list of messages given msg status
func (m *L2Message) GetL2Messages(fields map[string]interface{}, orderByList []string, limit int) ([]L2Message, error) {
	var l2MsgList []L2Message
	selectFields := "nonce, msg_hash, height, sender, target, value, calldata, layer2_hash, status"
	db := m.db.Select(selectFields)
	for key, value := range fields {
		db.Where(key, value)
	}

	for _, orderBy := range orderByList {
		db.Order(orderBy)
	}

	if limit != 0 {
		db.Limit(limit)
	}

	if err := db.Find(&l2MsgList).Error; err != nil {
		return nil, err
	}
	return l2MsgList, nil
}

// GetLayer2LatestWatchedHeight returns latest height stored in the table
func (m *L2Message) GetLayer2LatestWatchedHeight() (uint64, error) {
	// @note It's not correct, since we may don't have message in some blocks.
	// But it will only be called at start, some redundancy is acceptable.
	var L2Msg L2Message
	err := m.db.Select("COALESCE(MAX(height), -1)").First(&L2Msg).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	return L2Msg.Height, nil
}

// GetL2MessageByNonce fetch message by nonce
// for unit test
func (m *L2Message) GetL2MessageByNonce(nonce uint64) (*L2Message, error) {
	var msg L2Message
	selectFields := "nonce, msg_hash, height, sender, target, value, calldata, layer2_hash, status"
	err := m.db.Select(selectFields).Where("nonce", nonce).First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// SaveL2Messages batch save a list of layer2 messages
func (m *L2Message) SaveL2Messages(ctx context.Context, messages []L2Message) error {
	if len(messages) == 0 {
		return nil
	}

	err := m.db.WithContext(ctx).Create(&messages).Error
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
func (m *L2Message) UpdateLayer2Status(ctx context.Context, msgHash string, status types.MsgStatus) error {
	err := m.db.Model(&L2Message{}).WithContext(ctx).Where("msg_hash", msgHash).Update("status", int(status)).Error
	if err != nil {
		return err
	}
	return nil
}

// UpdateLayer2StatusAndLayer1Hash updates message stauts and layer1 transaction hash, given message hash
func (m *L2Message) UpdateLayer2StatusAndLayer1Hash(ctx context.Context, msgHash string, status types.MsgStatus, layer1Hash string) error {
	updateFields := map[string]interface{}{
		"status":      int(status),
		"layer1_hash": layer1Hash,
	}
	err := m.db.Model(&L2Message{}).WithContext(ctx).Where("msg_hash", msgHash).Updates(updateFields).Error
	if err != nil {
		return err
	}
	return nil
}
