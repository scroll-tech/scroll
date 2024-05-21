pub mod types;

use crate::types::CommonHash;
use anyhow::Result;
use ethers_core::types::BlockNumber;
use tokio::runtime::Runtime;
use types::{BlockTrace, Header};

use ethers_providers::{Http, Provider};

/// Serialize a type.
///
/// # Panics
///
/// If the type returns an error during serialization.
pub fn serialize<T: serde::Serialize>(t: &T) -> serde_json::Value {
    serde_json::to_value(t).expect("Types never fail to serialize.")
}

pub struct GethClient {
    id: String,
    provider: Provider<Http>,
    rt: Runtime,
}

impl GethClient {
    pub fn new(id: &str, api_url: &str) -> Result<Self> {
        let provider = Provider::<Http>::try_from(api_url)?;
        let rt = tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()?;

        Ok(Self {
            id: id.to_string(),
            provider,
            rt,
        })
    }

    pub fn get_block_trace_by_hash(&mut self, hash: &CommonHash) -> Result<BlockTrace> {
        log::info!(
            "{}: calling get_block_trace_by_hash, hash: {}",
            self.id,
            hash
        );

        let trace_future = self
            .provider
            .request("scroll_getBlockTraceByNumberOrHash", [format!("{hash:#x}")]);

        let trace = self.rt.block_on(trace_future)?;
        Ok(trace)
    }

    pub fn header_by_number(&mut self, block_number: &BlockNumber) -> Result<Header> {
        log::info!(
            "{}: calling header_by_number, hash: {}",
            self.id,
            block_number
        );

        let hash = serialize(block_number);
        let include_txs = serialize(&false);

        let trace_future = self
            .provider
            .request("eth_getBlockByNumber", [hash, include_txs]);

        let trace = self.rt.block_on(trace_future)?;
        Ok(trace)
    }

    pub fn block_number(&mut self) -> Result<BlockNumber> {
        log::info!("{}: calling block_number", self.id);

        let trace_future = self.provider.request("eth_blockNumber", ());

        let trace = self.rt.block_on(trace_future)?;
        Ok(trace)
    }
}
