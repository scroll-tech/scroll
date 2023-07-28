package provermanager

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"
	"gorm.io/gorm"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/orm"
)

var (
	once sync.Once
	// Manager the global prover manager
	Manager *proverManager
)

// RollerNode is the interface that controls the provers
type proverNode struct {
	// Roller name
	Name string
	// Roller type
	Type message.ProofType
	// Roller public key
	PublicKey string
	// Roller version
	Version string

	// task channel
	taskChan chan *message.TaskMsg
	// session id list which delivered to prover.
	TaskIDs cmap.ConcurrentMap

	// Time of message creation
	registerTime time.Time

	metrics *proverMetrics
}

type proverManager struct {
	proverPool    cmap.ConcurrentMap
	proverTaskOrm *orm.ProverTask
}

// InitRollerManager init a prover manager
func InitRollerManager(db *gorm.DB) {
	once.Do(func() {
		Manager = &proverManager{
			proverPool:    cmap.New(),
			proverTaskOrm: orm.NewProverTask(db),
		}
	})
}

// Register the identity message to prover manager with the public key
func (r *proverManager) Register(ctx context.Context, proverPublicKey string, identity *message.Identity) (<-chan *message.TaskMsg, error) {
	node, ok := r.proverPool.Get(proverPublicKey)
	if !ok {
		taskIDs, err := r.reloadRollerAssignedTasks(ctx, proverPublicKey)
		if err != nil {
			return nil, fmt.Errorf("register error:%w", err)
		}

		rMs := &proverMetrics{
			proverProofsVerifiedSuccessTimeTimer:   gethMetrics.GetOrRegisterTimer(fmt.Sprintf("prover/proofs/verified/success/time/%s", proverPublicKey), metrics.ScrollRegistry),
			proverProofsVerifiedFailedTimeTimer:    gethMetrics.GetOrRegisterTimer(fmt.Sprintf("prover/proofs/verified/failed/time/%s", proverPublicKey), metrics.ScrollRegistry),
			proverProofsGeneratedFailedTimeTimer:   gethMetrics.GetOrRegisterTimer(fmt.Sprintf("prover/proofs/generated/failed/time/%s", proverPublicKey), metrics.ScrollRegistry),
			proverProofsLastAssignedTimestampGauge: gethMetrics.GetOrRegisterGauge(fmt.Sprintf("prover/proofs/last/assigned/timestamp/%s", proverPublicKey), metrics.ScrollRegistry),
			proverProofsLastFinishedTimestampGauge: gethMetrics.GetOrRegisterGauge(fmt.Sprintf("prover/proofs/last/finished/timestamp/%s", proverPublicKey), metrics.ScrollRegistry),
		}
		node = &proverNode{
			Name:      identity.Name,
			Type:      identity.RollerType,
			Version:   identity.Version,
			PublicKey: proverPublicKey,
			TaskIDs:   *taskIDs,
			taskChan:  make(chan *message.TaskMsg, 4),
			metrics:   rMs,
		}
		r.proverPool.Set(proverPublicKey, node)
	}
	prover := node.(*proverNode)
	// avoid reconnection too frequently.
	if time.Since(prover.registerTime) < 60 {
		log.Warn("prover reconnect too frequently", "prover_name", identity.Name, "prover_type", identity.RollerType, "public key", proverPublicKey)
		return nil, fmt.Errorf("prover reconnect too frequently")
	}
	// update register time and status
	prover.registerTime = time.Now()

	return prover.taskChan, nil
}

func (r *proverManager) reloadRollerAssignedTasks(ctx context.Context, proverPublicKey string) (*cmap.ConcurrentMap, error) {
	var assignedProverTasks []orm.ProverTask
	page := 0
	limit := 100
	for {
		page++
		whereFields := make(map[string]interface{})
		whereFields["proving_status"] = int16(types.RollerAssigned)
		orderBy := []string{"id asc"}
		offset := (page - 1) * limit
		batchAssignedProverTasks, err := r.proverTaskOrm.GetProverTasks(ctx, whereFields, orderBy, offset, limit)
		if err != nil {
			log.Warn("reloadRollerAssignedTasks get all assigned failure", "error", err)
			return nil, fmt.Errorf("reloadRollerAssignedTasks error:%w", err)
		}
		if len(batchAssignedProverTasks) < limit {
			break
		}
		assignedProverTasks = append(assignedProverTasks, batchAssignedProverTasks...)
	}

	taskIDs := cmap.New()
	for _, assignedProverTask := range assignedProverTasks {
		if assignedProverTask.ProverPublicKey == proverPublicKey && assignedProverTask.ProvingStatus == int16(types.RollerAssigned) {
			taskIDs.Set(assignedProverTask.TaskID, struct{}{})
		}
	}
	return &taskIDs, nil
}

// SendTask send the need proved message to prover
func (r *proverManager) SendTask(proverType message.ProofType, msg *message.TaskMsg) (string, string, error) {
	tmpRoller := r.selectRoller(proverType)
	if tmpRoller == nil {
		return "", "", errors.New("selectRoller returns nil")
	}

	select {
	case tmpRoller.taskChan <- msg:
		tmpRoller.TaskIDs.Set(msg.ID, struct{}{})
	default:
		err := fmt.Errorf("prover channel is full, proverName:%s, publicKey:%s", tmpRoller.Name, tmpRoller.PublicKey)
		return "", "", err
	}

	r.UpdateMetricRollerProofsLastAssignedTimestampGauge(tmpRoller.PublicKey)

	return tmpRoller.PublicKey, tmpRoller.Name, nil
}

// ExistTaskIDForRoller check the task exist
func (r *proverManager) ExistTaskIDForRoller(pk string, id string) bool {
	node, ok := r.proverPool.Get(pk)
	if !ok {
		return false
	}
	prover := node.(*proverNode)
	return prover.TaskIDs.Has(id)
}

// FreeRoller free the prover with the pk key
func (r *proverManager) FreeRoller(pk string) {
	r.proverPool.Pop(pk)
}

// FreeTaskIDForRoller free a task of the pk prover
func (r *proverManager) FreeTaskIDForRoller(pk string, id string) {
	if node, ok := r.proverPool.Get(pk); ok {
		prover := node.(*proverNode)
		prover.TaskIDs.Pop(id)
	}
}

// GetNumberOfIdleRollers return the count of idle provers.
func (r *proverManager) GetNumberOfIdleRollers(proverType message.ProofType) (count int) {
	for item := range r.proverPool.IterBuffered() {
		prover := item.Val.(*proverNode)
		if prover.TaskIDs.Count() == 0 && prover.Type == proverType {
			count++
		}
	}
	return count
}

func (r *proverManager) selectRoller(proverType message.ProofType) *proverNode {
	pubkeys := r.proverPool.Keys()
	for len(pubkeys) > 0 {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(pubkeys))))
		if val, ok := r.proverPool.Get(pubkeys[idx.Int64()]); ok {
			rn := val.(*proverNode)
			if rn.TaskIDs.Count() == 0 && rn.Type == proverType {
				return rn
			}
		}
		pubkeys[idx.Int64()], pubkeys = pubkeys[0], pubkeys[1:]
	}
	return nil
}
