package auth

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/version"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/verifier"
	"scroll-tech/coordinator/internal/orm"
	"scroll-tech/coordinator/internal/types"
)

// LoginLogic the auth logic
type LoginLogic struct {
	cfg          *config.Config
	challengeOrm *orm.Challenge
	chunkVks     map[string]struct{}
	batchVKs     map[string]struct{}
	bundleVks    map[string]struct{}

	proverVersionHardForkMap map[string]string
}

// NewLoginLogic new a LoginLogic
func NewLoginLogic(db *gorm.DB, cfg *config.Config, vf *verifier.Verifier) *LoginLogic {
	proverVersionHardForkMap := map[string]string{
		cfg.ProverManager.Verifier.LowVersionCircuit.MinProverVersion:  cfg.ProverManager.Verifier.LowVersionCircuit.ForkName,
		cfg.ProverManager.Verifier.HighVersionCircuit.MinProverVersion: cfg.ProverManager.Verifier.HighVersionCircuit.ForkName,
	}

	return &LoginLogic{
		cfg:                      cfg,
		chunkVks:                 vf.ChunkVKMap,
		batchVKs:                 vf.BatchVKMap,
		bundleVks:                vf.BundleVkMap,
		challengeOrm:             orm.NewChallenge(db),
		proverVersionHardForkMap: proverVersionHardForkMap,
	}
}

// InsertChallengeString insert and check the challenge string is existed
func (l *LoginLogic) InsertChallengeString(ctx *gin.Context, challenge string) error {
	return l.challengeOrm.InsertChallenge(ctx.Copy(), challenge)
}

func (l *LoginLogic) Check(login *types.LoginParameter) error {
	verify, err := login.Verify()
	if err != nil || !verify {
		log.Error("auth message verify failure", "prover_name", login.Message.ProverName,
			"prover_version", login.Message.ProverVersion, "message", login.Message)
		return errors.New("auth message verify failure")
	}

	if !version.CheckScrollRepoVersion(login.Message.ProverVersion, l.cfg.ProverManager.Verifier.LowVersionCircuit.MinProverVersion) {
		return fmt.Errorf("incompatible prover version. please upgrade your prover, minimum allowed version: %s, actual version: %s",
			l.cfg.ProverManager.Verifier.LowVersionCircuit.MinProverVersion, login.Message.ProverVersion)
	}

	if !version.CheckScrollRepoVersion(login.Message.ProverVersion, l.cfg.ProverManager.Verifier.HighVersionCircuit.MinProverVersion) {
		return fmt.Errorf("incompatible prover version. please upgrade your prover, minimum allowed version: %s, actual version: %s",
			l.cfg.ProverManager.Verifier.HighVersionCircuit.MinProverVersion, login.Message.ProverVersion)
	}

	if len(login.Message.ProverTypes) > 0 {
		vks := make(map[string]struct{})
		for _, proverType := range login.Message.ProverTypes {
			switch proverType {
			case types.ProverTypeChunk:
				for vk := range l.chunkVks {
					vks[vk] = struct{}{}
				}
			case types.ProverTypeBatch:
				for vk := range l.batchVKs {
					vks[vk] = struct{}{}
				}
				for vk := range l.bundleVks {
					vks[vk] = struct{}{}
				}
			default:
				log.Error("invalid prover_type", "value", proverType, "prover name", login.Message.ProverName, "prover_version", login.Message.ProverVersion)
			}
		}

		for _, vk := range login.Message.VKs {
			if _, ok := vks[vk]; !ok {
				log.Error("vk inconsistency", "prover vk", vk, "prover name", login.Message.ProverName,
					"prover_version", login.Message.ProverVersion, "message", login.Message)
				if !version.CheckScrollProverVersion(login.Message.ProverVersion) {
					return fmt.Errorf("incompatible prover version. please upgrade your prover, expect version: %s, actual version: %s",
						version.Version, login.Message.ProverVersion)
				}
				// if the prover reports a same prover version
				return errors.New("incompatible vk. please check your params files or config files")
			}
		}
	}
	return nil
}

// ProverHardName retrieves hard fork name which prover belongs to
func (l *LoginLogic) ProverHardName(login *types.LoginParameter) (string, error) {
	if _, ok := l.proverVersionHardForkMap[login.Message.ProverVersion]; ok {
		return l.proverVersionHardForkMap[login.Message.ProverVersion], nil
	}
	return "", fmt.Errorf("invalid prover prover_version:%s", login.Message.ProverVersion)
}
