package provertask

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
	cutils "scroll-tech/coordinator/internal/utils"
)

// BatchProverTask is prover task implement for batch proof
type BatchProverTask struct {
	BaseProverTask

	batchTaskGetTaskTotal  *prometheus.CounterVec
	batchTaskGetTaskProver *prometheus.CounterVec
}

// NewBatchProverTask new a batch collector
func NewBatchProverTask(cfg *config.Config, chainCfg *params.ChainConfig, db *gorm.DB, reg prometheus.Registerer) *BatchProverTask {
	bp := &BatchProverTask{
		BaseProverTask: BaseProverTask{
			db:                 db,
			cfg:                cfg,
			chainCfg:           chainCfg,
			blockOrm:           orm.NewL2Block(db),
			chunkOrm:           orm.NewChunk(db),
			batchOrm:           orm.NewBatch(db),
			proverTaskOrm:      orm.NewProverTask(db),
			proverBlockListOrm: orm.NewProverBlockList(db),
		},
		batchTaskGetTaskTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "coordinator_batch_get_task_total",
			Help: "Total number of batch get task.",
		}, []string{"fork_name"}),
		batchTaskGetTaskProver: newGetTaskCounterVec(promauto.With(reg), "batch"),
	}
	return bp
}

// Assign load and assign batch tasks
func (bp *BatchProverTask) Assign(ctx *gin.Context, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error) {
	taskCtx, err := bp.checkParameter(ctx)
	if err != nil || taskCtx == nil {
		return nil, fmt.Errorf("check prover task parameter failed, error:%w", err)
	}

	maxActiveAttempts := bp.cfg.ProverManager.ProversPerSession
	maxTotalAttempts := bp.cfg.ProverManager.SessionAttempts
	if taskCtx.ProverProviderType == uint8(coordinatorType.ProverProviderTypeExternal) {
		unassignedBatchCount, getCountError := bp.batchOrm.GetUnassignedBatchCount(ctx.Copy(), maxActiveAttempts, maxTotalAttempts)
		if getCountError != nil {
			log.Error("failed to get unassigned batch proving tasks count", "height", getTaskParameter.ProverHeight, "err", getCountError)
			return nil, ErrCoordinatorInternalFailure
		}
		// Assign external prover if unassigned task number exceeds threshold
		if unassignedBatchCount < bp.cfg.ProverManager.ExternalProverThreshold {
			return nil, nil
		}
	}

	var batchTask *orm.Batch
	var hardForkName string
	for i := 0; i < 5; i++ {
		var getTaskError error
		var tmpBatchTask *orm.Batch
		tmpBatchTask, getTaskError = bp.batchOrm.GetAssignedBatch(ctx.Copy(), maxActiveAttempts, maxTotalAttempts)
		if getTaskError != nil {
			log.Error("failed to get assigned batch proving tasks", "height", getTaskParameter.ProverHeight, "err", getTaskError)
			return nil, ErrCoordinatorInternalFailure
		}

		// Why here need get again? In order to support a task can assign to multiple prover, need also assign `ProvingTaskAssigned`
		// batch to prover. But use `proving_status in (1, 2)` will not use the postgres index. So need split the sql.
		if tmpBatchTask == nil {
			tmpBatchTask, getTaskError = bp.batchOrm.GetUnassignedBatch(ctx.Copy(), maxActiveAttempts, maxTotalAttempts)
			if getTaskError != nil {
				log.Error("failed to get unassigned batch proving tasks", "height", getTaskParameter.ProverHeight, "err", getTaskError)
				return nil, ErrCoordinatorInternalFailure
			}
		}

		if tmpBatchTask == nil {
			log.Debug("get empty batch", "height", getTaskParameter.ProverHeight)
			return nil, nil
		}

		taskCtx.taskType = message.ProofTypeBatch
		taskCtx.batchTask = tmpBatchTask

		var checkErr error
		hardForkName, checkErr = bp.hardForkSanityCheck(ctx, taskCtx)
		if checkErr != nil {
			log.Debug("hard fork sanity check failed", "height", getTaskParameter.ProverHeight, "err", checkErr)
			return nil, nil
		}

		// Don't dispatch the same failing job to the same prover
		proverTasks, getFailedTaskError := bp.proverTaskOrm.GetFailedProverTasksByHash(ctx.Copy(), message.ProofTypeBatch, tmpBatchTask.Hash, 2)
		if getFailedTaskError != nil {
			log.Error("failed to get prover tasks", "proof type", message.ProofTypeBatch.String(), "task ID", tmpBatchTask.Hash, "error", getFailedTaskError)
			return nil, ErrCoordinatorInternalFailure
		}
		for i := 0; i < len(proverTasks); i++ {
			if proverTasks[i].ProverPublicKey == taskCtx.PublicKey ||
				taskCtx.ProverProviderType == uint8(coordinatorType.ProverProviderTypeExternal) && cutils.IsExternalProverNameMatch(proverTasks[i].ProverName, taskCtx.ProverName) {
				log.Debug("get empty batch, the prover already failed this task", "height", getTaskParameter.ProverHeight, "task ID", tmpBatchTask.Hash, "prover name", taskCtx.ProverName, "prover public key", taskCtx.PublicKey)
				return nil, nil
			}
		}

		rowsAffected, updateAttemptsErr := bp.batchOrm.UpdateBatchAttempts(ctx.Copy(), tmpBatchTask.Index, tmpBatchTask.ActiveAttempts, tmpBatchTask.TotalAttempts)
		if updateAttemptsErr != nil {
			log.Error("failed to update batch attempts", "height", getTaskParameter.ProverHeight, "err", updateAttemptsErr)
			return nil, ErrCoordinatorInternalFailure
		}

		if rowsAffected == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		batchTask = tmpBatchTask
		break
	}

	if batchTask == nil {
		log.Debug("get empty unassigned batch after retry 5 times", "height", getTaskParameter.ProverHeight)
		return nil, nil
	}

	log.Info("start batch proof generation session", "task_id", batchTask.Hash, "public key", taskCtx.PublicKey, "prover name", taskCtx.ProverName)
	proverTask := orm.ProverTask{
		TaskID:          batchTask.Hash,
		ProverPublicKey: taskCtx.PublicKey,
		TaskType:        int16(message.ProofTypeBatch),
		ProverName:      taskCtx.ProverName,
		ProverVersion:   taskCtx.ProverVersion,
		ProvingStatus:   int16(types.ProverAssigned),
		FailureType:     int16(types.ProverTaskFailureTypeUndefined),
		// here why need use UTC time. see scroll/common/database/db.go
		AssignedAt: utils.NowUTC(),
	}

	// Store session info.
	if err = bp.proverTaskOrm.InsertProverTask(ctx.Copy(), &proverTask); err != nil {
		bp.recoverActiveAttempts(ctx, batchTask)
		log.Error("insert batch prover task info fail", "task_id", batchTask.Hash, "publicKey", taskCtx.PublicKey, "err", err)
		return nil, ErrCoordinatorInternalFailure
	}

	taskMsg, err := bp.formatProverTask(ctx.Copy(), &proverTask, batchTask, hardForkName)
	if err != nil {
		bp.recoverActiveAttempts(ctx, batchTask)
		log.Error("format prover task failure", "task_id", batchTask.Hash, "err", err)
		return nil, ErrCoordinatorInternalFailure
	}

	bp.batchTaskGetTaskTotal.WithLabelValues(hardForkName).Inc()
	bp.batchTaskGetTaskProver.With(prometheus.Labels{
		coordinatorType.LabelProverName:      proverTask.ProverName,
		coordinatorType.LabelProverPublicKey: proverTask.ProverPublicKey,
		coordinatorType.LabelProverVersion:   proverTask.ProverVersion,
	}).Inc()

	return taskMsg, nil
}

