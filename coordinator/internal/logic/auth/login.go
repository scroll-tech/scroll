package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	cfg          *config.VerifierConfig
	deduplicator ChallengeDeduplicator

	openVmVks map[string]struct{}

	proverVersionHardForkMap map[string][]string
}

type ChallengeDeduplicator interface {
	InsertChallenge(ctx context.Context, challengeString string) error
}

type SimpleDeduplicator struct {
}

func (s *SimpleDeduplicator) InsertChallenge(ctx context.Context, challengeString string) error {
	return nil
}

// NewLoginLogicWithSimpleDEduplicator new a LoginLogic, do not use db to deduplicate challege
func NewLoginLogicWithSimpleDEduplicator(vcfg *config.VerifierConfig, vf *verifier.Verifier) *LoginLogic {
	return newLoginLogic(&SimpleDeduplicator{}, vcfg, vf)
}

// NewLoginLogic new a LoginLogic
func NewLoginLogic(db *gorm.DB, vcfg *config.VerifierConfig, vf *verifier.Verifier) *LoginLogic {
	return newLoginLogic(orm.NewChallenge(db), vcfg, vf)
}

func newLoginLogic(deduplicator ChallengeDeduplicator, vcfg *config.VerifierConfig, vf *verifier.Verifier) *LoginLogic {
	proverVersionHardForkMap := make(map[string][]string)

	var hardForks []string
	for _, cfg := range vcfg.Verifiers {
		hardForks = append(hardForks, cfg.ForkName)
	}
	proverVersionHardForkMap[vcfg.MinProverVersion] = hardForks

	return &LoginLogic{
		cfg:                      vcfg,
		openVmVks:                vf.OpenVMVkMap,
		deduplicator:             deduplicator,
		proverVersionHardForkMap: proverVersionHardForkMap,
	}
}

// Verify the completeness of login message
func VerifyMsg(login *types.LoginParameter) error {
	verify, err := login.Verify()
	if err != nil || !verify {
		log.Error("auth message verify failure", "prover_name", login.Message.ProverName,
			"prover_version", login.Message.ProverVersion, "message", login.Message)
		return errors.New("auth message verify failure")
	}
	return nil
}

// InsertChallengeString insert and check the challenge string is existed
func (l *LoginLogic) InsertChallengeString(ctx *gin.Context, challenge string) error {
	return l.deduplicator.InsertChallenge(ctx.Copy(), challenge)
}

// Check if the login client is compatible with the setting in coordinator
func (l *LoginLogic) CompatiblityCheck(login *types.LoginParameter) error {

	if !version.CheckScrollRepoVersion(login.Message.ProverVersion, l.cfg.MinProverVersion) {
		return fmt.Errorf("incompatible prover version. please upgrade your prover, minimum allowed version: %s, actual version: %s", l.cfg.MinProverVersion, login.Message.ProverVersion)
	}

	vks := make(map[string]struct{})
	for vk := range l.openVmVks {
		vks[vk] = struct{}{}
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

	switch login.Message.ProverProviderType {
	case types.ProverProviderTypeInternal:
	case types.ProverProviderTypeExternal:
	case types.ProverProviderTypeProxy:
	case types.ProverProviderTypeUndefined:
		// for backward compatibility, set ProverProviderType as internal
		login.Message.ProverProviderType = types.ProverProviderTypeInternal
	default:
		log.Error("invalid prover_provider_type", "value", login.Message.ProverProviderType, "prover name", login.Message.ProverName, "prover version", login.Message.ProverVersion)
		return errors.New("invalid prover provider type.")
	}

	return nil
}

// ProverHardForkName retrieves hard fork name which prover belongs to
func (l *LoginLogic) ProverHardForkName(login *types.LoginParameter) (string, error) {
	proverVersionSplits := strings.Split(login.Message.ProverVersion, "-")
	if len(proverVersionSplits) == 0 {
		return "", fmt.Errorf("invalid prover prover_version:%s", login.Message.ProverVersion)
	}

	proverVersion := proverVersionSplits[0]
	if hardForkNames, ok := l.proverVersionHardForkMap[proverVersion]; ok {
		return strings.Join(hardForkNames, ","), nil
	}

	return "", fmt.Errorf("invalid prover prover_version:%s", login.Message.ProverVersion)
}
