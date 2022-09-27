use crate::db::models::BlockResult;
use crate::db::{table_name, DbPool};
use rust_decimal::Decimal;
use sqlx::{query_as, Result};

pub async fn fetch_results_by_ids(db_pool: &DbPool, ids: Vec<i32>) -> Result<Vec<BlockResult>> {
    let stmt = format!(
        "select
            number, tx_num, hash, status, block_timestamp
        from {} where number = any($1)",
        table_name::BLOCK_RESULT,
    );
    query_as::<_, BlockResult>(&stmt)
        .bind(ids)
        .fetch_all(db_pool)
        .await
}

pub async fn get_tx_num_by_ids(db_pool: &DbPool, ids: Vec<i32>) -> Result<Decimal> {
    let stmt = format!(
        "select sum(tx_num)
        from {} where number = any($1)",
        table_name::BLOCK_RESULT,
    );
    query_as::<_, (Option<Decimal>,)>(&stmt)
        .bind(ids)
        .fetch_one(db_pool)
        .await
        .map(|d| d.0.unwrap_or(Decimal::ZERO))
}
