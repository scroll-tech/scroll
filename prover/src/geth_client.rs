use crate::types::CommonHash;
use anyhow::Result;
use ethers_core::types::BlockNumber;
use tokio::runtime::Runtime;

use serde::{de::DeserializeOwned, Serialize};
use std::fmt::Debug;

use ethers_providers::{Http, Provider};

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

    pub fn get_block_trace_by_hash<T>(&mut self, hash: &CommonHash) -> Result<T>
    where
        T: Serialize + DeserializeOwned + Debug + Send,
    {
        log::info!(
            "{}: calling get_block_trace_by_hash, hash: {:#?}",
            self.id,
            hash
        );

        let trace_future = self
            .provider
            .request("scroll_getBlockTraceByNumberOrHash", [format!("{hash:#x}")]);

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
