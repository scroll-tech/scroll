use crate::db::models::{BlockResult, RollupResult, RollupStatus};
use poem_openapi::Object;
use rust_decimal::Decimal;
use std::collections::HashMap;

#[derive(Clone, Debug, Object)]
pub struct L2Block {
    block_height: i32,
    tx_num: i64,
    status: String,
    header_hash: String, // hash
    rollup_tx_hash: String,
    finalize_tx_hash: String,
    block_timestamp: Decimal,
}

pub fn build_l2_blocks_by_db_results(
    block_results: Vec<BlockResult>,
    rollup_results: Vec<RollupResult>,
) -> Vec<L2Block> {
    let block_results: HashMap<_, _> =
        HashMap::from_iter(block_results.iter().map(|br| (br.number, br)));

    rollup_results
        .into_iter()
        .map(|rr| {
            let id = rr.number;
            let (tx_num, header_hash, block_timestamp) = block_results
                .get(&id)
                .map(|br| (br.tx_num, br.hash.clone(), br.block_timestamp))
                .unwrap_or((0, "".to_string(), Decimal::ZERO));
            let status = rollup_status_to_str(&rr.status).to_string();
            L2Block {
                block_height: id,
                tx_num,
                status,
                header_hash,
                rollup_tx_hash: rr.rollup_tx_hash.unwrap_or_else(|| "".to_string()),
                finalize_tx_hash: rr.finalize_tx_hash.unwrap_or_else(|| "".to_string()),
                block_timestamp,
            }
        })
        .collect()
}

pub fn rollup_status_to_str(status: &RollupStatus) -> &str {
    match status {
        RollupStatus::Undefined => "unknown",
        RollupStatus::Pending | RollupStatus::Committing => "precommitted",
        RollupStatus::Committed | RollupStatus::Finalizing => "committed",
        RollupStatus::Finalized => "finalized",
        RollupStatus::FinalizationSkipped => "skipped",
    }
}
