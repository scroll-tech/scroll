package coordinator

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/scroll-tech/go-ethereum/log"
	geth_metrics "github.com/scroll-tech/go-ethereum/metrics"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
)

// rollerNode records roller status and send task to connected roller.
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

	metrics *rollerMetrics
}

func (r *rollerNode) sendTask(msg *message.TaskMsg) bool {
	select {
	case r.taskChan <- msg:
		r.TaskIDs.Set(msg.ID, struct{}{})
	default:
		log.Warn("roller channel is full", "roller name", r.Name, "public key", r.PublicKey)
		return false
	}
	return true
}

func (m *Manager) reloadRollerAssignedTasks(pubkey string) *cmap.ConcurrentMap {
	m.mu.RLock()
	defer m.mu.RUnlock()
	taskIDs := cmap.New()
	for id, sess := range m.sessions {
		for _, sessionInfo := range sess.sessionInfos {
			if sessionInfo.RollerPublicKey == pubkey && sessionInfo.ProvingStatus == int16(types.RollerAssigned) {
				taskIDs.Set(id, struct{}{})
			}
		}
	}
	return &taskIDs
}

func (m *Manager) register(pubkey string, identity *message.Identity) (<-chan *message.TaskMsg, error) {
	node, ok := m.rollerPool.Get(pubkey)
	if !ok {
		taskIDs := m.reloadRollerAssignedTasks(pubkey)
		rMs := &rollerMetrics{
			rollerProofsVerifiedSuccessTimeTimer:   geth_metrics.GetOrRegisterTimer(fmt.Sprintf("roller/proofs/verified/success/time/%s", pubkey), metrics.ScrollRegistry),
			rollerProofsVerifiedFailedTimeTimer:    geth_metrics.GetOrRegisterTimer(fmt.Sprintf("roller/proofs/verified/failed/time/%s", pubkey), metrics.ScrollRegistry),
			rollerProofsGeneratedFailedTimeTimer:   geth_metrics.GetOrRegisterTimer(fmt.Sprintf("roller/proofs/generated/failed/time/%s", pubkey), metrics.ScrollRegistry),
			rollerProofsLastAssignedTimestampGauge: geth_metrics.GetOrRegisterGauge(fmt.Sprintf("roller/proofs/last/assigned/timestamp/%s", pubkey), metrics.ScrollRegistry),
			rollerProofsLastFinishedTimestampGauge: geth_metrics.GetOrRegisterGauge(fmt.Sprintf("roller/proofs/last/finished/timestamp/%s", pubkey), metrics.ScrollRegistry),
		}
		node = &rollerNode{
			Name:      identity.Name,
			Type:      identity.RollerType,
			Version:   identity.Version,
			PublicKey: pubkey,
			TaskIDs:   *taskIDs,
			taskChan:  make(chan *message.TaskMsg, 4),
			metrics:   rMs,
		}
		m.rollerPool.Set(pubkey, node)
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

func (m *Manager) freeRoller(pk string) {
	m.rollerPool.Pop(pk)
}

func (m *Manager) existTaskIDForRoller(pk string, id string) bool {
	if node, ok := m.rollerPool.Get(pk); ok {
		r := node.(*rollerNode)
		return r.TaskIDs.Has(id)
	}
	return false
}

func (m *Manager) freeTaskIDForRoller(pk string, id string) {
	if node, ok := m.rollerPool.Get(pk); ok {
		r := node.(*rollerNode)
		r.TaskIDs.Pop(id)
	}
}

// GetNumberOfIdleRollers return the count of idle rollers.
func (m *Manager) GetNumberOfIdleRollers(rollerType message.ProveType) (count int) {
	for _, pk := range m.rollerPool.Keys() {
		if val, ok := m.rollerPool.Get(pk); ok {
			r := val.(*rollerNode)
			if r.TaskIDs.Count() == 0 && r.Type == rollerType {
				count++
			}
		}
	}
	return count
}

func (m *Manager) selectRoller(rollerType message.ProveType) *rollerNode {
	pubkeys := m.rollerPool.Keys()
	for len(pubkeys) > 0 {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(pubkeys))))
		if val, ok := m.rollerPool.Get(pubkeys[idx.Int64()]); ok {
			r := val.(*rollerNode)
			if r.TaskIDs.Count() == 0 && r.Type == rollerType {
				return r
			}
		}
		pubkeys[idx.Int64()], pubkeys = pubkeys[0], pubkeys[1:]
	}
	return nil
}
