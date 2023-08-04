package auth

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"scroll-tech/coordinator/internal/orm"
)

// LoginLogic the auth logic
type LoginLogic struct {
	challengeOrm *orm.Challenge
}

// NewLoginLogic new a LoginLogic
func NewLoginLogic(db *gorm.DB) *LoginLogic {
	return &LoginLogic{
		challengeOrm: orm.NewChallenge(db),
	}
}

// InsertChallengeString insert and check the challenge string is existed
func (l *LoginLogic) InsertChallengeString(ctx *gin.Context, signature string) error {
	return l.challengeOrm.InsertChallenge(ctx, signature)
}
