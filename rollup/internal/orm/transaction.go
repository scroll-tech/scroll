package orm

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"gorm.io/gorm"

	"scroll-tech/common/types"
)

// SenderMeta holds the metadata for a transaction sender including the name, service, address and type.
type SenderMeta struct {
	Name    string
	Service string
	Address common.Address
	Type    types.SenderType
}

// Transaction represents the structure of a transaction in the database.
type Transaction struct {
	db *gorm.DB `gorm:"column:-"`

	ID            uint             `json:"id" gorm:"primaryKey"`
	ContextID     string           `json:"context_id" gorm:"context_id"`
	Hash          string           `json:"hash" gorm:"hash"`
	Type          uint8            `json:"type" gorm:"type"`
	GasFeeCap     string           `json:"gas_fee_cap" gorm:"gas_fee_cap"`
	GasTipCap     string           `json:"gas_tip_cap" gorm:"gas_tip_cap"`
	GasPrice      string           `json:"gas_price" gorm:"gas_price"`
	GasLimit      uint64           `json:"gas_limit" gorm:"gas_limit"`
	Nonce         uint64           `json:"nonce" gorm:"nonce"`
	SubmitAt      uint64           `json:"submit_at" gorm:"submit_at"`
	Status        types.TxStatus   `json:"status" gorm:"status"`
	RLPEncoding   []byte           `json:"rlp_encoding" gorm:"rlp_encoding"`
	SenderName    string           `json:"sender_name" gorm:"sender_name"`
	SenderService string           `json:"sender_service" gorm:"sender_service"`
	SenderAddress string           `json:"sender_address" gorm:"sender_address"`
	SenderType    types.SenderType `json:"sender_type" gorm:"sender_type"`
	CreatedAt     time.Time        `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     time.Time        `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt     gorm.DeletedAt   `json:"deleted_at" gorm:"column:deleted_at"`
}

// TableName returns the table name for the Transaction model.
func (*Transaction) TableName() string {
	return "transaction"
}

// NewTransaction returns a new instance of Transaction.
func NewTransaction(db *gorm.DB) *Transaction {
	return &Transaction{db: db}
}

// InsertTransaction creates a new transaction record and stores it in the database.
func (t *Transaction) InsertTransaction(ctx context.Context, contextID string, senderMeta *SenderMeta, tx *gethTypes.Transaction, submitAt uint64) error {
	rlp := new(bytes.Buffer)
	if err := tx.EncodeRLP(rlp); err != nil {
		return fmt.Errorf("failed to encode rlp, err: %w", err)
	}

	newTransaction := &Transaction{
		ContextID:     contextID,
		Hash:          tx.Hash().String(),
		Type:          tx.Type(),
		GasFeeCap:     tx.GasFeeCap().String(),
		GasTipCap:     tx.GasTipCap().String(),
		GasPrice:      tx.GasPrice().String(),
		GasLimit:      tx.Gas(),
		Nonce:         tx.Nonce(),
		SubmitAt:      submitAt,
		Status:        types.TxStatusPending,
		RLPEncoding:   rlp.Bytes(),
		SenderName:    senderMeta.Name,
		SenderAddress: senderMeta.Address.String(),
		SenderService: senderMeta.Service,
		SenderType:    senderMeta.Type,
	}

	db := t.db.WithContext(ctx)
	if err := db.Create(newTransaction).Error; err != nil {
		return fmt.Errorf("failed to InsertTransaction, error: %w", err)
	}
	return nil
}

// UpdateTransactionStatusByContextID updates the status of a transaction based on the given context ID.
func (t *Transaction) UpdateTransactionStatusByContextID(ctx context.Context, contextID string, status types.TxStatus) error {
	db := t.db.WithContext(ctx)
	db = db.Model(&Transaction{})
	db = db.Where("context_id = ?", contextID)
	if err := db.Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to UpdateTransactionStatus, error: %w", err)
	}
	return nil
}

// GetPendingTransactionsBySenderType retrieves pending transactions filtered by sender type, ordered by nonce, and limited to a specified count.
func (t *Transaction) GetPendingTransactionsBySenderType(ctx context.Context, senderType types.SenderType, limit int) ([]Transaction, error) {
	var transactions []Transaction
	db := t.db.WithContext(ctx)
	db = db.Where("sender_type = ?", senderType)
	db = db.Where("status = ?", types.TxStatusPending)
	db = db.Order("nonce asc")
	db = db.Limit(limit)
	if err := db.Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending transactions by sender type, error: %w", err)
	}
	return transactions, nil
}
