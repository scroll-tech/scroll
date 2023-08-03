package types

const (
	// Success shows OK.
	Success = 0
	// ErrJWTAuthFailure jwt auth failure
	ErrJWTAuthFailure = 20000
	// ErrParameterInvalidNo is invalid params
	ErrParameterInvalidNo = 20001
	// ErrProverTaskFailure is getting prover task  error
	ErrProverTaskFailure = 20002
	// ErrHandleZkProofFailure is handle submit proof error
	ErrHandleZkProofFailure = 20003
	// ErrEmptyProofData get empty proof data
	ErrEmptyProofData = 20004
)
