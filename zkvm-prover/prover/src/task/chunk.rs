use alloy_primitives::B256;
use openvm_sdk::StdIn;
use sbv_primitives::types::BlockWitness;
use scroll_zkvm_circuit_input_types::{chunk::ChunkWitness, public_inputs::ForkName};

use crate::task::ProvingTask;

/// Message indicating a sanity check failure.
const CHUNK_SANITY_MSG: &str = "chunk must have at least one block";

/// Proving task for the [`ChunkCircuit`][scroll_zkvm_chunk_circuit].
///
/// The identifier for a chunk proving task is:
/// - {first_block_number}-{last_block_number}
#[derive(Clone, Debug, serde::Deserialize, serde::Serialize)]
pub struct ChunkProvingTask {
    /// Witnesses for every block in the chunk.
    pub block_witnesses: Vec<BlockWitness>,
    /// The on-chain L1 msg queue hash before applying L1 msg txs from the chunk.
    pub prev_msg_queue_hash: B256,
    /// Fork name specify
    pub fork_name: String,
}

#[derive(Clone, Debug)]
pub struct ChunkDetails {
    pub num_blocks: usize,
    pub num_txs: usize,
    pub total_gas_used: u64,
}

impl ChunkProvingTask {
    pub fn stats(&self) -> ChunkDetails {
        let num_blocks = self.block_witnesses.len();
        let num_txs = self
            .block_witnesses
            .iter()
            .map(|b| b.transaction.len())
            .sum::<usize>();
        let total_gas_used = self
            .block_witnesses
            .iter()
            .map(|b| b.header.gas_used)
            .sum::<u64>();

        ChunkDetails {
            num_blocks,
            num_txs,
            total_gas_used,
        }
    }
}

impl ProvingTask for ChunkProvingTask {
    fn identifier(&self) -> String {
        assert!(!self.block_witnesses.is_empty(), "{CHUNK_SANITY_MSG}",);

        let (first, last) = (
            self.block_witnesses
                .first()
                .expect(CHUNK_SANITY_MSG)
                .header
                .number,
            self.block_witnesses
                .last()
                .expect(CHUNK_SANITY_MSG)
                .header
                .number,
        );

        format!("{first}-{last}")
    }

    fn fork_name(&self) -> ForkName {
        ForkName::from(self.fork_name.as_str())
    }

    fn build_guest_input(&self) -> Result<StdIn, rkyv::rancor::Error> {
        let witness = ChunkWitness {
            blocks: self.block_witnesses.to_vec(),
            prev_msg_queue_hash: self.prev_msg_queue_hash,
            fork_name: self.fork_name.to_lowercase().as_str().into(),
        };

        let serialized = rkyv::to_bytes::<rkyv::rancor::Error>(&witness)?;

        let mut stdin = StdIn::default();
        stdin.write_bytes(&serialized);
        Ok(stdin)
    }
}
