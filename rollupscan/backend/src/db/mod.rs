pub mod models;
pub mod queries;

pub use queries::*;

pub type DbPool = sqlx::postgres::PgPool;

pub mod table_name {
    pub const BLOCK_RESULT: &str = "block_result";
    pub const ROLLUP_RESULT: &str = "rollup_result";
}
