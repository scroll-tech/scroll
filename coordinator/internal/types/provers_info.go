package types

import (
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
)

// ProversInfo is assigned provers info of a task (session)
type ProversInfo struct {
	ID               string            `json:"id"`
	ProverStatusList []*ProverStatus   `json:"provers"`
	StartTimestamp   int64             `json:"start_timestamp"`
	ProveType        message.ProofType `json:"prove_type,omitempty"`
}

// ProverStatus is the prover name and prover prove status
type ProverStatus struct {
	PublicKey string                  `json:"public_key"`
	Name      string                  `json:"name"`
	Status    types.ProverProveStatus `json:"status"`
}
