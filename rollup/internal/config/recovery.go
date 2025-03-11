package config

type RecoveryConfig struct {
	Enable bool `json:"enable"`

	L1BeaconNodeEndpoint      string `json:"l1_beacon_node_endpoint"`
	L1BlockHeight             uint64 `json:"l1_block_height"`
	LatestFinalizedBatch      uint64 `json:"latest_finalized_batch"`       // the latest finalized batch number
	ForceLatestFinalizedBatch bool   `json:"force_latest_finalized_batch"` // whether to force usage of the latest finalized batch - mainly used for testing

	L2BlockHeightLimit  uint64 `json:"l2_block_height_limit"`
	ForceL1MessageCount uint64 `json:"force_l1_message_count"`
}
