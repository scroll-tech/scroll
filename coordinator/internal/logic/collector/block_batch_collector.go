package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
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

// BlockBatchCollector the block batch collector
type BlockBatchCollector struct {
	BaseCollector
}

// NewBlockBatchCollector new a BlockBatch collector
func NewBlockBatchCollector(cfg *config.Config, db *gorm.DB) *BlockBatchCollector {
	bbc := &BlockBatchCollector{
		BaseCollector: BaseCollector{
			cache:          cache.New(10*time.Minute, time.Hour),
			cfg:            cfg,
			blockBatchOrm:  orm.NewBlockBatch(db),
			blockTraceOrm:  orm.NewBlockTrace(db),
			sessionInfoOrm: orm.NewSessionInfo(db),
		},
	}
	return bbc
}

// Type return a block batch collector name
func (bbc *BlockBatchCollector) Type() message.ProveType {
	return message.BasicProve
}

// Collect the block batch which need to prove
func (bbc *BlockBatchCollector) Collect(ctx context.Context) error {
	if bbc.IsPaused() {
		return nil
	}
	whereField := map[string]interface{}{"proving_status": types.ProvingTaskUnassigned}
	orderByList := []string{"index ASC"}
	blockBatches, err := bbc.blockBatchOrm.GetBlockBatches(whereField, orderByList, 1)
	if err != nil {
		log.Error("failed to unassigned basic proving tasks err:", err)
		return err
	}

	if len(blockBatches) == 0 {
		return nil
	}

	if len(blockBatches) != 1 {
		log.Error("get unassigned basic proving task len not 1")
		return err
	}

	blockBatch := blockBatches[0]
	log.Info("start basic proof generation session", "id", blockBatch.Hash)

	if !bbc.checkAttempts(blockBatch.Hash) {
		return fmt.Errorf("the session id:%s check attempts error", blockBatch.Hash)
	}

	if rollermanager.Manager.GetNumberOfIdleRollers(message.BasicProve) == 0 {
		err = fmt.Errorf("no idle basic roller when starting proof generation session, id:%s", blockBatch.Hash)
		log.Error(err.Error())
		return err
	}

	rollers, err := bbc.sendTask(blockBatch.Hash)
	if err != nil {
		log.Error("send task error, id", blockBatch.Hash)
		return err
	}

	// Update session proving status as assigned.
	if err = bbc.blockBatchOrm.UpdateProvingStatus(blockBatch.Hash, types.ProvingTaskAssigned); err != nil {
		log.Error("failed to update task status", "id", blockBatch.Hash, "err", err)
		return err
	}

	// Create a proof generation session.
	info := &coordinatorType.RollersInfo{
		ID:             blockBatch.Hash,
		Rollers:        rollers,
		ProveType:      message.BasicProve,
		StartTimestamp: time.Now().Unix(),
	}

	for _, roller := range info.Rollers {
		log.Info("assigned proof to roller", "session id", info.ID, "session type", info.ProveType,
			"roller name", roller.Name, "roller pk", roller.PublicKey, "proof status", roller.Status)
	}

	// Store session info.
	if err = bbc.sessionInfoOrm.InsertSessionInfo(info); err != nil {
		log.Error("db set session info fail", "session id", info.ID, "error", err)
		return err
	}
	return nil
}

func (bbc *BlockBatchCollector) sendTask(hash string) (map[string]*coordinatorType.RollerStatus, error) {
	blockTraceInfos, err := bbc.blockTraceOrm.GetL2BlockInfos(map[string]interface{}{"batch_hash": hash}, nil, 0)
	if err != nil {
		log.Error("could not GetBlockInfos batch_hash:%s err:%v", hash, err)
		return nil, err
	}

	var traces []common.Hash
	for _, blockTraceInfo := range blockTraceInfos {
		traces = append(traces, common.HexToHash(blockTraceInfo.Hash))
	}

	return bbc.BaseCollector.sendTask(message.BasicProve, hash, traces, nil)
}
