package orm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ProverBlockList represents the prover's block entry in the database.
type ProverBlockList struct {
	db *gorm.DB `gorm:"-"`

	ID         uint   `json:"id" gorm:"column:id;primaryKey"`
	ProverName string `json:"prover_name" gorm:"column:prover_name"`
	PublicKey  string `json:"public_key" gorm:"column:public_key"`

	// metadata
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewProverBlockList creates a new ProverBlockList instance.
func NewProverBlockList(db *gorm.DB) *ProverBlockList {
	return &ProverBlockList{db: db}
}

// TableName returns the name of the "prover_block_list" table.
func (*ProverBlockList) TableName() string {
	return "prover_block_list"
}

// InsertProverPublicKey adds a new Prover public key to the block list.
// for unit test only.
func (p *ProverBlockList) InsertProverPublicKey(ctx context.Context, proverName, publicKey string) error {
	prover := ProverBlockList{
		ProverName: proverName,
		PublicKey:  publicKey,
	}

	db := p.db.WithContext(ctx)
	db = db.Model(&ProverBlockList{})
	if err := db.Create(&prover).Error; err != nil {
		return fmt.Errorf("ProverBlockList.InsertProverPublicKey error: %w, prover name: %v, public key: %v", err, proverName, publicKey)
	}
	return nil
}

// DeleteProverPublicKey marks a Prover public key as deleted in the block list.
// for unit test only.
func (p *ProverBlockList) DeleteProverPublicKey(ctx context.Context, publicKey string) error {
	db := p.db.WithContext(ctx)
	db = db.Where("public_key = ?", publicKey)
	if err := db.Delete(&ProverBlockList{}).Error; err != nil {
		return fmt.Errorf("ProverBlockList.DeleteProverPublicKey error: %w, public key: %v", err, publicKey)
	}
	return nil
}

// IsPublicKeyBlocked checks if the given public key is blocked.
func (p *ProverBlockList) IsPublicKeyBlocked(ctx context.Context, publicKey string) (bool, error) {
	db := p.db.WithContext(ctx)
	db = db.Model(&ProverBlockList{})
	db = db.Where("public_key = ?", publicKey)
	if err := db.First(&ProverBlockList{}).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil // Public key not found, hence it's not blocked.
		}
		return true, fmt.Errorf("ProverBlockList.IsPublicKeyBlocked error: %w, public key: %v", err, publicKey)
	}

	return true, nil
}
