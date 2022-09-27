use crate::db::models::{RollupResult, RollupStatus};
use crate::db::{table_name, DbPool};
use sqlx::{query_as, query_scalar, Result};
use std::collections::HashMap;

pub async fn fetch_all(db_pool: &DbPool, offset: u64, limit: u64) -> Result<Vec<RollupResult>> {
    let stmt = format!(
        "select
            number, status, rollup_tx_hash, finalize_tx_hash, created_time, updated_time
        from {} order by number desc offset {} limit {}",
        table_name::ROLLUP_RESULT,
        offset,
        limit,
    );
    query_as::<_, RollupResult>(&stmt).fetch_all(db_pool).await
}

pub async fn get_total(db_pool: &DbPool) -> Result<i32> {
    let stmt = format!(
        "select coalesce(max(number), 0) FROM {}",
        table_name::ROLLUP_RESULT,
    );
    match query_scalar::<_, i32>(&stmt).fetch_one(db_pool).await {
        Ok(max_num) => Ok(max_num),
        Err(error) => Err(error),
    }
}

pub async fn get_status_max_nums(db_pool: &DbPool) -> Result<HashMap<RollupStatus, i32>> {
    let stmt = format!(
        "select status, max(number) FROM {} group by status",
        table_name::ROLLUP_RESULT,
    );
    query_as::<_, (RollupStatus, i32)>(&stmt)
        .fetch_all(db_pool)
        .await
        .map(|v| v.into_iter().collect())
}
