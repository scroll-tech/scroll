package coordinator

import (
	"fmt"
	"time"

	"scroll-tech/common/types"
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
			for _, sessionInfo := range sess.sessionInfos {
				if sessionInfo.RollerPublicKey == pk {
					info.ActiveSessionStartTime = *sessionInfo.CreatedAt
					info.ActiveSession = id
					break
				}
			}
		}
		res = append(res, info)
	}

	return res, nil
}

func newSessionInfo(sess *session, status types.ProvingStatus, errMsg string, finished bool) *SessionInfo {
	now := time.Now()
	var nameList []string
	for _, sessionInfo := range sess.sessionInfos {
		nameList = append(nameList, sessionInfo.RollerName)
	}
	info := SessionInfo{
		ID:              sess.taskID,
		Status:          status.String(),
		AssignedRollers: nameList,
		StartTime:       *sess.sessionInfos[0].CreatedAt,
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
		return newSessionInfo(s, types.ProvingTaskAssigned, "", false), nil
	}
	return nil, fmt.Errorf("no such session, sessionID: %s", sessionID)
}

// GetRollerSubmissions returns all submissions by given roller pubkey.
func (m *Manager) GetRollerSubmissions(pubKey string) ([]*types.SubmissionInfo, error) {
	return m.orm.GetSubmissionInfosByRoller(pubKey)
}

// GetTotalRewards returns the total rewards by given roller pubkey.
func (m *Manager) GetTotalRewards(pubKey string) (uint64, error) {
	subs, err := m.orm.GetSubmissionInfosByRoller(pubKey)
	if err != nil {
		return 0, err
	}
	var total uint64
	for _, sub := range subs {
		if types.ProvingStatus(sub.ProvingStatus) == types.ProvingTaskVerified {
			total += sub.Reward
		}
	}
	return total, nil
}

// GetSubmission returns submission by given task id.
func (m *Manager) GetSubmission(taskID string) (sub *types.SubmissionInfo, err error) {
	var subs []*types.SubmissionInfo
	subs, err = m.orm.GetSubmissionInfosByHashes([]string{taskID})
	if err != nil {
		return
	}
	if len(subs) > 0 {
		sub = subs[0]
	}
	return
}
