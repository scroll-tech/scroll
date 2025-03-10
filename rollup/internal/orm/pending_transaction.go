package orm

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
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
	ChainID           uint64           `json:"chain_id" gorm:"chain_id"`
	Type              uint8            `json:"type" gorm:"type"`
	GasTipCap         uint64           `json:"gas_tip_cap" gorm:"gas_tip_cap"`
	GasFeeCap         uint64           `json:"gas_fee_cap" gorm:"gas_fee_cap"`
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
func (o *PendingTransaction) GetTxStatusByTxHash(ctx context.Context, hash common.Hash) (types.TxStatus, error) {
	var status types.TxStatus
	db := o.db.WithContext(ctx)
	db = db.Model(&PendingTransaction{})
	db = db.Select("status")
	db = db.Where("hash = ?", hash.String())
	if err := db.Scan(&status).Error; err != nil {
		return types.TxStatusUnknown, fmt.Errorf("failed to get tx status by hash, hash: %v, err: %w", hash, err)
	}
	return status, nil
}

// GetPendingOrReplacedTransactionsBySenderType retrieves pending or replaced transactions filtered by sender type, ordered by nonce, then gas_fee_cap (gas_price in legacy tx), and limited to a specified count.
func (o *PendingTransaction) GetPendingOrReplacedTransactionsBySenderType(ctx context.Context, senderType types.SenderType, limit int) ([]PendingTransaction, error) {
	var transactions []PendingTransaction
	db := o.db.WithContext(ctx)
	db = db.Model(&PendingTransaction{})
	db = db.Where("sender_type = ?", senderType)
	db = db.Where("status = ? OR status = ?", types.TxStatusPending, types.TxStatusReplaced)
	db = db.Order("nonce asc")
	db = db.Order("gas_fee_cap asc")
	db = db.Limit(limit)
	if err := db.Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending or replaced transactions by sender type, error: %w", err)
	}
	return transactions, nil
}

// GetCountPendingTransactionsBySenderType retrieves number of pending transactions filtered by sender type
func (o *PendingTransaction) GetCountPendingTransactionsBySenderType(ctx context.Context, senderType types.SenderType) (int64, error) {
	var count int64
	db := o.db.WithContext(ctx)
	db = db.Model(&PendingTransaction{})
	db = db.Where("sender_type = ?", senderType)
	db = db.Where("status = ?", types.TxStatusPending)
	if err := db.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count pending transactions by sender type, error: %w", err)
	}
	return count, nil
}

// GetConfirmedTransactionsBySenderType retrieves confirmed transactions filtered by sender type, limited to a specified count.
// for unit test
func (o *PendingTransaction) GetConfirmedTransactionsBySenderType(ctx context.Context, senderType types.SenderType, limit int) ([]PendingTransaction, error) {
	var transactions []PendingTransaction
	db := o.db.WithContext(ctx)
	db = db.Model(&PendingTransaction{})
	db = db.Where("sender_type = ?", senderType)
	db = db.Where("status = ?", types.TxStatusConfirmed)
	db = db.Limit(limit)
	if err := db.Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get confirmed transactions by sender type, error: %w", err)
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
		ChainID:           tx.ChainId().Uint64(),
		GasFeeCap:         tx.GasFeeCap().Uint64(),
		GasTipCap:         tx.GasTipCap().Uint64(),
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

// DeleteTransactionByTxHash permanently deletes a transaction record from the database by transaction hash.
// Using permanent delete (Unscoped) instead of soft delete to prevent database bloat, as repeated SendTransaction failures
// could write a large number of transactions to the database.
func (o *PendingTransaction) DeleteTransactionByTxHash(ctx context.Context, hash common.Hash, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&PendingTransaction{})

	// Perform permanent delete by using Unscoped()
	result := db.Where("hash = ?", hash.String()).Unscoped().Delete(&PendingTransaction{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete transaction, err: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no transaction found with hash: %s", hash.String())
	}
	if result.RowsAffected > 0 {
		log.Warn("Successfully deleted transaction", "hash", hash.String())
	}
	return nil
}

// UpdateTransactionStatusByTxHash updates the status of a transaction based on the transaction hash.
func (o *PendingTransaction) UpdateTransactionStatusByTxHash(ctx context.Context, hash common.Hash, status types.TxStatus, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&PendingTransaction{})
	db = db.Where("hash = ?", hash.String())
	if err := db.Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to UpdateTransactionStatusByTxHash, txHash: %s, error: %w", hash, err)
	}
	return nil
}

// UpdateOtherTransactionsAsFailedByNonce updates the status of all transactions to TxStatusConfirmedFailed for a specific nonce and sender address, excluding a specified transaction hash.
func (o *PendingTransaction) UpdateOtherTransactionsAsFailedByNonce(ctx context.Context, senderAddress string, nonce uint64, hash common.Hash, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&PendingTransaction{})
	db = db.Where("sender_address = ?", senderAddress)
	db = db.Where("nonce = ?", nonce)
	db = db.Where("hash != ?", hash.String())
	if err := db.Update("status", types.TxStatusConfirmedFailed).Error; err != nil {
		return fmt.Errorf("failed to update other transactions as failed by nonce, senderAddress: %s, nonce: %d, txHash: %s, error: %w", senderAddress, nonce, hash, err)
	}
	return nil
}
