package types

import (
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
)

// RollersInfo is assigned rollers info of a task (session)
type RollersInfo struct {
	ID               string            `json:"id"`
	RollerStatusList []*RollerStatus   `json:"rollers"`
	StartTimestamp   int64             `json:"start_timestamp"`
	ProveType        message.ProofType `json:"prove_type,omitempty"`
}

// RollerStatus is the prover name and prover prove status
type RollerStatus struct {
	PublicKey string                  `json:"public_key"`
	Name      string                  `json:"name"`
	Status    types.ProverProveStatus `json:"status"`
}
