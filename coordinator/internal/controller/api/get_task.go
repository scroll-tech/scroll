package api

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/provertask"
	"scroll-tech/coordinator/internal/logic/verifier"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// GetTaskController the get prover task api controller
type GetTaskController struct {
	proverTasks map[message.ProofType]provertask.ProverTask

	getTaskAccessCounter *prometheus.CounterVec

	l2syncer *l2Syncer
}

// NewGetTaskController create a get prover task controller
func NewGetTaskController(cfg *config.Config, chainCfg *params.ChainConfig, db *gorm.DB, verifier *verifier.Verifier, reg prometheus.Registerer) *GetTaskController {
	chunkProverTask := provertask.NewChunkProverTask(cfg, chainCfg, db, verifier.ChunkVk, reg)
	batchProverTask := provertask.NewBatchProverTask(cfg, chainCfg, db, verifier.BatchVk, reg)
	bundleProverTask := provertask.NewBundleProverTask(cfg, chainCfg, db, verifier.BundleVk, reg)

	ptc := &GetTaskController{
		proverTasks: make(map[message.ProofType]provertask.ProverTask),
		getTaskAccessCounter: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "coordinator_get_task_access_count",
			Help: "Multi dimensions get task counter.",
		}, []string{coordinatorType.LabelProverName, coordinatorType.LabelProverPublicKey, coordinatorType.LabelProverVersion}),
	}

	ptc.proverTasks[message.ProofTypeChunk] = chunkProverTask
	ptc.proverTasks[message.ProofTypeBatch] = batchProverTask
	ptc.proverTasks[message.ProofTypeBundle] = bundleProverTask

	if syncer, err := createL2Syncer(cfg); err != nil {
		log.Crit("can not init l2 syncer", "err", err)
	} else {
		ptc.l2syncer = syncer
	}

	return ptc
}

func (ptc *GetTaskController) incGetTaskAccessCounter(ctx *gin.Context) error {
	publicKey, publicKeyExist := ctx.Get(coordinatorType.PublicKey)
	if !publicKeyExist {
		return errors.New("get public key from context failed")
	}
	proverName, proverNameExist := ctx.Get(coordinatorType.ProverName)
	if !proverNameExist {
		return errors.New("get prover name from context failed")
	}
	proverVersion, proverVersionExist := ctx.Get(coordinatorType.ProverVersion)
	if !proverVersionExist {
		return errors.New("get prover version from context failed")
	}

	ptc.getTaskAccessCounter.With(prometheus.Labels{
		coordinatorType.LabelProverPublicKey: publicKey.(string),
		coordinatorType.LabelProverName:      proverName.(string),
		coordinatorType.LabelProverVersion:   proverVersion.(string),
	}).Inc()
	return nil
}

// GetTasks get assigned chunk/batch task
func (ptc *GetTaskController) GetTasks(ctx *gin.Context) {
	var getTaskParameter coordinatorType.GetTaskParameter
	if err := ctx.ShouldBind(&getTaskParameter); err != nil {
		nerr := fmt.Errorf("prover task parameter invalid, err:%w", err)
		types.RenderFailure(ctx, types.ErrCoordinatorParameterInvalidNo, nerr)
		return
	}

	if getTaskParameter.ProverHeight == 0 {
		// help update the prover height with internal l2geth
		if blk, err := ptc.l2syncer.getLatestBlockNumber(ctx); err == nil {
			getTaskParameter.ProverHeight = blk
		} else {
			nerr := fmt.Errorf("inner l2geth failure, err:%w", err)
			types.RenderFailure(ctx, types.InternalServerError, nerr)
			return
		}
	}

	proofType := ptc.proofType(&getTaskParameter)
	proverTask, isExist := ptc.proverTasks[proofType]
	if !isExist {
		nerr := fmt.Errorf("parameter wrong proof type:%v", proofType)
		types.RenderFailure(ctx, types.ErrCoordinatorParameterInvalidNo, nerr)
		return
	}

	if err := ptc.incGetTaskAccessCounter(ctx); err != nil {
		log.Warn("get_task access counter inc failed", "error", err.Error())
	}

	result, err := proverTask.Assign(ctx, &getTaskParameter)
	if err != nil {
		nerr := fmt.Errorf("return prover task err:%w", err)
		types.RenderFailure(ctx, types.ErrCoordinatorGetTaskFailure, nerr)
		return
	}

	if result == nil {
		nerr := errors.New("get empty prover task")
		types.RenderFailure(ctx, types.ErrCoordinatorEmptyProofData, nerr)
		return
	}

	types.RenderSuccess(ctx, result)
}

func (ptc *GetTaskController) proofType(para *coordinatorType.GetTaskParameter) message.ProofType {
	var proofTypes []message.ProofType
	for _, proofType := range para.TaskTypes {
		proofTypes = append(proofTypes, message.ProofType(proofType))
	}

	if len(proofTypes) == 0 {
		proofTypes = []message.ProofType{
			message.ProofTypeChunk,
			message.ProofTypeBatch,
			message.ProofTypeBundle,
		}
	}

	rand.Shuffle(len(proofTypes), func(i, j int) {
		proofTypes[i], proofTypes[j] = proofTypes[j], proofTypes[i]
	})
	return proofTypes[0]
}
