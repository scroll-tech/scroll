package orm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Challenge store the challenge string from prover client
type Challenge struct {
	db *gorm.DB `gorm:"column:-"`

	ID        int64  `json:"id" gorm:"column:id"`
	Challenge string `json:"challenge" gorm:"column:challenge"`
	// metadata
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at"`
}

// NewChallenge creates a new change instance.
func NewChallenge(db *gorm.DB) *Challenge {
	return &Challenge{db: db}
}

// TableName returns the name of the "prover_task" table.
func (r *Challenge) TableName() string {
	return "challenge"
}

// InsertChallenge check the challenge string exist, if the challenge string is existed
// return error, if not, just insert it
func (r *Challenge) InsertChallenge(ctx context.Context, challengeString string) error {
	challenge := Challenge{
		Challenge: challengeString,
	}

	db := r.db.WithContext(ctx)
	db = db.Model(&Challenge{})
	db = db.Where("challenge = ?", challengeString)
	result := db.FirstOrCreate(&challenge)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 1 {
		return nil
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("the challenge string:%s have been used", challengeString)
	}

	return errors.New("insert challenge string affected rows more than 1")
}

// DeleteExpireChallenge delete the expire challenge
func (r *Challenge) DeleteExpireChallenge(ctx context.Context, expiredTime time.Time) error {
	db := r.db.WithContext(ctx)
	db = db.Model(&Challenge{})
	db = db.Where("created_at < ?", expiredTime)
	if err := db.Unscoped().Delete(&Challenge{}).Error; err != nil {
		return fmt.Errorf("Challenge.DeleteExpireChallenge err: %w", err)
	}
	return nil
}
