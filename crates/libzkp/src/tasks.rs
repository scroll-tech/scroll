pub mod batch;
pub mod bundle;
pub mod chunk;
pub mod chunk_interpreter;

pub use batch::BatchProvingTask;
pub use bundle::BundleProvingTask;
pub use chunk::{ChunkProvingTask, ChunkTask};
pub use chunk_interpreter::ChunkInterpreter;
pub use scroll_zkvm_types::task::ProvingTask;

use crate::{
    proofs::{BatchProofMetadata, BundleProofMetadata, ChunkProofMetadata},
    utils::panic_catch,
};
use sbv_primitives::B256;
use scroll_zkvm_types::public_inputs::{MultiVersionPublicInputs, Version};

fn encode_task_to_witness<T: serde::Serialize>(task: &T) -> eyre::Result<Vec<u8>> {
    let config = bincode::config::standard();
    Ok(bincode::serde::encode_to_vec(task, config)?)
}

fn check_aggregation_proofs<Metadata: MultiVersionPublicInputs>(
    metadata: &[Metadata],
    version: Version,
) -> eyre::Result<()> {
    panic_catch(|| {
        for w in metadata.windows(2) {
            w[1].validate(&w[0], version);
        }
    })
    .map_err(|e| eyre::eyre!("Metadata validation failed: {}", e))?;

    Ok(())
}

#[derive(serde::Deserialize, serde::Serialize)]
pub struct ProvingTaskExt {
    #[serde(flatten)]
    task: ProvingTask,
    #[serde(default)]
    pub use_openvm_13: bool,
}

impl From<ProvingTaskExt> for ProvingTask {
    fn from(wrap_t: ProvingTaskExt) -> Self {
        wrap_t.task
    }
}

impl ProvingTaskExt {
    pub fn new(task: ProvingTask) -> Self {
        Self {
            task,
            use_openvm_13: false,
        }
    }
}

/// Generate required staff for chunk proving
pub fn gen_universal_chunk_task(
    task: ChunkProvingTask,
) -> eyre::Result<(B256, ChunkProofMetadata, ProvingTask)> {
    let chunk_total_gas = task.stats().total_gas_used;
    let (proving_task, chunk_info, chunk_pi_hash) = task.into_proving_task_with_precheck()?;
    Ok((
        chunk_pi_hash,
        ChunkProofMetadata {
            chunk_info,
            chunk_total_gas,
        },
        proving_task,
    ))
}

/// Generate required staff for batch proving
pub fn gen_universal_batch_task(
    task: BatchProvingTask,
) -> eyre::Result<(B256, BatchProofMetadata, ProvingTask)> {
    let (proving_task, batch_info, batch_pi_hash) = task.into_proving_task_with_precheck()?;
    Ok((
        batch_pi_hash,
        BatchProofMetadata { batch_info },
        proving_task,
    ))
}

/// Generate required staff for bundle proving
pub fn gen_universal_bundle_task(
    task: BundleProvingTask,
) -> eyre::Result<(B256, BundleProofMetadata, ProvingTask)> {
    let (proving_task, bundle_info, bundle_pi_hash) = task.into_proving_task_with_precheck()?;
    Ok((
        bundle_pi_hash,
        BundleProofMetadata {
            bundle_info,
            bundle_pi_hash,
        },
        proving_task,
    ))
}
