package auth

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"scroll-tech/coordinator/internal/orm"
)

// LoginLogic the auth logic
type LoginLogic struct {
	randomOrm *orm.Random
}

// NewLoginLogic new a LoginLogic
func NewLoginLogic(db *gorm.DB) *LoginLogic {
	return &LoginLogic{
		randomOrm: orm.NewRandom(db),
	}
}

// InsertRandomString insert and check the random string is existed
func (l *LoginLogic) InsertRandomString(ctx *gin.Context, signature string) error {
	return l.randomOrm.InsertRandom(ctx, signature)
}
