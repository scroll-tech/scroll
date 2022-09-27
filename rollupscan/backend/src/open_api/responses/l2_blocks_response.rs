use crate::cache::Cache;
use crate::db::models::{BlockResult, RollupResult};
use crate::open_api::objects::{build_l2_blocks_by_db_results, L2Block};
use poem_openapi::Object;

#[derive(Clone, Debug, Object)]
pub struct L2BlocksResponse {
    total: i32,
    blocks: Vec<L2Block>,
}

impl L2BlocksResponse {
    pub fn new(
        total: i32,
        block_results: Vec<BlockResult>,
        rollup_results: Vec<RollupResult>,
    ) -> Self {
        let blocks = build_l2_blocks_by_db_results(block_results, rollup_results);
        Self { total, blocks }
    }

    pub async fn from_cache(cache: &Cache, cache_key: &str) -> Option<Self> {
        cache
            .get(cache_key)
            .await
            .ok()
            .flatten()
            .and_then(|any| any.downcast_ref::<L2BlocksResponse>().cloned())
    }
}
