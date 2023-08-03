package orm

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Random store the random string from prover client
type Random struct {
	db *gorm.DB `gorm:"column:-"`

	ID     int64  `json:"id" gorm:"column:id"`
	Random string `json:"random" gorm:"column:random"`
	// metadata
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at"`
}

// NewRandom creates a new random instance.
func NewRandom(db *gorm.DB) *Random {
	return &Random{db: db}
}

// TableName returns the name of the "prover_task" table.
func (r *Random) TableName() string {
	return "random"
}

// InsertRandom check the random string exist, if the random string is existed
// return error, if not, just insert it
func (r *Random) InsertRandom(ctx context.Context, randomString string) error {
	random := Random{
		Random: randomString,
	}

	db := r.db.WithContext(ctx)
	db = db.Model(&Random{})
	db = db.Where("random = ?", randomString)
	result := db.FirstOrCreate(&random)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 1 {
		return nil
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("the random string:%s have been used", randomString)
	}

	return fmt.Errorf("insert random string affected rows more than 1")
}
