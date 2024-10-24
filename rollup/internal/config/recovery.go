package config

// L1Config loads l1eth configuration items.
type RecoveryConfig struct {
	Enable bool `json:"enable"`

	LatestFinalizedBatch uint64 `json:"latest_finalized_batch"`
	L1BlockHeight        uint64 `json:"l1_block_height"`

	L2BlockHeightLimit uint64 `json:"l2_block_height_limit"`
}
