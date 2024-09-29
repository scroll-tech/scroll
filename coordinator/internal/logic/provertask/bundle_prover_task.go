package provertask

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/common/forks"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// BundleProverTask is prover task implement for bundle proof
type BundleProverTask struct {
	BaseProverTask

	bundleTaskGetTaskTotal  *prometheus.CounterVec
	bundleTaskGetTaskProver *prometheus.CounterVec
}

// NewBundleProverTask new a bundle collector
func NewBundleProverTask(cfg *config.Config, chainCfg *params.ChainConfig, db *gorm.DB, reg prometheus.Registerer) *BundleProverTask {
	bp := &BundleProverTask{
		BaseProverTask: BaseProverTask{
			db:                 db,
			chainCfg:           chainCfg,
			cfg:                cfg,
			blockOrm:           orm.NewL2Block(db),
			chunkOrm:           orm.NewChunk(db),
			batchOrm:           orm.NewBatch(db),
			bundleOrm:          orm.NewBundle(db),
			proverTaskOrm:      orm.NewProverTask(db),
			proverBlockListOrm: orm.NewProverBlockList(db),
		},
		bundleTaskGetTaskTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "coordinator_bundle_get_task_total",
			Help: "Total number of bundle get task.",
		}, []string{"fork_name"}),
		bundleTaskGetTaskProver: newGetTaskCounterVec(promauto.With(reg), "bundle"),
	}
	return bp
}

// Assign load and assign batch tasks
func (bp *BundleProverTask) Assign(ctx *gin.Context, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error) {
	taskCtx, err := bp.checkParameter(ctx)
	if err != nil || taskCtx == nil {
		return nil, fmt.Errorf("check prover task parameter failed, error:%w", err)
	}

	maxActiveAttempts := bp.cfg.ProverManager.ProversPerSession
	maxTotalAttempts := bp.cfg.ProverManager.SessionAttempts
	var bundleTask *orm.Bundle
	for i := 0; i < 5; i++ {
		var getTaskError error
		var tmpBundleTask *orm.Bundle
		tmpBundleTask, getTaskError = bp.bundleOrm.GetAssignedBundle(ctx.Copy(), maxActiveAttempts, maxTotalAttempts)
		if getTaskError != nil {
			log.Error("failed to get assigned bundle proving tasks", "height", getTaskParameter.ProverHeight, "err", getTaskError)
			return nil, ErrCoordinatorInternalFailure
		}

		// Why here need get again? In order to support a task can assign to multiple prover, need also assign `ProvingTaskAssigned`
		// bundle to prover. But use `proving_status in (1, 2)` will not use the postgres index. So need split the sql.
		if tmpBundleTask == nil {
			tmpBundleTask, getTaskError = bp.bundleOrm.GetUnassignedBundle(ctx.Copy(), maxActiveAttempts, maxTotalAttempts)
			if getTaskError != nil {
				log.Error("failed to get unassigned bundle proving tasks", "height", getTaskParameter.ProverHeight, "err", getTaskError)
				return nil, ErrCoordinatorInternalFailure
			}
		}

		if tmpBundleTask == nil {
			log.Debug("get empty bundle", "height", getTaskParameter.ProverHeight)
			return nil, nil
		}

		rowsAffected, updateAttemptsErr := bp.bundleOrm.UpdateBundleAttempts(ctx.Copy(), tmpBundleTask.Hash, tmpBundleTask.ActiveAttempts, tmpBundleTask.TotalAttempts)
		if updateAttemptsErr != nil {
			log.Error("failed to update bundle attempts", "height", getTaskParameter.ProverHeight, "err", updateAttemptsErr)
			return nil, ErrCoordinatorInternalFailure
		}

		if rowsAffected == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		bundleTask = tmpBundleTask
		break
	}

	if bundleTask == nil {
		log.Debug("get empty unassigned bundle after retry 5 times", "height", getTaskParameter.ProverHeight)
		return nil, nil
	}

	log.Info("start bundle proof generation session", "task index", bundleTask.Index, "public key", taskCtx.PublicKey, "prover name", taskCtx.ProverName)

	hardForkName, getHardForkErr := bp.hardForkName(ctx, bundleTask)
	if getHardForkErr != nil {
		bp.recoverActiveAttempts(ctx, bundleTask)
		log.Error("retrieve hard fork name by bundle failed", "task_id", bundleTask.Hash, "err", getHardForkErr)
		return nil, ErrCoordinatorInternalFailure
	}

	//if _, ok := taskCtx.HardForkNames[hardForkName]; !ok {
	//	bp.recoverActiveAttempts(ctx, bundleTask)
	//	log.Error("incompatible prover version",
	//		"requisite hard fork name", hardForkName,
	//		"prover hard fork name", taskCtx.HardForkNames,
	//		"task_id", bundleTask.Hash)
	//	return nil, ErrCoordinatorInternalFailure
	//}

	proverTask := orm.ProverTask{
		TaskID:          bundleTask.Hash,
		ProverPublicKey: taskCtx.PublicKey,
		TaskType:        int16(message.ProofTypeBundle),
		ProverName:      taskCtx.ProverName,
		ProverVersion:   taskCtx.ProverVersion,
		ProvingStatus:   int16(types.ProverAssigned),
		FailureType:     int16(types.ProverTaskFailureTypeUndefined),
		// here why need use UTC time. see scroll/common/database/db.go
		AssignedAt: utils.NowUTC(),
	}

	// Store session info.
	if err = bp.proverTaskOrm.InsertProverTask(ctx.Copy(), &proverTask); err != nil {
		bp.recoverActiveAttempts(ctx, bundleTask)
		log.Error("insert bundle prover task info fail", "task_id", bundleTask.Hash, "publicKey", taskCtx.PublicKey, "err", err)
		return nil, ErrCoordinatorInternalFailure
	}

	taskMsg, err := bp.formatProverTask(ctx.Copy(), &proverTask, hardForkName)
	if err != nil {
		bp.recoverActiveAttempts(ctx, bundleTask)
		log.Error("format bundle prover task failure", "task_id", bundleTask.Hash, "err", err)
		return nil, ErrCoordinatorInternalFailure
	}

	bp.bundleTaskGetTaskTotal.WithLabelValues(hardForkName).Inc()
	bp.bundleTaskGetTaskProver.With(prometheus.Labels{
		coordinatorType.LabelProverName:      proverTask.ProverName,
		coordinatorType.LabelProverPublicKey: proverTask.ProverPublicKey,
		coordinatorType.LabelProverVersion:   proverTask.ProverVersion,
	}).Inc()

	return taskMsg, nil
}

