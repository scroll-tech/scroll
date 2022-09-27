use crate::cache::Cache;
use crate::db::models::RollupStatus;
use crate::open_api::objects::rollup_status_to_str;
use poem_openapi::Object;
use std::collections::HashMap;

#[derive(Clone, Debug, Object)]
pub struct LastBlockNumsResponse {
    committed_num: i32,
    finalized_num: i32,
    precommitted_num: i32,
}

impl LastBlockNumsResponse {
    pub fn new(status_nums: HashMap<RollupStatus, i32>) -> Self {
        let mut committed_num = 0;
        let mut precommitted_num = 0;
        let mut finalized_num = 0;

        for (status, num) in status_nums.into_iter() {
            match rollup_status_to_str(&status) {
                "precommitted" => precommitted_num = precommitted_num.max(num),
                "committed" => committed_num = committed_num.max(num),
                "finalized" => finalized_num = finalized_num.max(num),
                _ => (),
            }
        }

        // Set `committed` number as maximum number of both committed and
        // finalized blocks.
        committed_num = committed_num.max(finalized_num);

        // Set `pre-committed` number as maximum number of all pre-committed,
        // committed and finalized blocks.
        precommitted_num = precommitted_num.max(committed_num);

        Self {
            committed_num,
            precommitted_num,
            finalized_num,
        }
    }

    pub async fn from_cache(cache: &Cache, cache_key: &str) -> Option<Self> {
        cache
            .get(cache_key)
            .await
            .ok()
            .flatten()
            .and_then(|any| any.downcast_ref::<Self>().cloned())
    }
}
