package types

const (
	// Success shows OK.
	Success = 0
	// InternalServerError show server fatal
	InternalServerError = 500

	// ErrJWTCommonErr jwt common error
	ErrJWTCommonErr = 50000
	// ErrJWTTokenExpired jwt token expired
	ErrJWTTokenExpired = 50001

	// ErrCoordinatorParameterInvalidNo is invalid params
	ErrCoordinatorParameterInvalidNo = 20001
	// ErrCoordinatorGetTaskFailure is getting prover task error
	ErrCoordinatorGetTaskFailure = 20002
	// ErrCoordinatorHandleZkProofFailure is handle submit proof error
	ErrCoordinatorHandleZkProofFailure = 20003
	// ErrCoordinatorEmptyProofData get empty proof data
	ErrCoordinatorEmptyProofData = 20004
)
