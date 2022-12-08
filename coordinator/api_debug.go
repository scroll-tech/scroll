package coordinator

import (
	"fmt"
	"time"

	"scroll-tech/database/orm"
)

// RollerDebugAPI roller api interface in order go get debug message.
type RollerDebugAPI interface {
	// ListRollers returns all live rollers
	ListRollers() ([]*RollerInfo, error)
	// GetSessionInfo returns the session information given the session id.
	GetSessionInfo(sessionID string) (*SessionInfo, error)
}

// RollerInfo records the roller name, pub key and active session info (id, start time).
type RollerInfo struct {
	Name                   string    `json:"name"`
	Version                string    `json:"version"`
	PublicKey              string    `json:"public_key"`
	ActiveSession          string    `json:"active_session,omitempty"`
	ActiveSessionStartTime time.Time `json:"active_session_start_time"` // latest proof start time.
}

// SessionInfo records proof create or proof verify failed session.
type SessionInfo struct {
	ID              string    `json:"id"`
	Status          string    `json:"status"`
	StartTime       time.Time `json:"start_time"`
	FinishTime      time.Time `json:"finish_time,omitempty"`      // set to 0 if not finished
	AssignedRollers []string  `json:"assigned_rollers,omitempty"` // roller name list
	Error           string    `json:"error,omitempty"`            // empty string if no error encountered
}

// ListRollers returns all live rollers.
func (m *Manager) ListRollers() ([]*RollerInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*RollerInfo
	for _, pk := range m.rollerPool.Keys() {
		node, exist := m.rollerPool.Get(pk)
		if !exist {
			continue
		}
		roller := node.(*rollerNode)
		info := &RollerInfo{
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

func newSessionInfo(sess *session, status orm.ProvingStatus, errMsg string, finished bool) *SessionInfo {
	now := time.Now()
	var nameList []string
	for pk := range sess.info.Rollers {
		nameList = append(nameList, sess.info.Rollers[pk].Name)
	}
	info := SessionInfo{
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
func (m *Manager) GetSessionInfo(sessionID string) (*SessionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if info, ok := m.failedSessionInfos[sessionID]; ok {
		return info, nil
	}
	if s, ok := m.sessions[sessionID]; ok {
		return newSessionInfo(s, orm.ProvingTaskAssigned, "", false), nil
	}
	return nil, fmt.Errorf("no such session, sessionID: %s", sessionID)
}
