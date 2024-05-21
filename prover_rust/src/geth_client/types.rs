use eth_types::{H256, U64};
use serde::{Deserialize, Serialize};

use crate::types::CommonHash;
use prover::BlockTrace as ProverBlockTrace;

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
