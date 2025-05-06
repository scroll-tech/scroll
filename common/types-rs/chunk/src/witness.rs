use alloy_primitives::B256;
use sbv_primitives::types::BlockWitness;
use std::collections::HashSet;

use types_base::public_inputs::ForkName;

/// The witness type accepted by the chunk-circuit.
#[derive(
    Clone,
    Debug,
    serde::Deserialize,
    serde::Serialize,
    rkyv::Archive,
    rkyv::Deserialize,
    rkyv::Serialize,
)]
#[rkyv(derive(Debug))]
pub struct ChunkWitness {
    /// The block witness for each block in the chunk.
    pub blocks: Vec<BlockWitness>,
    /// The on-chain rolling L1 message queue hash before enqueueing any L1 msg tx from the chunk.
    pub prev_msg_queue_hash: B256,
    /// The code version specify the chain spec
    pub fork_name: ForkName,
}

impl ChunkWitness {
    pub fn new(blocks: &[BlockWitness], prev_msg_queue_hash: B256, fork_name: ForkName) -> Self {
        let num_codes = blocks.iter().map(|w| w.codes.len()).sum();
        let num_states = blocks.iter().map(|w| w.states.len()).sum();
        let mut codes = HashSet::with_capacity(num_codes);
        let mut states = HashSet::with_capacity(num_states);

        let blocks = blocks
            .iter()
            .map(|block| BlockWitness {
                chain_id: block.chain_id,
                header: block.header.clone(),
                pre_state_root: block.pre_state_root,
                transaction: block.transaction.clone(),
                withdrawals: block.withdrawals.clone(),
                states: block
                    .states
                    .iter()
                    .filter(|s| states.insert(*s))
                    .cloned()
                    .collect(),
                codes: block
                    .codes
                    .iter()
                    .filter(|c| codes.insert(*c))
                    .cloned()
                    .collect(),
            })
            .collect();

        Self {
            blocks,
            prev_msg_queue_hash,
            fork_name,
        }
    }

    pub fn new_v1(blocks: &[BlockWitness]) -> Self {
        Self::new(blocks, Default::default(), ForkName::EuclidV1)
    }

    pub fn new_v2(blocks: &[BlockWitness], prev_msg_queue_hash: B256) -> Self {
        Self::new(blocks, prev_msg_queue_hash, ForkName::EuclidV2)
    }
}
