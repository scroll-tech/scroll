package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/rollermanager"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// AggTaskCollector agg task collector is collector implement
type AggTaskCollector struct {
	BaseCollector
}

// NewAggTaskCollector new a AggTaskCollector
func NewAggTaskCollector(cfg *config.Config, db *gorm.DB) *AggTaskCollector {
	atc := &AggTaskCollector{
		BaseCollector: BaseCollector{
			cache:          cache.New(10*time.Minute, time.Hour),
			cfg:            cfg,
			aggTaskOrm:     orm.NewAggTask(db),
			sessionInfoOrm: orm.NewSessionInfo(db),
		},
	}
	return atc
}

// Type return the AggTaskCollector name
func (atc *AggTaskCollector) Type() message.ProveType {
	return message.AggregatorProve
}

// Collect the agg task which need to prove
func (atc *AggTaskCollector) Collect(ctx context.Context) error {
	whereField := map[string]interface{}{"proving_status": types.ProvingTaskUnassigned}
	orderByList := []string{"id ASC"}
	aggTasks, err := atc.aggTaskOrm.GetAggTasks(whereField, orderByList, 1)
	if err != nil {
		log.Error("failed to get unassigned aggregator proving tasks", "error", err)
		return err
	}

	if len(aggTasks) == 0 {
		return nil
	}

	if len(aggTasks) != 1 {
		log.Error("get unassigned agg proving task len not 1")
		return err
	}

	aggTask := aggTasks[0]
	log.Info("start aggregator proof generation session", "id", aggTask.ID)

	if !atc.checkAttempts(aggTask.ID) {
		return fmt.Errorf("the agg task idid:%s check attempts error", aggTask.ID)
	}

	if rollermanager.Manager.GetNumberOfIdleRollers(message.AggregatorProve) == 0 {
		err = fmt.Errorf("no idle agg task roller when starting proof generation session, id:%s", aggTask.ID)
		log.Error(err.Error())
		return err
	}

	rollers, err := atc.sendTask(aggTask.ID)
	if err != nil {
		log.Error("send task error, id", aggTask.ID)
		return err
	}

	// Update session proving status as assigned.
	if err = atc.aggTaskOrm.UpdateAggTaskStatus(aggTask.ID, types.ProvingTaskAssigned); err != nil {
		log.Error("failed to update task status", "id", aggTask.ID, "err", err)
		return err
	}

	// Create a proof generation session.
	info := &coordinatorType.RollersInfo{
		ID:             aggTask.ID,
		Rollers:        rollers,
		ProveType:      message.AggregatorProve,
		StartTimestamp: time.Now().Unix(),
	}

	for _, roller := range info.Rollers {
		log.Info("assigned proof to roller", "session id", info.ID, "session type", info.ProveType,
			"roller name", roller.Name, "roller pk", roller.PublicKey, "proof status", roller.Status)
	}

	// Store session info.
	if err = atc.sessionInfoOrm.InsertSessionInfo(info); err != nil {
		log.Error("db set session info fail", "session id", info.ID, "error", err)
		return err
	}
	return nil
}

func (atc *AggTaskCollector) sendTask(taskID string) (map[string]*coordinatorType.RollerStatus, error) {
	subProofBytes, err := atc.aggTaskOrm.GetSubProofsByAggTaskID(taskID)
	if err != nil {
		log.Error("failed to get sub proofs for aggregator task", "id", taskID, "error", err)
		return nil, err
	}

	var subProofs []*message.AggProof
	for _, subProofByte := range subProofBytes {
		var proof message.AggProof
		if err := json.Unmarshal(subProofByte, &proof); err != nil {
			return nil, err
		}
		subProofs = append(subProofs, &proof)
	}
	return atc.BaseCollector.sendTask(message.AggregatorProve, taskID, nil, subProofs)
}
