package relayer

type Config struct {
	// Host for the grpc server
	Host string `mapstructure:"Host"`
	// Port for the grpc server
	Port int `mapstructure:"Port"`

	// RetryTime is the time the aggregator main loop sleeps if there are no proofs to aggregate
	// or batches to generate proofs. It is also used in the isSynced loop
	RetryTime types.Duration `mapstructure:"RetryTime"`

	// VerifyProofInterval is the interval of time to verify/send an proof in L1
	VerifyProofInterval types.Duration `mapstructure:"VerifyProofInterval"`

	// ProofStatePollingInterval is the interval time to polling the prover about the generation state of a proof
	ProofStatePollingInterval types.Duration `mapstructure:"ProofStatePollingInterval"`

	// TxProfitabilityCheckerType type for checking is it profitable for aggregator to validate batch
	// possible values: base/acceptall
	TxProfitabilityCheckerType TxProfitabilityCheckerType `mapstructure:"TxProfitabilityCheckerType"`

	// TxProfitabilityMinReward min reward for base tx profitability checker when aggregator will validate batch
	// this parameter is used for the base tx profitability checker
	TxProfitabilityMinReward TokenAmountWithDecimals `mapstructure:"TxProfitabilityMinReward"`

	// IntervalAfterWhichBatchConsolidateAnyway this is interval for the main sequencer, that will check if there is no transactions
	IntervalAfterWhichBatchConsolidateAnyway types.Duration `mapstructure:"IntervalAfterWhichBatchConsolidateAnyway"`

	// ChainID is the L2 ChainID provided by the Network Config
	ChainID uint64

	// ForkID is the L2 ForkID provided by the Network Config
	ForkId uint64 `mapstructure:"ForkId"`

	// SenderAddress defines which private key the eth tx manager needs to use
	// to sign the L1 txs
	SenderAddress string `mapstructure:"SenderAddress"`

	// CleanupLockedProofsInterval is the interval of time to clean up locked proofs.
	CleanupLockedProofsInterval types.Duration `mapstructure:"CleanupLockedProofsInterval"`

	// GeneratingProofCleanupThreshold represents the time interval after
	// which a proof in generating state is considered to be stuck and
	// allowed to be cleared.
	GeneratingProofCleanupThreshold string `mapstructure:"GeneratingProofCleanupThreshold"`

	StartBatchNum uint64 `mapstructure:"StartBatchNum"`

	MaxProofHashFlight int `mapstructure:"MaxProofHashFlight"`
}