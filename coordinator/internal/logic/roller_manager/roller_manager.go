package roller_manager

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"
	coordinatorType "scroll-tech/coordinator/internal/types"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
)

var (
	once    sync.Once
	Manager *rollerManager
)

// RollerNode the interface for controller how to use roller.
type rollerNode struct {
	// Roller name
	Name string
	// Roller type
	Type message.ProveType
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

	*rollerMetrics
}

type rollerManager struct {
	rollerPool cmap.ConcurrentMap
}

func InitRollerManager() {
	once.Do(func() {
		Manager = &rollerManager{
			rollerPool: cmap.New(),
		}
	})
}

func (r *rollerManager) Register(pubkey string, identity *message.Identity) (<-chan *message.TaskMsg, error) {
	node, ok := r.rollerPool.Get(pubkey)
	if !ok {
		rMs := &rollerMetrics{
			rollerProofsVerifiedSuccessTimeTimer:   gethMetrics.GetOrRegisterTimer(fmt.Sprintf("roller/proofs/verified/success/time/%s", pubkey), metrics.ScrollRegistry),
			rollerProofsVerifiedFailedTimeTimer:    gethMetrics.GetOrRegisterTimer(fmt.Sprintf("roller/proofs/verified/failed/time/%s", pubkey), metrics.ScrollRegistry),
			rollerProofsGeneratedFailedTimeTimer:   gethMetrics.GetOrRegisterTimer(fmt.Sprintf("roller/proofs/generated/failed/time/%s", pubkey), metrics.ScrollRegistry),
			rollerProofsLastAssignedTimestampGauge: gethMetrics.GetOrRegisterGauge(fmt.Sprintf("roller/proofs/last/assigned/timestamp/%s", pubkey), metrics.ScrollRegistry),
			rollerProofsLastFinishedTimestampGauge: gethMetrics.GetOrRegisterGauge(fmt.Sprintf("roller/proofs/last/finished/timestamp/%s", pubkey), metrics.ScrollRegistry),
		}
		node = &rollerNode{
			Name:          identity.Name,
			Type:          identity.RollerType,
			Version:       identity.Version,
			PublicKey:     pubkey,
			TaskIDs:       cmap.New(),
			taskChan:      make(chan *message.TaskMsg, 4),
			rollerMetrics: rMs,
		}
		r.rollerPool.Set(pubkey, node)
	}
	roller := node.(*rollerNode)
	// avoid reconnection too frequently.
	if time.Since(roller.registerTime) < 60 {
		log.Warn("roller reconnect too frequently", "roller_name", identity.Name, "roller_type", identity.RollerType, "public key", pubkey)
		return nil, fmt.Errorf("roller reconnect too frequently")
	}
	// update register time and status
	roller.registerTime = time.Now()

	return roller.taskChan, nil
}

func (r *rollerManager) SendTask(rollerType message.ProveType, msg *message.TaskMsg) (string, string, error) {
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

func (r *rollerManager) AddRollerInfo(rollersInfo *coordinatorType.RollersInfo) bool {
	for pk, roller := range rollersInfo.Rollers {
		taskIds, exist := r.rollerPool.Get(pk)
		if !exist {
			taskIds = cmap.New()
			r.rollerPool.Set(pk, taskIds)
		}

		m, ok := taskIds.(cmap.ConcurrentMap)
		if !ok {
			return false
		}

		if roller.Status == types.RollerAssigned {
			m.Set(rollersInfo.ID, rollersInfo)
		}
	}
	return true
}

func (r *rollerManager) RollersInfo(pk string, id string) (*coordinatorType.RollersInfo, bool) {
	node, ok := r.rollerPool.Get(pk)
	if !ok {
		return nil, false
	}

	rollerNode := node.(*rollerNode)
	data, existSessionInfo := rollerNode.TaskIDs.Get(id)
	if !existSessionInfo {
		return nil, false
	}

	sessionInfo, isSessionInfo := data.(*coordinatorType.RollersInfo)
	if !isSessionInfo {
		return nil, false
	}
	return sessionInfo, true
}

func (r *rollerManager) ExistTaskIDForRoller(pk string, id string) bool {
	node, ok := r.rollerPool.Get(pk)
	if !ok {
		return false
	}
	roller := node.(*rollerNode)
	return roller.TaskIDs.Has(id)
}

func (r *rollerManager) FreeRoller(pk string) {
	r.rollerPool.Pop(pk)
}

func (r *rollerManager) FreeTaskIDForRoller(pk string, id string) {
	if node, ok := r.rollerPool.Get(pk); ok {
		roller := node.(*rollerNode)
		roller.TaskIDs.Pop(id)
	}
}

// GetNumberOfIdleRollers return the count of idle rollers.
func (r *rollerManager) GetNumberOfIdleRollers(rollerType message.ProveType) (count int) {
	for item := range r.rollerPool.IterBuffered() {
		roller := item.Val.(*rollerNode)
		if roller.TaskIDs.Count() == 0 && roller.Type == rollerType {
			count++
		}
	}
	return count
}

func (r *rollerManager) selectRoller(rollerType message.ProveType) *rollerNode {
	pubKeys := r.rollerPool.Keys()
	for len(pubKeys) > 0 {
		idx := rand.Intn(len(pubKeys))
		if val, ok := r.rollerPool.Get(pubKeys[idx]); ok {
			roller := val.(*rollerNode)
			if roller.TaskIDs.Count() == 0 && roller.Type == rollerType {
				return roller
			}
		}
		// remove index idx
		pubKeys = append(pubKeys, pubKeys[:idx]...)
		pubKeys = append(pubKeys, pubKeys[idx+1:]...)
	}
	return nil
}
