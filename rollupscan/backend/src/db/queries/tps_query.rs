use crate::db::models::RollupStatus;
use crate::db::{block_result_query, table_name, DbPool};
use rust_decimal::Decimal;
use sqlx::{query_as, Result};

const L1_TPS_WHERE_CLAUSE: &str =
    "status in ($1, $2) and updated_time >= now() - interval '1' hour";
const L2_TPS_WHERE_CLAUSE: &str = "created_time >= now() - interval '1' hour";

pub async fn get_l1_tps(db_pool: &DbPool) -> Result<Decimal> {
    let interval_secs = l1_tps_interval(db_pool).await?;
    let ids = l1_tps_ids(db_pool).await?;
    let tx_num = block_result_query::get_tx_num_by_ids(db_pool, ids).await?;
    log::debug!("L1 TPS: inteval_secs = {interval_secs}, tx_num = {tx_num}");

    Ok(tx_num.checked_div(interval_secs).unwrap_or(Decimal::ZERO))
}

pub async fn get_l2_tps(db_pool: &DbPool) -> Result<Decimal> {
    let interval_secs = l2_tps_interval(db_pool).await?;
    let ids = l2_tps_ids(db_pool).await?;
    let tx_num = block_result_query::get_tx_num_by_ids(db_pool, ids).await?;
    log::debug!("L2 TPS: inteval_secs = {interval_secs}, tx_num = {tx_num}");

    Ok(tx_num.checked_div(interval_secs).unwrap_or(Decimal::ZERO))
}

async fn l1_tps_interval(db_pool: &DbPool) -> Result<Decimal> {
    let stmt = format!(
        "select coalesce(extract(epoch from (now() - min(updated_time))), 0) from {} where {}",
        table_name::ROLLUP_RESULT,
        L1_TPS_WHERE_CLAUSE,
    );
    match query_as::<_, (f64,)>(&stmt)
        .bind(RollupStatus::FinalizationSkipped)
        .bind(RollupStatus::Finalized)
        .fetch_one(db_pool)
        .await
    {
        Ok((secs,)) => Ok(Decimal::from_f64_retain(secs).unwrap_or(Decimal::ZERO)),
        Err(error) => Err(error),
    }
}

async fn l1_tps_ids(db_pool: &DbPool) -> Result<Vec<i32>> {
    let stmt = format!(
        "select number from {} where {}",
        table_name::ROLLUP_RESULT,
        L1_TPS_WHERE_CLAUSE,
    );
    query_as::<_, (i32,)>(&stmt)
        .bind(RollupStatus::FinalizationSkipped)
        .bind(RollupStatus::Finalized)
        .fetch_all(db_pool)
        .await
        .map(|v| v.into_iter().map(|i| i.0).collect())
}

async fn l2_tps_interval(db_pool: &DbPool) -> Result<Decimal> {
    let stmt = format!(
        "select coalesce(extract(epoch from (now() - min(created_time))), 0) from {} where {}",
        table_name::ROLLUP_RESULT,
        L2_TPS_WHERE_CLAUSE,
    );
    match query_as::<_, (f64,)>(&stmt).fetch_one(db_pool).await {
        Ok((secs,)) => Ok(Decimal::from_f64_retain(secs).unwrap_or(Decimal::ZERO)),
        Err(error) => Err(error),
    }
}

async fn l2_tps_ids(db_pool: &DbPool) -> Result<Vec<i32>> {
    let stmt = format!(
        "select number from {} where {}",
        table_name::ROLLUP_RESULT,
        L2_TPS_WHERE_CLAUSE,
    );
    query_as::<_, (i32,)>(&stmt)
        .fetch_all(db_pool)
        .await
        .map(|v| v.into_iter().map(|i| i.0).collect())
}
