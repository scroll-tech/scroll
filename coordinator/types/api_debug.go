package types

import "time"

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
