package provertask

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
	cutils "scroll-tech/coordinator/internal/utils"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"
)

// BundleProverTask is prover task implement for bundle proof
type BundleProverTask struct {
	BaseProverTask

	bundleTaskGetTaskTotal  *prometheus.CounterVec
	bundleTaskGetTaskProver *prometheus.CounterVec
}

// NewBundleProverTask new a bundle collector
func NewBundleProverTask(cfg *config.Config, chainCfg *params.ChainConfig, db *gorm.DB, expectedVk map[string][]byte, reg prometheus.Registerer) *BundleProverTask {
	bp := &BundleProverTask{
		BaseProverTask: BaseProverTask{
			db:                 db,
			chainCfg:           chainCfg,
			cfg:                cfg,
			expectedVk:         expectedVk,
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
	if taskCtx.ProverProviderType == uint8(coordinatorType.ProverProviderTypeExternal) {
		unassignedBundleCount, getCountError := bp.bundleOrm.GetUnassignedBundleCount(ctx.Copy(), maxActiveAttempts, maxTotalAttempts)
		if getCountError != nil {
			log.Error("failed to get unassigned bundle proving tasks count", "height", getTaskParameter.ProverHeight, "err", getCountError)
			return nil, ErrCoordinatorInternalFailure
		}
		// Assign external prover if unassigned task number exceeds threshold
		if unassignedBundleCount < bp.cfg.ProverManager.ExternalProverThreshold {
			return nil, nil
		}
	}

	var bundleTask *orm.Bundle
	var hardForkName string
	for i := 0; i < 5; i++ {
		var getTaskError error
		var tmpBundleTask *orm.Bundle

		if taskCtx.hasAssignedTask != nil {
			tmpBundleTask, getTaskError = bp.bundleOrm.GetBundleByHash(ctx.Copy(), taskCtx.hasAssignedTask.TaskID)
			if getTaskError != nil {
				log.Error("failed to get bundle has assigned to prover", "taskID", taskCtx.hasAssignedTask.TaskID, "err", getTaskError)
				return nil, ErrCoordinatorInternalFailure
			} else if tmpBundleTask == nil {
				// if the assigned chunk dropped, there would be too much issue to assign another
				return nil, fmt.Errorf("prover with publicKey %s is already assigned a dropped bundle. ProverName: %s, ProverVersion: %s",
					taskCtx.PublicKey, taskCtx.ProverName, taskCtx.ProverVersion)
			}
		}

		if tmpBundleTask == nil {
			tmpBundleTask, getTaskError = bp.bundleOrm.GetAssignedBundle(ctx.Copy(), maxActiveAttempts, maxTotalAttempts)
			if getTaskError != nil {
				log.Error("failed to get assigned bundle proving tasks", "height", getTaskParameter.ProverHeight, "err", getTaskError)
				return nil, ErrCoordinatorInternalFailure
			}
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

		taskCtx.taskType = message.ProofTypeBundle
		taskCtx.bundleTask = tmpBundleTask

		var checkErr error
		hardForkName, checkErr = bp.hardForkSanityCheck(ctx, taskCtx)
		if checkErr != nil {
			log.Debug("hard fork sanity check failed", "height", getTaskParameter.ProverHeight, "err", checkErr)
			return nil, nil
		}

		// we are simply pick the chunk which has been assigned, so don't bother to update attempts or check failed before
		if taskCtx.hasAssignedTask == nil {
			// Don't dispatch the same failing job to the same prover
			proverTasks, getTaskError := bp.proverTaskOrm.GetFailedProverTasksByHash(ctx.Copy(), message.ProofTypeBundle, tmpBundleTask.Hash, 2)
			if getTaskError != nil {
				log.Error("failed to get prover tasks", "proof type", message.ProofTypeBundle.String(), "task ID", tmpBundleTask.Hash, "error", getTaskError)
				return nil, ErrCoordinatorInternalFailure
			}
			for i := 0; i < len(proverTasks); i++ {
				if proverTasks[i].ProverPublicKey == taskCtx.PublicKey ||
					taskCtx.ProverProviderType == uint8(coordinatorType.ProverProviderTypeExternal) && cutils.IsExternalProverNameMatch(proverTasks[i].ProverName, taskCtx.ProverName) {
					log.Debug("get empty bundle, the prover already failed this task", "height", getTaskParameter.ProverHeight, "task ID", tmpBundleTask.Hash, "prover name", taskCtx.ProverName, "prover public key", taskCtx.PublicKey)
					return nil, nil
				}
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
		}
		bundleTask = tmpBundleTask
		break
	}

	if bundleTask == nil {
		log.Debug("get empty unassigned bundle after retry 5 times", "height", getTaskParameter.ProverHeight)
		return nil, nil
	}

	log.Info("start bundle proof generation session", "task index", bundleTask.Index, "public key", taskCtx.PublicKey, "prover name", taskCtx.ProverName)
	var proverTask *orm.ProverTask
	if taskCtx.hasAssignedTask == nil {
		proverTask = &orm.ProverTask{
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
	} else {
		proverTask = taskCtx.hasAssignedTask
	}

	taskMsg, err := bp.formatProverTask(ctx.Copy(), proverTask, hardForkName)
	if err != nil {
		bp.recoverActiveAttempts(ctx, bundleTask)
		log.Error("format bundle prover task failure", "task_id", bundleTask.Hash, "err", err)
		return nil, ErrCoordinatorInternalFailure
	}
	if getTaskParameter.Universal {
		var metadata []byte
		taskMsg, metadata, err = bp.applyUniversal(taskMsg)
		if err != nil {
			bp.recoverActiveAttempts(ctx, bundleTask)
			log.Error("Generate universal prover task failure", "task_id", bundleTask.Hash, "type", "bundle")
			return nil, ErrCoordinatorInternalFailure
		}
		// bundle proof require snark
		taskMsg.UseSnark = true
		proverTask.Metadata = metadata
	}

	// Store session info.
	if taskCtx.hasAssignedTask == nil {
		if err = bp.proverTaskOrm.InsertProverTask(ctx.Copy(), proverTask); err != nil {
			bp.recoverActiveAttempts(ctx, bundleTask)
			log.Error("insert bundle prover task info fail", "task_id", bundleTask.Hash, "publicKey", taskCtx.PublicKey, "err", err)
			return nil, ErrCoordinatorInternalFailure
		}
	}
	// notice uuid is set as a side effect of InsertProverTask
	taskMsg.UUID = proverTask.UUID.String()

	bp.bundleTaskGetTaskTotal.WithLabelValues(hardForkName).Inc()
	bp.bundleTaskGetTaskProver.With(prometheus.Labels{
		coordinatorType.LabelProverName:      proverTask.ProverName,
		coordinatorType.LabelProverPublicKey: proverTask.ProverPublicKey,
		coordinatorType.LabelProverVersion:   proverTask.ProverVersion,
	}).Inc()

	return taskMsg, nil
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

	parentBatch, err := bp.batchOrm.GetBatchByHash(ctx, batches[0].ParentBatchHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent batch for batch task id:%s err:%w", task.TaskID, err)
	}

	var batchProofs []*message.OpenVMBatchProof
	for _, batch := range batches {
		var proof message.OpenVMBatchProof
		if encodeErr := json.Unmarshal(batch.Proof, &proof); encodeErr != nil {
			return nil, fmt.Errorf("failed to unmarshal proof: %w, bundle hash: %v, batch hash: %v", encodeErr, task.TaskID, batch.Hash)
		}
		batchProofs = append(batchProofs, &proof)
	}

	taskDetail := message.BundleTaskDetail{
		BatchProofs: batchProofs,
	}

	if hardForkName == message.EuclidV2Fork {
		taskDetail.ForkName = message.EuclidV2ForkNameForProver
	} else {
		log.Error("unsupported hard fork name", "hard_fork_name", hardForkName)
		return nil, fmt.Errorf("unsupported hard fork name: %s", hardForkName)
	}

	taskDetail.BundleInfo = &message.OpenVMBundleInfo{
		ChainID:       bp.cfg.L2.ChainID,
		PrevStateRoot: common.HexToHash(parentBatch.StateRoot),
		PostStateRoot: common.HexToHash(batches[len(batches)-1].StateRoot),
		WithdrawRoot:  common.HexToHash(batches[len(batches)-1].WithdrawRoot),
		NumBatches:    uint32(len(batches)),
		PrevBatchHash: common.HexToHash(batches[0].ParentBatchHash),
		BatchHash:     common.HexToHash(batches[len(batches)-1].Hash),
		MsgQueueHash:  common.HexToHash(batches[len(batches)-1].PostL1MessageQueueHash),
	}

	batchProofsBytes, err := json.Marshal(taskDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch proofs, taskID:%s err:%w", task.TaskID, err)
	}

	taskMsg := &coordinatorType.GetTaskSchema{
		TaskID:       task.TaskID,
		TaskType:     int(message.ProofTypeBundle),
		TaskData:     string(batchProofsBytes),
		HardForkName: hardForkName,
	}

	log.Debug("TaskData", "task_id", task.TaskID, "task_type", message.ProofTypeBundle.String(), "hard_fork_name", hardForkName, "task_data", taskMsg.TaskData)

	return taskMsg, nil
}

func (bp *BundleProverTask) recoverActiveAttempts(ctx *gin.Context, bundleTask *orm.Bundle) {
	if err := bp.bundleOrm.DecreaseActiveAttemptsByHash(ctx.Copy(), bundleTask.Hash); err != nil {
		log.Error("failed to recover bundle active attempts", "hash", bundleTask.Hash, "error", err)
	}
}