func (bp *BatchProverTask) formatProverTask(ctx context.Context, task *orm.ProverTask, batch *orm.Batch, hardForkName string) (*coordinatorType.GetTaskSchema, error) {
	// get chunk from db
	chunks, err := bp.chunkOrm.GetChunksByBatchHash(ctx, task.TaskID)
	if err != nil {
		err = fmt.Errorf("failed to get chunk proofs for batch task id:%s err:%w ", task.TaskID, err)
		return nil, err
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunk found for batch task id:%s", task.TaskID)
	}

	var chunkProofs []message.ChunkProof
	var chunkInfos []*message.ChunkInfo
	for _, chunk := range chunks {
		proof := message.NewChunkProof(hardForkName)
		if encodeErr := json.Unmarshal(chunk.Proof, &proof); encodeErr != nil {
			return nil, fmt.Errorf("Chunk.GetProofsByBatchHash unmarshal proof error: %w, batch hash: %v, chunk hash: %v", encodeErr, task.TaskID, chunk.Hash)
		}
		chunkProofs = append(chunkProofs, proof)

		chunkInfo := message.ChunkInfo{
			ChainID:          bp.cfg.L2.ChainID,
			PrevStateRoot:    common.HexToHash(chunk.ParentChunkStateRoot),
			PostStateRoot:    common.HexToHash(chunk.StateRoot),
			WithdrawRoot:     common.HexToHash(chunk.WithdrawRoot),
			DataHash:         common.HexToHash(chunk.Hash),
			PrevMsgQueueHash: common.HexToHash(chunk.PrevL1MessageQueueHash),
			PostMsgQueueHash: common.HexToHash(chunk.PostL1MessageQueueHash),
			IsPadding:        false,
		}
		if halo2Proof, ok := proof.(*message.Halo2ChunkProof); ok {
			if halo2Proof.ChunkInfo != nil {
				chunkInfo.TxBytes = halo2Proof.ChunkInfo.TxBytes
			}
		}
		if openvmProof, ok := proof.(*message.OpenVMChunkProof); ok {
			chunkInfo.InitialBlockNumber = openvmProof.MetaData.ChunkInfo.InitialBlockNumber
			chunkInfo.BlockCtxs = openvmProof.MetaData.ChunkInfo.BlockCtxs
			chunkInfo.TxDataLength = openvmProof.MetaData.ChunkInfo.TxDataLength
		}
		chunkInfos = append(chunkInfos, &chunkInfo)
	}

	taskDetail, err := bp.getBatchTaskDetail(batch, chunkInfos, chunkProofs, hardForkName)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch task detail, taskID:%s err:%w", task.TaskID, err)
	}

	chunkProofsBytes, err := json.Marshal(taskDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chunk proofs, taskID:%s err:%w", task.TaskID, err)
	}

	taskMsg := &coordinatorType.GetTaskSchema{
		UUID:         task.UUID.String(),
		TaskID:       task.TaskID,
		TaskType:     int(message.ProofTypeBatch),
		TaskData:     string(chunkProofsBytes),
		HardForkName: hardForkName,
	}

	log.Debug("TaskData", "task_id", task.TaskID, "task_type", message.ProofTypeBatch.String(), "hard_fork_name", hardForkName, "task_data", taskMsg.TaskData)

	return taskMsg, nil
}

