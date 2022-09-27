use crate::cache::Cache;
use poem_openapi::Object;
use rust_decimal::Decimal;

#[derive(Clone, Debug, Object)]
pub struct TpsResponse {
    tps: Decimal,
}

impl TpsResponse {
    pub fn new(tps: Decimal) -> Self {
        // Only keep two decimal digits.
        Self {
            tps: tps.round_dp(2),
        }
    }

    pub async fn from_cache(cache: &Cache, cache_key: &str) -> Option<Self> {
        cache
            .get(cache_key)
            .await
            .ok()
            .flatten()
            .and_then(|any| any.downcast_ref::<TpsResponse>().cloned())
    }
}
