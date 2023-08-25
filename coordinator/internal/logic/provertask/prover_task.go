package provertask

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"scroll-tech/common/version"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// ProverTask the interface of a collector who send data to prover
type ProverTask interface {
	Assign(ctx *gin.Context, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error)
}

// BaseProverTask a base prover task which contain series functions
type BaseProverTask struct {
	cfg *config.Config
	db  *gorm.DB
	vk  string

	batchOrm      *orm.Batch
	chunkOrm      *orm.Chunk
	blockOrm      *orm.L2Block
	proverTaskOrm *orm.ProverTask
}

type proverTaskContext struct {
	PublicKey     string
	ProverName    string
	ProverVersion string
}

// checkParameter check the prover task parameter illegal
func (b *BaseProverTask) checkParameter(ctx *gin.Context, getTaskParameter *coordinatorType.GetTaskParameter) (*proverTaskContext, error) {
	var ptc proverTaskContext

	publicKey, publicKeyExist := ctx.Get(coordinatorType.PublicKey)
	if !publicKeyExist {
		return nil, fmt.Errorf("get public key from context failed")
	}
	ptc.PublicKey = publicKey.(string)

	proverName, proverNameExist := ctx.Get(coordinatorType.ProverName)
	if !proverNameExist {
		return nil, fmt.Errorf("get prover name from context failed")
	}
	ptc.ProverName = proverName.(string)

	proverVersion, proverVersionExist := ctx.Get(coordinatorType.ProverVersion)
	if !proverVersionExist {
		return nil, fmt.Errorf("get prover version from context failed")
	}
	ptc.ProverVersion = proverVersion.(string)

	if getTaskParameter.VK == "" { // allow vk being empty, because for the first time the prover may not know its vk
		if !version.CheckScrollProverVersionTag(proverVersion.(string)) { // but reject too-old provers
			return nil, fmt.Errorf("incompatible prover version. please upgrade your prover, expect version: %s, actual version: %s", version.Version, proverVersion.(string))
		}
	} else if getTaskParameter.VK != b.vk { // non-empty vk but different
		if version.CheckScrollProverVersion(proverVersion.(string)) { // same prover version but different vks
			return nil, fmt.Errorf("incompatible vk. please check your params files or config files")
		}
		// different prover versions and different vks
		return nil, fmt.Errorf("incompatible prover version. please upgrade your prover, expect version: %s, actual version: %s", version.Version, proverVersion.(string))
	}

	isAssigned, err := b.proverTaskOrm.IsProverAssigned(ctx, publicKey.(string))
	if err != nil {
		return nil, fmt.Errorf("failed to check if prover is assigned a task: %w", err)
	}

	if isAssigned {
		return nil, fmt.Errorf("prover with publicKey %s is already assigned a task", publicKey)
	}
	return &ptc, nil
}
