package api

//
//import (
//	"fmt"
//	"scroll-tech/common/types"
//	"time"
//)
//
//type RollerDebug struct {
//}
//
//func NewRollerDebug() *RollerDebug {
//	return &RollerDebug{}
//}
//
//// ListRollers returns all live rollers.
//func (m *RollerDebug) ListRollers() ([]*RollerInfo, error) {
//	m.mu.RLock()
//	defer m.mu.RUnlock()
//	var res []*RollerInfo
//	for _, pk := range m.rollerPool.Keys() {
//		node, exist := m.rollerPool.Get(pk)
//		if !exist {
//			continue
//		}
//		roller := node.(*rollerNode)
//		info := &RollerInfo{
//			Name:      roller.Name,
//			Version:   roller.Version,
//			PublicKey: pk,
//		}
//		for id, sess := range m.sessions {
//			if _, ok := sess.info.Rollers[pk]; ok {
//				info.ActiveSessionStartTime = time.Unix(sess.info.StartTimestamp, 0)
//				info.ActiveSession = id
//				break
//			}
//		}
//		res = append(res, info)
//	}
//
//	return res, nil
//}
//
//func newSessionInfo(sess *session, status types.ProvingStatus, errMsg string, finished bool) *SessionInfo {
//	now := time.Now()
//	var nameList []string
//	for pk := range sess.info.Rollers {
//		nameList = append(nameList, sess.info.Rollers[pk].Name)
//	}
//	info := SessionInfo{
//		ID:              sess.info.ID,
//		Status:          status.String(),
//		AssignedRollers: nameList,
//		StartTime:       time.Unix(sess.info.StartTimestamp, 0),
//		Error:           errMsg,
//	}
//	if finished {
//		info.FinishTime = now
//	}
//	return &info
//}
//
//// GetSessionInfo returns the session information given the session id.
//func (m *RollerDebug) GetSessionInfo(sessionID string) (*SessionInfo, error) {
//	m.mu.RLock()
//	defer m.mu.RUnlock()
//	if info, ok := m.failedSessionInfos[sessionID]; ok {
//		return info, nil
//	}
//	if s, ok := m.sessions[sessionID]; ok {
//		return newSessionInfo(s, types.ProvingTaskAssigned, "", false), nil
//	}
//	return nil, fmt.Errorf("no such session, sessionID: %s", sessionID)
//}
