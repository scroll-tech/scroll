package rollermanager

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
	// Manager the global roller manager
	Manager *rollerManager
)

// RollerNode is the interface that controls the rollers
type rollerNode struct {
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
	// session id list which delivered to roller.
	TaskIDs cmap.ConcurrentMap

	// Time of message creation
	registerTime time.Time

	metrics *rollerMetrics
}

type rollerManager struct {
	rollerPool    cmap.ConcurrentMap
	proverTaskOrm *orm.ProverTask
}

// InitRollerManager init a roller manager
func InitRollerManager(db *gorm.DB) {
	once.Do(func() {
		Manager = &rollerManager{
			rollerPool:    cmap.New(),
			proverTaskOrm: orm.NewProverTask(db),
		}
	})
}

// Register the identity message to roller manager with the public key
func (r *rollerManager) Register(ctx context.Context, proverPublicKey string, identity *message.Identity) (<-chan *message.TaskMsg, error) {
	node, ok := r.rollerPool.Get(proverPublicKey)
	if !ok {
		taskIDs, err := r.reloadRollerAssignedTasks(ctx, proverPublicKey)
		if err != nil {
			return nil, fmt.Errorf("register error:%w", err)
		}

		rMs := &rollerMetrics{
			rollerProofsVerifiedSuccessTimeTimer:   gethMetrics.GetOrRegisterTimer(fmt.Sprintf("roller/proofs/verified/success/time/%s", proverPublicKey), metrics.ScrollRegistry),
			rollerProofsVerifiedFailedTimeTimer:    gethMetrics.GetOrRegisterTimer(fmt.Sprintf("roller/proofs/verified/failed/time/%s", proverPublicKey), metrics.ScrollRegistry),
			rollerProofsGeneratedFailedTimeTimer:   gethMetrics.GetOrRegisterTimer(fmt.Sprintf("roller/proofs/generated/failed/time/%s", proverPublicKey), metrics.ScrollRegistry),
			rollerProofsLastAssignedTimestampGauge: gethMetrics.GetOrRegisterGauge(fmt.Sprintf("roller/proofs/last/assigned/timestamp/%s", proverPublicKey), metrics.ScrollRegistry),
			rollerProofsLastFinishedTimestampGauge: gethMetrics.GetOrRegisterGauge(fmt.Sprintf("roller/proofs/last/finished/timestamp/%s", proverPublicKey), metrics.ScrollRegistry),
		}
		node = &rollerNode{
			Name:      identity.Name,
			Type:      identity.RollerType,
			Version:   identity.Version,
			PublicKey: proverPublicKey,
			TaskIDs:   *taskIDs,
			taskChan:  make(chan *message.TaskMsg, 4),
			metrics:   rMs,
		}
		r.rollerPool.Set(proverPublicKey, node)
	}
	roller := node.(*rollerNode)
	// avoid reconnection too frequently.
	if time.Since(roller.registerTime) < 60 {
		log.Warn("roller reconnect too frequently", "prover_name", identity.Name, "roller_type", identity.RollerType, "public key", proverPublicKey)
		return nil, fmt.Errorf("roller reconnect too frequently")
	}
	// update register time and status
	roller.registerTime = time.Now()

	return roller.taskChan, nil
}

func (r *rollerManager) reloadRollerAssignedTasks(ctx context.Context, proverPublicKey string) (*cmap.ConcurrentMap, error) {
	var assignedProverTasks []orm.ProverTask
	for {
		limit := 100
		batchAssignedProverTasks, err := r.proverTaskOrm.GetAssignedProverTasks(ctx, limit)
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

// SendTask send the need proved message to roller
func (r *rollerManager) SendTask(rollerType message.ProofType, msg *message.TaskMsg) (string, string, error) {
	tmpRoller := r.selectRoller(rollerType)
	if tmpRoller == nil {
		return "", "", errors.New("selectRoller returns nil")
	}

	select {
	case tmpRoller.taskChan <- msg:
		tmpRoller.TaskIDs.Set(msg.ID, struct{}{})
	default:
		err := fmt.Errorf("roller channel is full, rollerName:%s, publicKey:%s", tmpRoller.Name, tmpRoller.PublicKey)
		return "", "", err
	}

	r.UpdateMetricRollerProofsLastAssignedTimestampGauge(tmpRoller.PublicKey)

	return tmpRoller.PublicKey, tmpRoller.Name, nil
}

// ExistTaskIDForRoller check the task exist
func (r *rollerManager) ExistTaskIDForRoller(pk string, id string) bool {
	node, ok := r.rollerPool.Get(pk)
	if !ok {
		return false
	}
	roller := node.(*rollerNode)
	return roller.TaskIDs.Has(id)
}

// FreeRoller free the roller with the pk key
func (r *rollerManager) FreeRoller(pk string) {
	r.rollerPool.Pop(pk)
}

// FreeTaskIDForRoller free a task of the pk roller
func (r *rollerManager) FreeTaskIDForRoller(pk string, id string) {
	if node, ok := r.rollerPool.Get(pk); ok {
		roller := node.(*rollerNode)
		roller.TaskIDs.Pop(id)
	}
}

// GetNumberOfIdleRollers return the count of idle rollers.
func (r *rollerManager) GetNumberOfIdleRollers(rollerType message.ProofType) (count int) {
	for item := range r.rollerPool.IterBuffered() {
		roller := item.Val.(*rollerNode)
		if roller.TaskIDs.Count() == 0 && roller.Type == rollerType {
			count++
		}
	}
	return count
}

func (r *rollerManager) selectRoller(rollerType message.ProofType) *rollerNode {
	pubkeys := r.rollerPool.Keys()
	for len(pubkeys) > 0 {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(pubkeys))))
		if val, ok := r.rollerPool.Get(pubkeys[idx.Int64()]); ok {
			rn := val.(*rollerNode)
			if rn.TaskIDs.Count() == 0 && rn.Type == rollerType {
				return rn
			}
		}
		pubkeys[idx.Int64()], pubkeys = pubkeys[0], pubkeys[1:]
	}
	return nil
}
