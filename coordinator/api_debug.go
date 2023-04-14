package coordinator

import (
	"fmt"
	"time"

	ctypes "scroll-tech/common/types"
	"scroll-tech/coordinator/types"
)

// RollerDebugAPI roller api interface in order go get debug message.
type RollerDebugAPI interface {
	// ListRollers returns all live rollers
	ListRollers() ([]*types.RollerInfo, error)
	// GetSessionInfo returns the session information given the session id.
	GetSessionInfo(sessionID string) (*types.SessionInfo, error)
}

// ListRollers returns all live rollers.
func (m *Manager) ListRollers() ([]*types.RollerInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*types.RollerInfo
	for _, pk := range m.rollerPool.Keys() {
		node, exist := m.rollerPool.Get(pk)
		if !exist {
			continue
		}
		roller := node.(*rollerNode)
		info := &types.RollerInfo{
			Name:      roller.Name,
			Version:   roller.Version,
			PublicKey: pk,
		}
		for id, sess := range m.sessions {
			if _, ok := sess.info.Rollers[pk]; ok {
				info.ActiveSessionStartTime = time.Unix(sess.info.StartTimestamp, 0)
				info.ActiveSession = id
				break
			}
		}
		res = append(res, info)
	}

	return res, nil
}

func newSessionInfo(sess *session, status ctypes.ProvingStatus, errMsg string, finished bool) *types.SessionInfo {
	now := time.Now()
	var nameList []string
	for pk := range sess.info.Rollers {
		nameList = append(nameList, sess.info.Rollers[pk].Name)
	}
	info := types.SessionInfo{
		ID:              sess.info.ID,
		Status:          status.String(),
		AssignedRollers: nameList,
		StartTime:       time.Unix(sess.info.StartTimestamp, 0),
		Error:           errMsg,
	}
	if finished {
		info.FinishTime = now
	}
	return &info
}

// GetSessionInfo returns the session information given the session id.
func (m *Manager) GetSessionInfo(sessionID string) (*types.SessionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if info, ok := m.failedSessionInfos[sessionID]; ok {
		return info, nil
	}
	if s, ok := m.sessions[sessionID]; ok {
		return newSessionInfo(s, ctypes.ProvingTaskAssigned, "", false), nil
	}
	return nil, fmt.Errorf("no such session, sessionID: %s", sessionID)
}
