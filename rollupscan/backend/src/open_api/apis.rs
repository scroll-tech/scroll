use crate::db::{block_result_query, rollup_result_query, tps_query};
use crate::open_api::responses::{L2BlocksResponse, LastBlockNumsResponse, TpsResponse};
use crate::open_api::State;
use poem::error::InternalServerError;
use poem::web::Data;
use poem::Result;
use poem_openapi::param::Query;
use poem_openapi::payload::Json;
use poem_openapi::OpenApi;
use std::sync::Arc;

// Expired seconds of cache data.
const L2_BLOCKS_CACHE_EXPIRED_SECS: u64 = 1;
const LAST_BLOCK_NUMS_CACHE_EXPIRED_SECS: u64 = 1;
const TPS_CACHE_EXPIRED_SECS: u64 = 10;

// Query parameter `page` starts from `1`, and default `per_page` is 20.
const DEFAULT_PER_PAGE: u64 = 20;

pub(crate) struct Apis;

#[OpenApi]
impl Apis {
    #[oai(path = "/l1_tps", method = "get")]
    async fn l1_tps(&self, state: Data<&State>) -> Result<Json<TpsResponse>> {
        // Return directly if cached.
        if let Some(response) = TpsResponse::from_cache(state.cache.as_ref(), "l1_tps").await {
            log::debug!("OpenAPI - Get L1 TPS from Cache: {response:?}");
            return Ok(Json(response));
        };

        let tps = tps_query::get_l1_tps(&state.db_pool)
            .await
            .map_err(InternalServerError)?;

        let response = TpsResponse::new(tps);

        // Save to cache.
        if let Err(error) = state
            .cache
            .set("l1_tps", Arc::new(response.clone()), TPS_CACHE_EXPIRED_SECS)
            .await
        {
            log::error!("OpenAPI - Failed to save cache for L1 TPS: {error}");
        }

        Ok(Json(response))
    }

    #[oai(path = "/l2_blocks", method = "get")]
    async fn l2_blocks(
        &self,
        state: Data<&State>,
        page: Query<Option<u64>>,
        per_page: Query<Option<u64>>,
    ) -> Result<Json<L2BlocksResponse>> {
        let limit = per_page.0.map_or_else(
            || DEFAULT_PER_PAGE,
            |val| if val > 0 { val } else { DEFAULT_PER_PAGE },
        );
        let offset = page
            .0
            .map_or_else(|| 0, |val| if val > 0 { (val - 1) * limit } else { 0 });

        // Return directly if cached.
        let cache_key = format!("l2_block-{offset}-{limit}");
        if let Some(response) = L2BlocksResponse::from_cache(state.cache.as_ref(), &cache_key).await
        {
            log::debug!("OpenAPI - Get L2 blocks from Cache: {response:?}");
            return Ok(Json(response));
        };

        let total = rollup_result_query::get_total(&state.db_pool)
            .await
            .map_err(InternalServerError)?;

        let rollup_results = rollup_result_query::fetch_all(&state.db_pool, offset, limit)
            .await
            .map_err(InternalServerError)?;

        let ids = rollup_results.iter().map(|rr| rr.number).collect();
        let block_results = block_result_query::fetch_results_by_ids(&state.db_pool, ids)
            .await
            .map_err(InternalServerError)?;

        let response = L2BlocksResponse::new(total, block_results, rollup_results);

        // Save to cache.
        if let Err(error) = state
            .cache
            .set(
                &cache_key,
                Arc::new(response.clone()),
                L2_BLOCKS_CACHE_EXPIRED_SECS,
            )
            .await
        {
            log::error!("OpenAPI - Failed to save cache for L2 blocks: {error}");
        }

        Ok(Json(response))
    }

    #[oai(path = "/l2_tps", method = "get")]
    async fn l2_tps(&self, state: Data<&State>) -> Result<Json<TpsResponse>> {
        // Return directly if cached.
        if let Some(response) = TpsResponse::from_cache(state.cache.as_ref(), "l2_tps").await {
            log::debug!("OpenAPI - Get L2 TPS from Cache: {response:?}");
            return Ok(Json(response));
        };

        let tps = tps_query::get_l2_tps(&state.db_pool)
            .await
            .map_err(InternalServerError)?;

        let response = TpsResponse::new(tps);

        // Save to cache.
        if let Err(error) = state
            .cache
            .set("l2_tps", Arc::new(response.clone()), TPS_CACHE_EXPIRED_SECS)
            .await
        {
            log::error!("OpenAPI - Failed to save cache for L2 TPS: {error}");
        }

        Ok(Json(response))
    }

    #[oai(path = "/last_block_nums", method = "get")]
    async fn last_block_nums(&self, state: Data<&State>) -> Result<Json<LastBlockNumsResponse>> {
        // Return directly if cached.
        if let Some(response) =
            LastBlockNumsResponse::from_cache(&state.cache, "last_block_nums").await
        {
            log::debug!("OpenAPI - Get last block numbers from Cache: {response:?}");
            return Ok(Json(response));
        };

        let status_nums = rollup_result_query::get_status_max_nums(&state.db_pool)
            .await
            .map_err(InternalServerError)?;
        let response = LastBlockNumsResponse::new(status_nums);

        // Save to cache.
        if let Err(error) = state
            .cache
            .set(
                "last_block_nums",
                Arc::new(response.clone()),
                LAST_BLOCK_NUMS_CACHE_EXPIRED_SECS,
            )
            .await
        {
            log::error!("OpenAPI - Failed to save cache for last block numbers: {error}");
        }

        Ok(Json(response))
    }
}
