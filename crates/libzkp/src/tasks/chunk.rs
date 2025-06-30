use super::chunk_interpreter::*;
use eyre::Result;
use sbv_primitives::{types::BlockWitness, B256};
use scroll_zkvm_types::{
    chunk::{execute, ChunkInfo, ChunkWitness, ToArchievedWitness},
    task::ProvingTask,
    utils::{to_rkyv_bytes, RancorError},
};

/// The type aligned with coordinator's defination
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct ChunkTask {
    /// block hashes for a series of block
    pub block_hashes: Vec<B256>,
    /// The on-chain L1 msg queue hash before applying L1 msg txs from the chunk.
    pub prev_msg_queue_hash: B256,
    /// Fork name specify
    pub fork_name: String,
}

impl TryFromWithInterpreter<ChunkTask> for ChunkProvingTask {
    fn try_from_with_interpret(
        value: ChunkTask,
        interpreter: impl ChunkInterpreter,
    ) -> Result<Self> {
        let mut block_witnesses = Vec::new();
        for block_hash in value.block_hashes {
            let witness =
                interpreter.try_fetch_block_witness(block_hash, block_witnesses.last())?;
            block_witnesses.push(witness);
        }

        Ok(Self {
            block_witnesses,
            prev_msg_queue_hash: value.prev_msg_queue_hash,
            fork_name: value.fork_name,
        })
    }
}

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

impl TryFrom<ChunkProvingTask> for ProvingTask {
    type Error = eyre::Error;

    fn try_from(value: ChunkProvingTask) -> Result<Self> {
        let witness = value.build_guest_input();

        Ok(ProvingTask {
            identifier: value.identifier(),
            fork_name: value.fork_name,
            aggregated_proofs: Vec::new(),
            serialized_witness: vec![to_rkyv_bytes::<RancorError>(&witness)?.to_vec()],
            vk: Vec::new(),
        })
    }
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

    fn build_guest_input(&self) -> ChunkWitness {
        ChunkWitness {
            blocks: self.block_witnesses.to_vec(),
            prev_msg_queue_hash: self.prev_msg_queue_hash,
            fork_name: self.fork_name.to_lowercase().as_str().into(),
        }
    }

    fn insert_state(&mut self, node: sbv_primitives::Bytes) {
        self.block_witnesses[0].states.push(node);
    }

    pub fn precheck_and_build_metadata(&self) -> Result<ChunkInfo> {
        let witness = self.build_guest_input();
        let archieved = ToArchievedWitness::create(&witness)
            .map_err(|e| eyre::eyre!("archieve chunk witness fail: {e}"))?;
        let archieved_witness = archieved
            .access()
            .map_err(|e| eyre::eyre!("access archieved chunk witness fail: {e}"))?;

        let ret = ChunkInfo::try_from(archieved_witness).map_err(|e| eyre::eyre!("{e}"))?;
        Ok(ret)
    }

    /// this method check the validate of current task (there may be missing storage node)
    /// and try fixing it until everything is ok
    pub fn prepare_task_via_interpret(
        &mut self,
        interpreter: impl ChunkInterpreter,
    ) -> eyre::Result<()> {
        use eyre::eyre;

        let err_prefix = format!(
            "metadata_with_prechecks for task_id={:?}",
            self.identifier()
        );

        if self.block_witnesses.is_empty() {
            return Err(eyre!(
                "{err_prefix}: chunk should contain at least one block",
            ));
        }

        // resume from node missing error and keep executing process
        let pattern = r"SparseTrieError\(BlindedNode \{ path: Nibbles\((0x[0-9a-fA-F]+)\), hash: (0x[0-9a-fA-F]+) \}\)";
        let err_parse_re = regex::Regex::new(pattern)?;
        let mut attempts = 0;
        loop {
            let witness = self.build_guest_input();
            let archieved = ToArchievedWitness::create(&witness)
                .map_err(|e| eyre::eyre!("archieve chunk witness fail: {e}"))?;
            let archieved_witness = archieved
                .access()
                .map_err(|e| eyre::eyre!("access archieved chunk witness fail: {e}"))?;

            match execute(archieved_witness) {
                Ok(_) => return Ok(()),
                Err(e) => {
                    if let Some(caps) = err_parse_re.captures(&e) {
                        let hash = caps[2].to_string();
                        tracing::debug!("missing trie hash {hash}");

                        attempts += 1;
                        if attempts >= MAX_FETCH_NODES_ATTEMPTS {
                            return Err(eyre!(
                            "failed to fetch nodes after {MAX_FETCH_NODES_ATTEMPTS} attempts: {e}"
                        ));
                        }

                        let node_hash =
                            hash.parse::<sbv_primitives::B256>().expect("should be hex");
                        let node = interpreter.try_fetch_storage_node(node_hash)?;
                        tracing::warn!("missing node fetched: {node}");
                        self.insert_state(node);
                    } else {
                        return Err(eyre!("{err_prefix}: {e}"));
                    }
                }
            }
        }
    }
}

const MAX_FETCH_NODES_ATTEMPTS: usize = 15;
