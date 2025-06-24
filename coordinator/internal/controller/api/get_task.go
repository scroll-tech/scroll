package api

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/provertask"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// GetTaskController the get prover task api controller
type GetTaskController struct {
	proverTasks map[message.ProofType]provertask.ProverTask

	getTaskAccessCounter *prometheus.CounterVec

	l2gethClient    *ethclient.Client
	lastBlockNumber struct {
		sync.RWMutex
		data uint64
		t    time.Time
	}
}

// NewGetTaskController create a get prover task controller
func NewGetTaskController(cfg *config.Config, chainCfg *params.ChainConfig, db *gorm.DB, reg prometheus.Registerer) *GetTaskController {
	chunkProverTask := provertask.NewChunkProverTask(cfg, chainCfg, db, reg)
	batchProverTask := provertask.NewBatchProverTask(cfg, chainCfg, db, reg)
	bundleProverTask := provertask.NewBundleProverTask(cfg, chainCfg, db, reg)

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

	if cfg.L2 == nil || cfg.L2.Endpoint == nil {
		log.Crit("l2 endpoint is not set in config")
	} else {
		var err error
		ptc.l2gethClient, err = ethclient.Dial(cfg.L2.Endpoint.Url)
		if err != nil {
			log.Crit("dial l2geth endpoint fail", "err", err)
		}
	}

	return ptc
}

// getLatestBlockNumber gets the latest block number, using cache if available and not expired
func (ptc *GetTaskController) getLatestBlockNumber(ctx *gin.Context) (uint64, error) {
	// First check if we have a cached value that's still valid
	ptc.lastBlockNumber.RLock()
	if !ptc.lastBlockNumber.t.IsZero() && time.Since(ptc.lastBlockNumber.t) < time.Second*10 {
		blockNumber := ptc.lastBlockNumber.data
		ptc.lastBlockNumber.RUnlock()
		return blockNumber, nil
	}
	ptc.lastBlockNumber.RUnlock()

	// If not cached or expired, fetch from the client
	if ptc.l2gethClient == nil {
		return 0, errors.New("L2 geth client not initialized")
	}

	blockNumber, err := ptc.l2gethClient.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block number: %w", err)
	}

	// Update the cache
	ptc.lastBlockNumber.Lock()
	ptc.lastBlockNumber.data = blockNumber
	ptc.lastBlockNumber.t = time.Now()
	ptc.lastBlockNumber.Unlock()

	log.Debug("updated block height reference", "height", blockNumber)
	return blockNumber, nil
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
		if blk, err := ptc.getLatestBlockNumber(ctx); err == nil {
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
