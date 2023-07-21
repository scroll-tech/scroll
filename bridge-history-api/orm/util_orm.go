package orm

import (
	"gorm.io/gorm"
)

// UtilDBOrm provide combined db operations
type UtilDBOrm struct {
	db *gorm.DB
}

// NewUtilDBOrm init the UtilDBOrm
func NewUtilDBOrm(db *gorm.DB) *UtilDBOrm {
	return &UtilDBOrm{
		db: db,
	}
}

// GetTotalCrossMsgCountByAddress get total cross msg count by address
func (u *UtilDBOrm) GetTotalCrossMsgCountByAddress(sender string) (uint64, error) {
	var count int64
	err := u.db.Model(&CrossMsg{}).
		Where("sender = ?", sender).
		Count(&count).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return uint64(count), err
}

// GetCrossMsgsByAddressWithOffset get cross msgs by address with offset
func (u *UtilDBOrm) GetCrossMsgsByAddressWithOffset(sender string, offset int, limit int) ([]CrossMsg, error) {
	var messages []CrossMsg
	err := u.db.Model(&CrossMsg{}).
		Where("sender = ?", sender).
		Order("block_timestamp DESC NULLS FIRST, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
	}
	return messages, err
}
