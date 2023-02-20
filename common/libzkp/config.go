package libzkp

// ProverConfig load zk prover config.
type ProverConfig struct {
	ParamsPath string `json:"params_path"`
	SeedPath   string `json:"seed_path"`
}

// VerifierConfig load zk verifier config.
type VerifierConfig struct {
	MockMode   bool   `json:"mock_mode"`
	ParamsPath string `json:"params_path"`
	AggVkPath  string `json:"agg_vk_path"`
}
