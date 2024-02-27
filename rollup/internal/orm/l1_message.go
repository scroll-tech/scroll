package orm

import (
	"context"
	"database/sql"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"
)

// L1Message est la structure des messages de pont de couche 1 stockés
type L1Message struct {
	ID         uint64         `gorm:"primaryKey"`
	QueueIndex uint64         `gorm:"column:queue_index"`
	MsgHash    string         `gorm:"column:msg_hash"`
	Height     uint64         `gorm:"column:height"`
	// Ajoutez d'autres champs de modèle ici
	CreatedAt  time.Time      `gorm:"column:created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// NewL1Message crée une instance L1Message
func NewL1Message(db *gorm.DB) *L1Message {
	return &L1Message{db: db}
}

// TableName définit le nom de la table L1Message
func (*L1Message) TableName() string {
	return "l1_message"
}

// GetLayer1LatestWatchedHeight renvoie la dernière hauteur stockée dans la table
func (m *L1Message) GetLayer1LatestWatchedHeight(ctx context.Context) (int64, error) {
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

// SaveL1Messages enregistre en lot une liste de messages de couche 1
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
