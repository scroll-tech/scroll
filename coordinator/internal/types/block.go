package types

// BatchInfo contains the BlockBatch's main info
type BatchInfo struct {
	Index     uint64 `json:"index"`
	Hash      string `json:"hash"`
	StateRoot string `json:"state_root"`
}
