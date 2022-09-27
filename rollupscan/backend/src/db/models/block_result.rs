use rust_decimal::Decimal;
use serde::Serialize;
use std::fmt;

#[derive(sqlx::Type, Clone, Debug, Serialize)]
#[repr(i32)]
pub enum BlockStatus {
    Undefined = 0,
    Unassigned,
    Skipped,
    Assigned,
    Proved,
    Verified,
    Failed,
}

impl fmt::Display for BlockStatus {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "{:?}", self)
    }
}

#[derive(sqlx::FromRow, Clone, Debug, Serialize)]
pub struct BlockResult {
    pub number: i32,
    pub tx_num: i64,
    pub hash: String,
    pub status: BlockStatus,
    pub block_timestamp: Decimal,
}