func (bp *BatchProverTask) recoverActiveAttempts(ctx *gin.Context, batchTask *orm.Batch) {
	if err := bp.batchOrm.DecreaseActiveAttemptsByHash(ctx.Copy(), batchTask.Hash); err != nil {
		log.Error("failed to recover batch active attempts", "hash", batchTask.Hash, "error", err)
	}
}

func (bp *BatchProverTask) getBatchTaskDetail(dbBatch *orm.Batch, chunkInfos []*message.ChunkInfo, chunkProofs []message.ChunkProof, hardForkName string) (*message.BatchTaskDetail, error) {
	taskDetail := &message.BatchTaskDetail{
		ChunkInfos:  chunkInfos,
		ChunkProofs: chunkProofs,
	}

	if hardForkName == message.EuclidV2Fork {
		taskDetail.ForkName = message.EuclidV2ForkNameForProver
	} else if hardForkName == message.EuclidFork {
		taskDetail.ForkName = message.EuclidForkNameForProver
	}

	dbBatchCodecVersion := encoding.CodecVersion(dbBatch.CodecVersion)
	switch dbBatchCodecVersion {
	case encoding.CodecV3, encoding.CodecV4, encoding.CodecV6, encoding.CodecV7:
	default:
		return taskDetail, nil
	}

	codec, err := encoding.CodecFromVersion(encoding.CodecVersion(dbBatch.CodecVersion))
	if err != nil {
		return nil, fmt.Errorf("failed to get codec from version %d, err: %w", dbBatch.CodecVersion, err)
	}

	batchHeader, decodeErr := codec.NewDABatchFromBytes(dbBatch.BatchHeader)
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to decode batch header version %d: %w", dbBatch.CodecVersion, decodeErr)
	}
	taskDetail.BatchHeader = batchHeader
	taskDetail.BlobBytes = dbBatch.BlobBytes
	taskDetail.ChallengeDigest = common.HexToHash(dbBatch.ChallengeDigest)
	// Memory layout of `BlobDataProof`: used in Codec.BlobDataProofForPointEvaluation()
	// | z       | y       | kzg_commitment | kzg_proof |
	// |---------|---------|----------------|-----------|
	// | bytes32 | bytes32 | bytes48        | bytes48   |
	taskDetail.KzgProof = message.Byte48{Big: hexutil.Big(*new(big.Int).SetBytes(dbBatch.BlobDataProof[112:160]))}
	taskDetail.KzgCommitment = message.Byte48{Big: hexutil.Big(*new(big.Int).SetBytes(dbBatch.BlobDataProof[64:112]))}

	return taskDetail, nil
}
