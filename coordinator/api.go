package coordinator

import (
	"fmt"
	"time"

	"scroll-tech/database/orm"
)

// RollerInfo records the roller name, pub key and active session info (id, start time).
type RollerInfo struct {
	Name                   string    `json:"name"`
	PublicKey              string    `json:"public_key"`
	ActiveSession          uint64    `json:"active_session,omitempty"`
	ActiveSessionStartTime time.Time `json:"active_session_start_time"` // latest proof start time.
}

// SessionInfo records proof create or proof verify failed session.
type SessionInfo struct {
	ID              uint64    `json:"id"`
	Status          string    `json:"status"`
	StartTime       time.Time `json:"start_time"`
	FinishTime      time.Time `json:"finish_time,omitempty"`      // set to 0 if not finished
	AssignedRollers []string  `json:"assigned_rollers,omitempty"` // roller name list
	Error           string    `json:"error,omitempty"`            // empty string if no error encountered
}

// RollerDebugAPI roller api interface in order go get debug message.
type RollerDebugAPI interface {
	// ListRollers returns all live rollers
	ListRollers() ([]*RollerInfo, error)
	// GetSessionInfo returns the session information given the session id.
	GetSessionInfo(sessionID uint64) (*SessionInfo, error)
}

// ListRollers returns all live rollers.
func (m *Manager) ListRollers() ([]*RollerInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*RollerInfo
	for _, roller := range m.server.conns.getAll() {
		pk := roller.AuthMsg.Identity.PublicKey
		info := &RollerInfo{
			Name:      roller.AuthMsg.Identity.Name,
			PublicKey: pk,
		}
		for id, sess := range m.sessions {
			if sess.rollers[pk] {
				info.ActiveSessionStartTime = sess.startTime
				info.ActiveSession = id
				break
			}
		}
		res = append(res, info)
	}
	return res, nil
}

func newSessionInfo(s *session, status orm.TaskStatus, errMsg string, finished bool) *SessionInfo {
	now := time.Now()
	var nameList []string
	for pk := range s.roller_names {
		nameList = append(nameList, s.roller_names[pk])
	}
	info := SessionInfo{
		ID:              s.id,
		Status:          status.String(),
		AssignedRollers: nameList,
		StartTime:       s.startTime,
		Error:           errMsg,
	}
	if finished {
		info.FinishTime = now
	}
	return &info
}

// GetSessionInfo returns the session information given the session id.
func (m *Manager) GetSessionInfo(sessionID uint64) (*SessionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if info, ok := m.failedSessionInfos[sessionID]; ok {
		return &info, nil
	}
	if s, ok := m.sessions[sessionID]; ok {
		return newSessionInfo(&s, orm.TaskAssigned, "", false), nil
	}
	return nil, fmt.Errorf("no such session, sessionID: %d", sessionID)
}
