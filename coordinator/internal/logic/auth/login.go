package auth

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/log"
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
	vks          map[string]struct{}
}

// NewLoginLogic new a LoginLogic
func NewLoginLogic(db *gorm.DB, cfg *config.Config, vkMap map[string]string) *LoginLogic {
	l := &LoginLogic{
		cfg:          cfg,
		challengeOrm: orm.NewChallenge(db),
	}
	for _, vk := range vkMap {
		l.vks[vk] = struct{}{}
	}
	return l
}

// InsertChallengeString insert and check the challenge string is existed
func (l *LoginLogic) InsertChallengeString(ctx *gin.Context, challenge string) error {
	return l.challengeOrm.InsertChallenge(ctx.Copy(), challenge)
}

func (l *LoginLogic) Check(login *types.LoginParameter) error {
	if login.PublicKey != "" {
		verify, err := login.Verify()
		if err != nil || !verify {
			return fmt.Errorf("auth message verify failure:%w", err)
		}
	}

	if !version.CheckScrollRepoVersion(login.Message.ProverVersion, l.cfg.ProverManager.MinProverVersion) {
		return fmt.Errorf("incompatible prover version. please upgrade your prover, minimum allowed version: %s, actual version: %s",
			l.cfg.ProverManager.MinProverVersion, login.Message.ProverVersion)
	}

	for _, vk := range login.Message.VKs {
		if _, ok := l.vks[vk]; !ok {
			log.Error("vk inconsistency", "prover vk", vk)
			if !version.CheckScrollProverVersion(login.Message.ProverVersion) {
				return fmt.Errorf("incompatible prover version. please upgrade your prover, expect version: %s, actual version: %s",
					version.Version, login.Message.ProverVersion)
			}
			// if the prover reports a same prover version
			return fmt.Errorf("incompatible vk. please check your params files or config files")
		}
	}
	return nil
}
