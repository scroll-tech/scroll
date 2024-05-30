package provertask

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/version"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

var (
	// ErrCoordinatorInternalFailure coordinator internal db failure
	ErrCoordinatorInternalFailure = fmt.Errorf("coordinator internal error")
	// ErrHardForkName indicates client request with the wrong hard fork name
	ErrHardForkName = fmt.Errorf("wrong hard fork name")
)

// ProverTask the interface of a collector who send data to prover
type ProverTask interface {
	Assign(ctx *gin.Context, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error)
}

func reverseMap(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for k, v := range input {
		output[v] = k
	}
	return output
}

// BaseProverTask a base prover task which contain series functions
type BaseProverTask struct {
	cfg *config.Config
	db  *gorm.DB

	// key is hardForkName, value is vk
	vkMap map[string]string
	// key is vk, value is hardForkName
	reverseVkMap map[string]string
	nameForkMap  map[string]uint64
	forkHeights  []uint64

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
	HardForkName  string
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

	// signals that the prover is multi-circuits version
	if len(getTaskParameter.VKs) > 0 {
		if len(getTaskParameter.VKs) != 2 {
			return nil, fmt.Errorf("parameter vks length must be 2")
		}
		// min prover version supporting multi circuits, maybe put it to config file?
		var minMultiCircuitsProverVersion = "v4.4.7"
		if !version.CheckScrollRepoVersion(ptc.ProverVersion, minMultiCircuitsProverVersion) {
			return nil, fmt.Errorf("incompatible prover version. please upgrade your prover, minimum allowed version: %s, actual version: %s", minMultiCircuitsProverVersion, ptc.ProverVersion)
		}
		for _, vk := range getTaskParameter.VKs {
			if _, exists := b.reverseVkMap[vk]; !exists {
				return nil, fmt.Errorf("incompatible vk. vk %s is invalid", vk)
			}
		}
	} else {
		hardForkName, hardForkNameExist := ctx.Get(coordinatorType.HardForkName)
		if !hardForkNameExist {
			return nil, fmt.Errorf("get hard fork name from context failed")
		}
		ptc.HardForkName = hardForkName.(string)

		vk, vkExist := b.vkMap[ptc.HardForkName]
		if !vkExist {
			return nil, fmt.Errorf("can't get vk for hard fork:%s, vkMap:%v", ptc.HardForkName, b.vkMap)
		}

		// if the prover has a different vk
		if getTaskParameter.VK != vk {
			log.Error("vk inconsistency", "prover vk", getTaskParameter.VK, "vk", vk, "hardForkName", ptc.HardForkName)
			// if the prover reports a different prover version
			if !version.CheckScrollProverVersion(proverVersion.(string)) {
				return nil, fmt.Errorf("incompatible prover version. please upgrade your prover, expect version: %s, actual version: %s", version.Version, proverVersion.(string))
			}
			// if the prover reports a same prover version
			return nil, fmt.Errorf("incompatible vk. please check your params files or config files")
		}
	}

	isBlocked, err := b.proverBlockListOrm.IsPublicKeyBlocked(ctx.Copy(), publicKey.(string))
	if err != nil {
		return nil, fmt.Errorf("failed to check whether the public key %s is blocked before assigning a chunk task, err: %w, proverName: %s", publicKey, err, proverName)
	}
	if isBlocked {
		return nil, fmt.Errorf("public key %s is blocked from fetching tasks. ProverName: %s, ProverVersion: %s", publicKey, proverName, ptc.ProverVersion)
	}

	isAssigned, err := b.proverTaskOrm.IsProverAssigned(ctx.Copy(), publicKey.(string))
	if err != nil {
		return nil, fmt.Errorf("failed to check if prover %s is assigned a task, err: %w", publicKey.(string), err)
	}

	if isAssigned {
		return nil, fmt.Errorf("prover with publicKey %s is already assigned a task. ProverName: %s", publicKey, proverName)
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
