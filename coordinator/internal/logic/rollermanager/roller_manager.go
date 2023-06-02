package rollermanager

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	coordinatorType "scroll-tech/coordinator/internal/types"
)

var (
	once sync.Once
	// Manager the global roller manager
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

// InitRollerManager init a roller manager
func InitRollerManager() {
	once.Do(func() {
		Manager = &rollerManager{
			rollerPool: cmap.New(),
		}
	})
}

// Register the identity message to roller manager with the public key
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

// SendTask send the need proved message to roller
func (r *rollerManager) SendTask(rollerType message.ProveType, msg *message.TaskMsg) (string, string, error) {
	tmpRoller := r.selectRoller(rollerType)
	if tmpRoller == nil {
		return "", "", errors.New("selectRoller returns nil")
	}

	select {
	case tmpRoller.taskChan <- msg:
	default:
		err := fmt.Errorf("roller channel is full, rollerName:%s, publicKey:%s", tmpRoller.Name, tmpRoller.PublicKey)
		return "", "", err
	}

	r.UpdateMetricRollerProofsLastAssignedTimestampGauge(tmpRoller.PublicKey)

	return tmpRoller.PublicKey, tmpRoller.Name, nil
}

// AddRollerInfo add a rollers info to the roller manager
func (r *rollerManager) AddRollerInfo(rollersInfo *coordinatorType.RollersInfo) bool {
	for pk, roller := range rollersInfo.Rollers {
		taskIds, exist := r.rollerPool.Get(pk)
		if !exist {
			log.Warn("pk", pk, "is not exist, add roller info failure")
			return false
		}

		tmpRollerNode, ok := taskIds.(*rollerNode)
		if tmpRollerNode == nil || !ok {
			log.Warn("pk", pk, "get roller node failure")
			return false
		}

		if roller.Status == types.RollerAssigned {
			tmpRollerNode.TaskIDs.Set(rollersInfo.ID, rollersInfo)
		}
	}
	return true
}

// RollersInfo get a rollers info by pk, id
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

	rollersInfo, isRollersInfo := data.(*coordinatorType.RollersInfo)
	if !isRollersInfo {
		return nil, false
	}
	return rollersInfo, true
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
