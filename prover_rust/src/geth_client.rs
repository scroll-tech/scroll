use crate::types::CommonHash;
use anyhow::Result;
use ethers_core::types::BlockNumber;
use tokio::runtime::Runtime;

use eth_types::{H256, U64};
use serde::{Deserialize, Serialize};

use ethers_providers::{Http, Provider};
use prover::BlockTrace as ProverBlockTrace;

// ======================= types ============================

/// l2 block full trace
#[derive(Deserialize, Serialize, Default, Debug, Clone)]
pub struct BlockTrace {
    #[serde(flatten)]
    pub block_trace: ProverBlockTrace,

    pub version: String,

    pub withdraw_trie_root: Option<CommonHash>,

    #[serde(rename = "mptwitness", default)]
    pub mpt_witness: Vec<u8>,
}

pub fn get_block_number(block_trace: &ProverBlockTrace) -> Option<u64> {
    block_trace.header.number.map(|n| n.as_u64())
}

pub type TxHash = H256;

/// this struct is tracked to https://github.com/scroll-tech/go-ethereum/blob/0f0cd99f7a2e/core/types/block.go#Header
/// the detail fields of struct are not 100% same as eth_types::Block so this needs to be changed in
/// some time currently only the `number` field is required
#[derive(Debug, Deserialize, Serialize, Default)]
pub struct Header {
    #[serde(flatten)]
    block: eth_types::Block<TxHash>,
}

impl Header {
    pub fn get_number(&self) -> Option<U64> {
        self.block.number
    }
}

/// Serialize a type.
///
/// # Panics
///
/// If the type returns an error during serialization.
pub fn serialize<T: serde::Serialize>(t: &T) -> serde_json::Value {
    serde_json::to_value(t).expect("Types never fail to serialize.")
}

// ======================= geth client ============================

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