func (bp *BundleProverTask) hardForkName(ctx *gin.Context, bundleTask *orm.Bundle) (string, error) {
	startBatch, getBatchErr := bp.batchOrm.GetBatchByHash(ctx, bundleTask.StartBatchHash)
	if getBatchErr != nil {
		return "", getBatchErr
	}

	startChunk, getChunkErr := bp.chunkOrm.GetChunkByHash(ctx, startBatch.StartChunkHash)
	if getChunkErr != nil {
		return "", getChunkErr
	}

	l2Block, getBlockErr := bp.blockOrm.GetL2BlockByNumber(ctx.Copy(), startChunk.StartBlockNumber)
	if getBlockErr != nil {
		return "", getBlockErr
	}

	hardForkName := forks.GetHardforkName(bp.chainCfg, l2Block.Number, l2Block.BlockTimestamp)
	return hardForkName, nil
}

func (bp *BundleProverTask) formatProverTask(ctx context.Context, task *orm.ProverTask, hardForkName string) (*coordinatorType.GetTaskSchema, error) {
	// get bundle from db
	batches, err := bp.batchOrm.GetBatchesByBundleHash(ctx, task.TaskID)
	if err != nil {
		err = fmt.Errorf("failed to get batch proofs for batch task id:%s err:%w ", task.TaskID, err)
		return nil, err
	}

	if len(batches) == 0 {
		return nil, fmt.Errorf("failed to get batch proofs for bundle task id:%s, no batch found", task.TaskID)
	}

	var batchProofs []*message.BatchProof
	for _, batch := range batches {
		var proof message.BatchProof
		if encodeErr := json.Unmarshal(batch.Proof, &proof); encodeErr != nil {
			return nil, fmt.Errorf("failed to unmarshal proof: %w, bundle hash: %v, batch hash: %v", encodeErr, task.TaskID, batch.Hash)
		}
		batchProofs = append(batchProofs, &proof)
	}

	taskDetail := message.BundleTaskDetail{
		BatchProofs: batchProofs,
	}

	batchProofsBytes, err := json.Marshal(taskDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch proofs, taskID:%s err:%w", task.TaskID, err)
	}

	taskMsg := &coordinatorType.GetTaskSchema{
		UUID:         task.UUID.String(),
		TaskID:       task.TaskID,
		TaskType:     int(message.ProofTypeBundle),
		TaskData:     string(batchProofsBytes),
		HardForkName: hardForkName,
	}
	return taskMsg, nil
}

func (bp *BundleProverTask) recoverActiveAttempts(ctx *gin.Context, bundleTask *orm.Bundle) {
	if err := bp.bundleOrm.DecreaseActiveAttemptsByHash(ctx.Copy(), bundleTask.Hash); err != nil {
		log.Error("failed to recover bundle active attempts", "hash", bundleTask.Hash, "error", err)
	}
}
