use serde::{Deserialize, Deserializer};
use std::fmt;

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum ErrorCode {
    Success,
    InternalServerError,

    ErrProverStatsAPIParameterInvalidNo,
    ErrProverStatsAPIProverTaskFailure,
    ErrProverStatsAPIProverTotalRewardFailure,

    ErrCoordinatorParameterInvalidNo,
    ErrCoordinatorGetTaskFailure,
    ErrCoordinatorHandleZkProofFailure,
    ErrCoordinatorEmptyProofData,

    ErrJWTCommonErr,
    ErrJWTTokenExpired,

    Undefined(i32),
}

impl ErrorCode {
    fn from_i32(v: i32) -> Self {
        match v {
            0 => ErrorCode::Success,
            500 => ErrorCode::InternalServerError,
            10001 => ErrorCode::ErrProverStatsAPIParameterInvalidNo,
            10002 => ErrorCode::ErrProverStatsAPIProverTaskFailure,
            10003 => ErrorCode::ErrProverStatsAPIProverTotalRewardFailure,
            20001 => ErrorCode::ErrCoordinatorParameterInvalidNo,
            20002 => ErrorCode::ErrCoordinatorGetTaskFailure,
            20003 => ErrorCode::ErrCoordinatorHandleZkProofFailure,
            20004 => ErrorCode::ErrCoordinatorEmptyProofData,
            50000 => ErrorCode::ErrJWTCommonErr,
            50001 => ErrorCode::ErrJWTTokenExpired,
            _ => {
                log::error!("get unexpected error code from coordinator: {v}");
                ErrorCode::Undefined(v)
            }
        }
    }
}

impl<'de> Deserialize<'de> for ErrorCode {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        let v: i32 = i32::deserialize(deserializer)?;
        Ok(ErrorCode::from_i32(v))
    }
}

// ====================================================

#[derive(Debug, Clone)]
pub struct ProofStatusNotOKError;

impl fmt::Display for ProofStatusNotOKError {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "proof status not ok")
    }
}
