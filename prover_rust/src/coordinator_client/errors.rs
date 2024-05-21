// TODO: refactor using enum
pub type ErrorCode = i32;

pub const Success: ErrorCode = 0;
pub const InternalServerError: ErrorCode = 500;

pub const ErrJWTCommonErr: ErrorCode = 50000;
pub const ErrJWTTokenExpired: ErrorCode = 50001;

pub const ErrProverStatsAPIParameterInvalidNo: ErrorCode = 10001;
pub const ErrProverStatsAPIProverTaskFailure: ErrorCode = 10002;
pub const ErrProverStatsAPIProverTotalRewardFailure: ErrorCode = 10003;

pub const ErrCoordinatorParameterInvalidNo: ErrorCode = 20001;
pub const ErrCoordinatorGetTaskFailure: ErrorCode = 20002;
pub const ErrCoordinatorHandleZkProofFailure: ErrorCode = 20003;
pub const ErrCoordinatorEmptyProofData: ErrorCode = 20004;
