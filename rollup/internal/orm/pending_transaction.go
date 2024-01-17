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

// PendingTransaction represents the structure of a transaction in the database.
type PendingTransaction struct {
	db *gorm.DB `gorm:"column:-"`

	ID                uint             `json:"id" gorm:"id;primaryKey"`
	ContextID         string           `json:"context_id" gorm:"context_id"`
	Hash              string           `json:"hash" gorm:"hash"`
	Type              uint8            `json:"type" gorm:"type"`
	GasFeeCap         string           `json:"gas_fee_cap" gorm:"gas_fee_cap"`
	GasTipCap         string           `json:"gas_tip_cap" gorm:"gas_tip_cap"`
	GasPrice          string           `json:"gas_price" gorm:"gas_price"`
	GasLimit          uint64           `json:"gas_limit" gorm:"gas_limit"`
	Nonce             uint64           `json:"nonce" gorm:"nonce"`
	SubmitBlockNumber uint64           `json:"submit_block_number" gorm:"submit_block_number"`
	Status            types.TxStatus   `json:"status" gorm:"status"`
	RLPEncoding       []byte           `json:"rlp_encoding" gorm:"rlp_encoding"`
	SenderName        string           `json:"sender_name" gorm:"sender_name"`
	SenderService     string           `json:"sender_service" gorm:"sender_service"`
	SenderAddress     string           `json:"sender_address" gorm:"sender_address"`
	SenderType        types.SenderType `json:"sender_type" gorm:"sender_type"`
	CreatedAt         time.Time        `json:"created_at" gorm:"column:created_at"`
	UpdatedAt         time.Time        `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt         gorm.DeletedAt   `json:"deleted_at" gorm:"column:deleted_at"`
}

// TableName returns the table name for the Transaction model.
func (*PendingTransaction) TableName() string {
	return "pending_transaction"
}

// NewPendingTransaction returns a new instance of PendingTransaction.
func NewPendingTransaction(db *gorm.DB) *PendingTransaction {
	return &PendingTransaction{db: db}
}

// GetTxStatusByTxHash retrieves the status of a transaction by its hash.
func (o *PendingTransaction) GetTxStatusByTxHash(ctx context.Context, hash string) (types.TxStatus, error) {
	var status types.TxStatus
	db := o.db.WithContext(ctx)
	db = db.Model(&PendingTransaction{})
	db = db.Select("status")
	db = db.Where("hash = ?", hash)
	if err := db.Scan(&status).Error; err != nil {
		return types.TxStatusUnknown, fmt.Errorf("failed to get tx status by hash, hash: %v, err: %w", hash, err)
	}
	return status, nil
}

// GetPendingOrReplacedTransactionsBySenderType retrieves pending or replaced transactions filtered by sender type, ordered by nonce, and limited to a specified count.
func (o *PendingTransaction) GetPendingOrReplacedTransactionsBySenderType(ctx context.Context, senderType types.SenderType, limit int) ([]PendingTransaction, error) {
	var transactions []PendingTransaction
	db := o.db.WithContext(ctx)
	db = db.Model(&PendingTransaction{})
	db = db.Where("sender_type = ?", senderType)
	db = db.Where("status = ? OR status = ?", types.TxStatusPending, types.TxStatusReplaced)
	db = db.Order("nonce asc")
	db = db.Limit(limit)
	if err := db.Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending or replaced transactions by sender type, error: %w", err)
	}
	return transactions, nil
}

// InsertPendingTransaction creates a new pending transaction record and stores it in the database.
func (o *PendingTransaction) InsertPendingTransaction(ctx context.Context, contextID string, senderMeta *SenderMeta, tx *gethTypes.Transaction, submitBlockNumber uint64, dbTX ...*gorm.DB) error {
	rlp := new(bytes.Buffer)
	if err := tx.EncodeRLP(rlp); err != nil {
		return fmt.Errorf("failed to encode rlp, err: %w", err)
	}

	newTransaction := &PendingTransaction{
		ContextID:         contextID,
		Hash:              tx.Hash().String(),
		Type:              tx.Type(),
		GasFeeCap:         tx.GasFeeCap().String(),
		GasTipCap:         tx.GasTipCap().String(),
		GasPrice:          tx.GasPrice().String(),
		GasLimit:          tx.Gas(),
		Nonce:             tx.Nonce(),
		SubmitBlockNumber: submitBlockNumber,
		Status:            types.TxStatusPending,
		RLPEncoding:       rlp.Bytes(),
		SenderName:        senderMeta.Name,
		SenderAddress:     senderMeta.Address.String(),
		SenderService:     senderMeta.Service,
		SenderType:        senderMeta.Type,
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&PendingTransaction{})
	if err := db.Create(newTransaction).Error; err != nil {
		return fmt.Errorf("failed to InsertTransaction, error: %w", err)
	}
	return nil
}

// UpdatePendingTransactionStatusByTxHash updates the status of a transaction based on the transaction hash.
func (o *PendingTransaction) UpdatePendingTransactionStatusByTxHash(ctx context.Context, hash string, status types.TxStatus, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&PendingTransaction{})
	db = db.Where("hash = ?", hash)
	if err := db.Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to UpdatePendingTransactionStatusByTxHash, txHash: %s, error: %w", hash, err)
	}
	return nil
}

// UpdateOtherTransactionsAsFailedByNonce updates the status of all transactions to TxStatusConfirmedFailed for a specific nonce and sender address, excluding a specified transaction hash.
func (o *PendingTransaction) UpdateOtherTransactionsAsFailedByNonce(ctx context.Context, senderAddress string, nonce uint64, txHash string, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&PendingTransaction{})
	db = db.Where("sender_address = ?", senderAddress)
	db = db.Where("nonce = ?", nonce)
	db = db.Where("hash != ?", txHash)
	if err := db.Update("status", types.TxStatusConfirmedFailed).Error; err != nil {
		return fmt.Errorf("failed to update other transactions as failed by nonce, senderAddress: %s, nonce: %d, txHash: %s, error: %w", senderAddress, nonce, txHash, err)
	}
	return nil
}
