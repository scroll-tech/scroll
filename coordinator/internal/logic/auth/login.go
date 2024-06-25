package auth

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"scroll-tech/common/version"
	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/types"

	"scroll-tech/coordinator/internal/orm"
)

// LoginLogic the auth logic
type LoginLogic struct {
	cfg          *config.Config
	challengeOrm *orm.Challenge
}

// NewLoginLogic new a LoginLogic
func NewLoginLogic(db *gorm.DB, cfg *config.Config) *LoginLogic {
	return &LoginLogic{
		cfg:          cfg,
		challengeOrm: orm.NewChallenge(db),
	}
}

// InsertChallengeString insert and check the challenge string is existed
func (l *LoginLogic) InsertChallengeString(ctx *gin.Context, challenge string) error {
	return l.challengeOrm.InsertChallenge(ctx.Copy(), challenge)
}

func (l *LoginLogic) Check(login *types.LoginParameter) error {
	verify, err := login.Verify()
	if err != nil || !verify {
		return fmt.Errorf("auth message verify failure:%w", err)
	}

	if !version.CheckScrollRepoVersion(login.Message.ProverVersion, l.cfg.ProverManager.MinProverVersion) {
		return fmt.Errorf("incompatible prover version. please upgrade your prover, minimum allowed version: %s, actual version: %s",
			l.cfg.ProverManager.MinProverVersion, login.Message.ProverVersion)
	}
}
