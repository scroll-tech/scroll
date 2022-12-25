package coordinator

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/message"

	"scroll-tech/database/orm"
)

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
}

func (r *rollerNode) sendTask(id string, traces []*types.BlockTrace) bool {
	select {
	case r.taskChan <- &message.TaskMsg{
		ID:     id,
		Traces: traces,
	}:
		r.TaskIDs.Set(id, struct{}{})
	default:
		log.Warn("roller channel is full", "roller name", r.Name, "public_key", r.PublicKey)
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
			if pk == pubkey && roller.Status == orm.RollerAssigned {
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
		node = &rollerNode{
			Name:      identity.Name,
			Version:   identity.Version,
			PublicKey: pubkey,
			TaskIDs:   *taskIDs,
			taskChan:  make(chan *message.TaskMsg, 4),
		}
		m.rollerPool.Set(pubkey, node)
	}
	roller := node.(*rollerNode)
	// avoid reconnection too frequently.
	if time.Since(roller.registerTime) < 60 {
		return nil, fmt.Errorf("roller reconnect too frequently. roller_name: %v. public_key: %v", identity.Name, pubkey)
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
func (m *Manager) GetNumberOfIdleRollers() int {
	pubkeys := m.rollerPool.Keys()
	for i := 0; i < len(pubkeys); i++ {
		if val, ok := m.rollerPool.Get(pubkeys[i]); ok {
			r := val.(*rollerNode)
			if r.TaskIDs.Count() > 0 {
				pubkeys[i], pubkeys = pubkeys[len(pubkeys)-1], pubkeys[:len(pubkeys)-1]
			}
		}
	}
	return len(pubkeys)
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
		}
		pubkeys[idx.Int64()], pubkeys = pubkeys[0], pubkeys[1:]
	}
	return nil
}
