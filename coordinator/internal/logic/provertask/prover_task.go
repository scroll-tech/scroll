package provertask

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"

	"scroll-tech/common/version"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// ErrCoordinatorInternalFailure coordinator internal db failure
var ErrCoordinatorInternalFailure = fmt.Errorf("coordinator internal error")

// ErrHardForkName indicates client request with the wrong hard fork name
var ErrHardForkName = fmt.Errorf("wrong hard fork name")

// ProverTask the interface of a collector who send data to prover
type ProverTask interface {
	Assign(ctx *gin.Context, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error)
}

// BaseProverTask a base prover task which contain series functions
type BaseProverTask struct {
	cfg *config.Config
	db  *gorm.DB
	vk  string

	nameForkMap map[string]uint64
	forkHeights []uint64

	batchOrm           *orm.Batch
	chunkOrm           *orm.Chunk
	blockOrm           *orm.L2Block
	proverTaskOrm      *orm.ProverTask
	proverBlockListOrm *orm.ProverBlockList
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

	if !version.CheckScrollRepoVersion(proverVersion.(string), b.cfg.ProverManager.MinProverVersion) {
		return nil, fmt.Errorf("incompatible prover version. please upgrade your prover, minimum allowed version: %s, actual version: %s", b.cfg.ProverManager.MinProverVersion, proverVersion.(string))
	}

	// if the prover has a different vk
	if getTaskParameter.VK != b.vk {
		// if the prover reports a different prover version
		if !version.CheckScrollProverVersion(proverVersion.(string)) {
			return nil, fmt.Errorf("incompatible prover version. please upgrade your prover, expect version: %s, actual version: %s", version.Version, proverVersion.(string))
		}
		// if the prover reports a same prover version
		return nil, fmt.Errorf("incompatible vk. please check your params files or config files")
	}

	isBlocked, err := b.proverBlockListOrm.IsPublicKeyBlocked(ctx, publicKey.(string))
	if err != nil {
		return nil, fmt.Errorf("failed to check whether the public key %s is blocked before assigning a chunk task, err: %w, proverName: %s, proverVersion: %s", publicKey, err, proverName, proverVersion)
	}
	if isBlocked {
		return nil, fmt.Errorf("public key %s is blocked from fetching tasks. ProverName: %s, ProverVersion: %s", publicKey, proverName, proverVersion)
	}

	isAssigned, err := b.proverTaskOrm.IsProverAssigned(ctx, publicKey.(string))
	if err != nil {
		return nil, fmt.Errorf("failed to check if prover %s is assigned a task, err: %w", publicKey.(string), err)
	}

	if isAssigned {
		return nil, fmt.Errorf("prover with publicKey %s is already assigned a task. ProverName: %s, ProverVersion: %s", publicKey, proverName, proverVersion)
	}
	return &ptc, nil
}

func (b *BaseProverTask) getHardForkNumberByName(forkName string) (uint64, error) {
	// when the first hard fork upgrade, the prover don't pass the fork_name to coordinator.
	// so coordinator need to be compatible.
	if forkName == "" {
		return 0, nil
	}

	hardForkNumber, exist := b.nameForkMap[forkName]
	if !exist {
		return 0, ErrHardForkName
	}

	return hardForkNumber, nil
}

var (
	getTaskCounterInitOnce sync.Once
	getTaskCounterVec      *prometheus.CounterVec = nil
)

func newGetTaskCounterVec(factory promauto.Factory, taskType string) *prometheus.CounterVec {
	getTaskCounterInitOnce.Do(func() {
		getTaskCounterVec = factory.NewCounterVec(prometheus.CounterOpts{
			Name: "coordinator_get_task_count",
			Help: "Multi dimensions get task counter.",
		}, []string{"task_type",
			coordinatorType.LabelProverName,
			coordinatorType.LabelProverPublicKey,
			coordinatorType.LabelProverVersion})
	})

	return getTaskCounterVec.MustCurryWith(prometheus.Labels{"task_type": taskType})
}
