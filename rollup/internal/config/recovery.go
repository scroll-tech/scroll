package config

type RecoveryConfig struct {
	Enable bool `json:"enable"`

	L1BeaconNodeEndpoint string `json:"l1_beacon_node_endpoint"` // the L1 beacon node endpoint to connect to
	L1BlockHeight        uint64 `json:"l1_block_height"`         // the L1 block height to start recovery from
	LatestFinalizedBatch uint64 `json:"latest_finalized_batch"`  // the latest finalized batch number
	L2BlockHeightLimit   uint64 `json:"l2_block_height_limit"`   // L2 block up to which to produce batch

	ForceLatestFinalizedBatch bool   `json:"force_latest_finalized_batch"` // whether to force usage of the latest finalized batch - mainly used for testing
	ForceL1MessageCount       uint64 `json:"force_l1_message_count"`       // force the number of L1 messages, useful for testing
	SubmitWithoutProof        bool   `json:"submit_without_proof"`         // whether to submit batches without proof, useful for testing
}
