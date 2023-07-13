package types

// ProverTaskFailureType the type of prover task failure
type ProverTaskFailureType int

const (
	// ProverTaskFailureTypeUnknown prover task unknown error
	ProverTaskFailureTypeUnknown ProverTaskFailureType = iota
	// ProverTaskFailureTypeTimeout prover task failure of timeout
	ProverTaskFailureTypeTimeout
)

func (r ProverTaskFailureType) String() string {
	switch r {
	case ProverTaskFailureTypeUnknown:
		return "prover task failure unknown"
	case ProverTaskFailureTypeTimeout:
		return "prover task failure timeout"
	default:
		return "illegal failure type"
	}
}
