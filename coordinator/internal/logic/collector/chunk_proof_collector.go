package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/rollermanager"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// ChunkProofCollector the chunk proof collector
type ChunkProofCollector struct {
	db *gorm.DB

	BaseCollector
}

// NewChunkProofCollector new a chunk proof collector
func NewChunkProofCollector(cfg *config.Config, db *gorm.DB) *ChunkProofCollector {
	cp := &ChunkProofCollector{
		db: db,
		BaseCollector: BaseCollector{
			cfg:           cfg,
			chunkOrm:      orm.NewChunk(db),
			blockOrm:      orm.NewL2Block(db),
			proverTaskOrm: orm.NewProverTask(db),
		},
	}
	return cp
}

// Name return a block batch collector name
func (cp *ChunkProofCollector) Name() string {
	return ChunkCollectorName
}

// Collect the chunk proof which need to prove
func (cp *ChunkProofCollector) Collect(ctx context.Context) error {
	// load and send chunk tasks
	chunkTasks, err := cp.chunkOrm.GetUnassignedChunks(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to get unassigned chunk proving tasks, error:%w", err)
	}

	if len(chunkTasks) == 0 {
		return nil
	}

	if len(chunkTasks) != 1 {
		return fmt.Errorf("get unassigned chunk proving task len not 1")
	}

	chunkTask := chunkTasks[0]

	log.Info("start chunk generation session", "id", chunkTask.Hash)

	if !cp.checkAttemptsExceeded(chunkTask.Hash) {
		return fmt.Errorf("the session id:%s check attempts have reach the maximum", chunkTask.Hash)
	}

	if rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeChunk) == 0 {
		return fmt.Errorf("no idle chunk roller when starting proof generation session, id:%s", chunkTask.Hash)
	}

	rollerStatusList, err := cp.sendTask(ctx, chunkTask.Hash)
	if err != nil {
		return fmt.Errorf("send task failure, id:%s error:%w", chunkTask.Hash, err)
	}

	transErr := cp.db.Transaction(func(tx *gorm.DB) error {
		// Update session proving status as assigned.
		if err = cp.chunkOrm.UpdateProvingStatus(ctx, chunkTask.Hash, types.ProvingTaskAssigned, tx); err != nil {
			log.Error("failed to update task status", "id", chunkTask.Hash, "err", err)
			return err
		}

		for _, rollerStatus := range rollerStatusList {
			proverTask := orm.ProverTask{
				TaskID:          chunkTask.Hash,
				ProverPublicKey: rollerStatus.PublicKey,
				TaskType:        int16(message.ProofTypeChunk),
				ProverName:      rollerStatus.Name,
				ProvingStatus:   int16(types.RollerAssigned),
				FailureType:     int16(types.RollerFailureTypeUndefined),
				CreatedAt:       time.Now(), // Used in proverTasks, should be explicitly assigned here.
			}
			if err = cp.proverTaskOrm.SetProverTask(ctx, &proverTask, tx); err != nil {
				return fmt.Errorf("db set session info fail, session id:%s , public key:%s, err:%w", chunkTask.Hash, rollerStatus.PublicKey, err)
			}
		}
		return nil
	})
	return transErr
}

func (cp *ChunkProofCollector) sendTask(ctx context.Context, hash string) ([]*coordinatorType.RollerStatus, error) {
	// Get block hashes.
	wrappedBlocks, err := cp.blockOrm.GetL2BlocksByChunkHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch wrapped blocks, batch hash:%s err:%w", hash, err)
	}
	blockHashes := make([]common.Hash, len(wrappedBlocks))
	for i, wrappedBlock := range wrappedBlocks {
		blockHashes[i] = wrappedBlock.Header.Hash()
	}

	return cp.BaseCollector.sendTask(message.ProofTypeChunk, hash, blockHashes, nil)
}
