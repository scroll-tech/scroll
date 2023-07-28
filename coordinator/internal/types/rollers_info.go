package types

import (
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
)

// RollersInfo is assigned provers info of a task (session)
type RollersInfo struct {
	ID               string            `json:"id"`
	RollerStatusList []*RollerStatus   `json:"provers"`
	StartTimestamp   int64             `json:"start_timestamp"`
	ProveType        message.ProofType `json:"prove_type,omitempty"`
}

// RollerStatus is the prover name and prover prove status
type RollerStatus struct {
	PublicKey string                  `json:"public_key"`
	Name      string                  `json:"name"`
	Status    types.RollerProveStatus `json:"status"`
}
