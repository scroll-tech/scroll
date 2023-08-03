package provertask

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// ChunkProverTask the chunk prover task
type ChunkProverTask struct {
	BaseCollector
}

// NewChunkProverTask new a chunk prover task
func NewChunkProverTask(cfg *config.Config, db *gorm.DB) *ChunkProverTask {
	cp := &ChunkProverTask{
		BaseCollector: BaseCollector{
			db:            db,
			cfg:           cfg,
			chunkOrm:      orm.NewChunk(db),
			blockOrm:      orm.NewL2Block(db),
			proverTaskOrm: orm.NewProverTask(db),
		},
	}
	return cp
}

// Collect the chunk proof which need to prove
func (cp *ChunkProverTask) Collect(ctx *gin.Context) (*coordinatorType.GetTaskSchema, error) {
	// load and send chunk tasks
	chunkTasks, err := cp.chunkOrm.GetUnassignedChunks(ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get unassigned chunk proving tasks, error:%w", err)
	}

	if len(chunkTasks) == 0 {
		return nil, nil
	}

	if len(chunkTasks) != 1 {
		return nil, fmt.Errorf("get unassigned chunk proving task len not 1, chunk tasks:%v", chunkTasks)
	}

	chunkTask := chunkTasks[0]

	log.Info("start chunk generation session", "id", chunkTask.Hash)

	if !cp.checkAttemptsExceeded(chunkTask.Hash, message.ProofTypeChunk) {
		return nil, fmt.Errorf("chunk proof hash id:%s check attempts have reach the maximum", chunkTask.Hash)
	}

	publicKey, publicKeyExist := ctx.Get(coordinatorType.PublicKey)
	if !publicKeyExist {
		return nil, fmt.Errorf("get public key from contex failed")
	}

	proverName, proverNameExist := ctx.Get(coordinatorType.ProverName)
	if !proverNameExist {
		return nil, fmt.Errorf("get prover name from contex failed")
	}

	transErr := cp.db.Transaction(func(tx *gorm.DB) error {
		// Update session proving status as assigned.
		if err = cp.chunkOrm.UpdateProvingStatus(ctx, chunkTask.Hash, types.ProvingTaskAssigned, tx); err != nil {
			log.Error("failed to update task status", "id", chunkTask.Hash, "err", err)
			return err
		}

		proverTask := orm.ProverTask{
			TaskID:          chunkTask.Hash,
			ProverPublicKey: publicKey.(string),
			TaskType:        int16(message.ProofTypeChunk),
			ProverName:      proverName.(string),
			ProvingStatus:   int16(types.ProverAssigned),
			FailureType:     int16(types.ProverTaskFailureTypeUndefined),
			// here why need use UTC time. see scroll/common/databased/db.go
			AssignedAt: utils.NowUTC(),
		}
		if err = cp.proverTaskOrm.SetProverTask(ctx, &proverTask, tx); err != nil {
			return fmt.Errorf("db set session info fail, session id:%s , public key:%s, err:%w", chunkTask.Hash, publicKey, err)
		}

		return nil
	})

	if transErr != nil {
		return nil, transErr
	}

	taskMsg, err := cp.formatProverTask(ctx, chunkTask.Hash)
	if err != nil {
		return nil, fmt.Errorf("format prover task failure, id:%s error:%w", chunkTask.Hash, err)
	}

	return taskMsg, nil
}

func (cp *ChunkProverTask) formatProverTask(ctx context.Context, hash string) (*coordinatorType.GetTaskSchema, error) {
	// Get block hashes.
	wrappedBlocks, err := cp.blockOrm.GetL2BlocksByChunkHash(ctx, hash)
	if err != nil || len(wrappedBlocks) == 0 {
		return nil, fmt.Errorf("failed to fetch wrapped blocks, batch hash:%s err:%w", hash, err)
	}

	blockHashes := make([]common.Hash, len(wrappedBlocks))
	for i, wrappedBlock := range wrappedBlocks {
		blockHashes[i] = wrappedBlock.Header.Hash()
	}

	taskDetail := message.ChunkTaskDetail{
		BlockHashes: blockHashes,
	}
	blockHashesBytes, err := json.Marshal(taskDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal block hashes hash:%s, err:%w", hash, err)
	}

	proverTaskSchema := &coordinatorType.GetTaskSchema{
		TaskID:    hash,
		TaskType:  int(message.ProofTypeChunk),
		ProofData: string(blockHashesBytes),
	}

	return proverTaskSchema, nil
}
