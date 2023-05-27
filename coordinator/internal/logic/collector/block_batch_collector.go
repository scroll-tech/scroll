package collector

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// BlockBatchCollector the block batch collector
type BlockBatchCollector struct {
	blockBatchOrm  *orm.BlockBatch
	blockTraceOrm  *orm.BlockTrace
	sessionInfoOrm *orm.SessionInfo
	l2gethClient   *ethclient.Client

	cfg   *config.RollerManagerConfig
	cache *cache.Cache
}

func NewBlockBatchCollector(ctx context.Context, cfg *config.Config, db *gorm.DB, l2gethClient *ethclient.Client) *BlockBatchCollector {
	bbc := &BlockBatchCollector{
		cfg:           cfg,
		blockBatchOrm: orm.NewBlockBatch(db),
		blockTraceOrm: orm.NewBlockTrace(db),
		l2gethClient:  l2gethClient,
		cache:         cache.New(10*time.Minute, time.Hour),
	}

	go bbc.Recover()

	return bbc
}

func (bbc *BlockBatchCollector) Name() string {
	return BlockBatchCollectorName
}

func (bbc *BlockBatchCollector) Recover() {
	defer func() {
		if err := recover(); err != nil {
			nerr := fmt.Errorf("blockBatchCollector Recover panic err:%w", err)
			log.Warn(nerr.Error())
		}
	}()

	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			// deal the times reach
		}
	}
}

func (bbc *BlockBatchCollector) Collect(ctx context.Context) error {
	whereField := map[string]interface{}{"proving_status": types.ProvingTaskUnassigned}
	orderByList := []string{"index ASC"}
	blockBatches, err := bbc.blockBatchOrm.GetBlockBatches(whereField, orderByList, 1)
	if err != nil {
		err = fmt.Errorf("failed to unassigned basic proving tasks err:%w", err)
		log.Error(err.Error())
		return err
	}

	if len(blockBatches) != 1 {
		err = errors.New("get unassigned basic proving task len not 1")
		log.Error(err.Error())
		return err
	}
	blockBatch := blockBatches[0]
	log.Info("start basic proof generation session", "id", blockBatch.Hash)

	if roller.TaskManager.GetNumberOfIdleRollers(message.BasicProve) == 0 {
		err = fmt.Errorf("no idle basic roller when starting proof generation session")
		log.Error(err.Error())
		return err
	}

	rollers, err := bbc.sendTask(ctx, blockBatch.Hash)
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
		Attempts:       1,
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

func (bbc *BlockBatchCollector) sendTask(ctx context.Context, hash string) (map[string]*coordinatorType.RollerStatus, error) {
	blockTraceInfos, err := bbc.blockTraceOrm.GetL2BlockInfos(map[string]interface{}{"batch_hash": hash}, nil, 0)
	if err != nil {
		log.Error("could not GetBlockInfos batch_hash:%s err:%v", hash, err)
		return nil, err
	}

	var traces []*gethTypes.BlockTrace
	for _, blockTraceInfo := range blockTraceInfos {
		tmpTrace, err := bbc.l2gethClient.GetBlockTraceByHash(ctx, common.HexToHash(blockTraceInfo.Hash))
		if err != nil {
			log.Error("could not GetBlockTraceByNumber", "block number", blockTraceInfo.Number, "block hash", blockTraceInfo.Hash, "error", err)
			return nil, err
		}
		traces = append(traces, tmpTrace)
	}

	// Dispatch task to basic rollers.
	var err1 error
	rollers := make(map[string]*coordinatorType.RollerStatus)
	for i := 0; i < int(bbc.cfg.RollersPerSession); i++ {
		sendMsg := &message.TaskMsg{
			ID:   hash,
			Type: message.BasicProve,
		}

		rollerPubKey, rollerName, sendErr := roller.TaskManager.SendTask(message.BasicProve, sendMsg)
		if err != nil {
			err = sendErr
			continue
		}

		rollerStatus := &coordinatorType.RollerStatus{
			PublicKey: rollerPubKey,
			Name:      rollerName,
			Status:    types.RollerAssigned,
		}

		rollersInfo := &coordinatorType.RollersInfo{
			ID:             hash,
			Rollers:        rollers,
			ProveType:      message.BasicProve,
			StartTimestamp: time.Now().Unix(),
			Attempts:       1,
		}

		roller.TaskManager.AddRollerInfo(rollersInfo)

		rollers[rollerPubKey] = rollerStatus
	}
	return rollers, err1
}
