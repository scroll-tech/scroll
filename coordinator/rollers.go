package coordinator

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	geth_metrics "github.com/scroll-tech/go-ethereum/metrics"

	"scroll-tech/common/message"
	"scroll-tech/common/metrics"
	"scroll-tech/common/types"
)

type rollerMetrics struct {
	rollerProofsProvingSuccessTimeTimer    geth_metrics.Timer
	rollerProofsProvingFailedTimeTimer     geth_metrics.Timer
	rollerProofsSuccessTotalCounter        geth_metrics.Counter
	rollerProofsFailedTotalCounter         geth_metrics.Counter
	rollerProofsLastAssignedTimestampGauge geth_metrics.Gauge
	rollerProofsLastFinishedTimestampGauge geth_metrics.Gauge
}

// rollerNode records roller status and send task to connected roller.
type rollerNode struct {
	// Roller name
	Name string
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

func (r *rollerNode) sendTask(id string, traces []*geth_types.BlockTrace) bool {
	select {
	case r.taskChan <- &message.TaskMsg{
		ID:     id,
		Traces: traces,
	}:
		r.TaskIDs.Set(id, struct{}{})
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
		for pk, roller := range sess.info.Rollers {
			if pk == pubkey && roller.Status == types.RollerAssigned {
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
			rollerProofsProvingSuccessTimeTimer:    geth_metrics.GetOrRegisterTimer(fmt.Sprintf("roller/proofs/proving/success/time/%s", pubkey), metrics.ScrollRegistry),
			rollerProofsProvingFailedTimeTimer:     geth_metrics.GetOrRegisterTimer(fmt.Sprintf("roller/proofs/proving/failed/time/%s", pubkey), metrics.ScrollRegistry),
			rollerProofsSuccessTotalCounter:        geth_metrics.GetOrRegisterCounter(fmt.Sprintf("roller/proofs/success/total/%s", pubkey), metrics.ScrollRegistry),
			rollerProofsFailedTotalCounter:         geth_metrics.GetOrRegisterCounter(fmt.Sprintf("roller/proofs/failed/total/%s", pubkey), metrics.ScrollRegistry),
			rollerProofsLastAssignedTimestampGauge: geth_metrics.GetOrRegisterGauge(fmt.Sprintf("roller/proofs/last/assigned/timestamp/%s", pubkey), metrics.ScrollRegistry),
			rollerProofsLastFinishedTimestampGauge: geth_metrics.GetOrRegisterGauge(fmt.Sprintf("roller/proofs/last/finished/timestamp/%s", pubkey), metrics.ScrollRegistry),
		}
		node = &rollerNode{
			Name:          identity.Name,
			Version:       identity.Version,
			PublicKey:     pubkey,
			TaskIDs:       *taskIDs,
			taskChan:      make(chan *message.TaskMsg, 4),
			rollerMetrics: rMs,
		}
		m.rollerPool.Set(pubkey, node)
	}
	roller := node.(*rollerNode)
	// avoid reconnection too frequently.
	if time.Since(roller.registerTime) < 60 {
		log.Warn("roller reconnect too frequently", "roller_name", identity.Name, "public key", pubkey)
		return nil, fmt.Errorf("roller reconnect too frequently")
	}
	// update register time and status
	roller.registerTime = time.Now()

	return roller.taskChan, nil
}

func (m *Manager) freeRoller(pk string) {
	m.rollerPool.Pop(pk)
}

func (m *Manager) updateRollerProofsSuccessTotal(pk string) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsSuccessTotalCounter.Inc(1)
		} else {
			log.Error("successTotal metric is nil", "roller pk", pk)
		}
	}
}

func (m *Manager) updateRollerProofsFailedTotal(pk string) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsFailedTotalCounter.Inc(1)
		} else {
			log.Error("failedTotal metric is nil", "roller pk", pk)
		}
	}
}

func (m *Manager) updateRollerProofsLastFinishedTimestamp(pk string) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsLastFinishedTimestampGauge.Update(time.Now().Unix())
		} else {
			log.Error("lastFinishedTimestamp metric is nil", "roller pk", pk)
		}
	}
}

func (m *Manager) updateRollerProofsLastAssignedTimestamp(pk string) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsLastAssignedTimestampGauge.Update(time.Now().Unix())
		} else {
			log.Error("lastAssignedTimestamp metric is nil", "roller pk", pk)
		}
	}
}

func (m *Manager) updateRollerProvingSuccessTimeTimer(pk string, d time.Duration) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsProvingSuccessTimeTimer.Update(d)
		} else {
			log.Error("provingSuccessTime metric is nil", "roller pk", pk)
		}
	}
}

func (m *Manager) updateRollerProvingFailedTimeTimer(pk string, d time.Duration) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsProvingFailedTimeTimer.Update(d)
		} else {
			log.Error("provingFailedTime metric is nil", "roller pk", pk)
		}
	}
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
func (m *Manager) GetNumberOfIdleRollers() (count int) {
	for i, pk := range m.rollerPool.Keys() {
		if val, ok := m.rollerPool.Get(pk); ok {
			r := val.(*rollerNode)
			if r.TaskIDs.Count() == 0 {
				count++
			}
		} else {
			log.Error("rollerPool Get fail", "pk", pk, "idx", i, "pk len", pk)
		}
	}
	return count
}

func (m *Manager) selectRoller() *rollerNode {
	pubkeys := m.rollerPool.Keys()
	for len(pubkeys) > 0 {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(pubkeys))))
		if val, ok := m.rollerPool.Get(pubkeys[idx.Int64()]); ok {
			r := val.(*rollerNode)
			if r.TaskIDs.Count() == 0 {
				return r
			}
		} else {
			log.Error("rollerPool Get fail", "pk", pubkeys[idx.Int64()], "idx", idx.Int64(), "pk len", len(pubkeys))
		}
		pubkeys[idx.Int64()], pubkeys = pubkeys[0], pubkeys[1:]
	}
	return nil
}
