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
use scroll_zkvm_types::public_inputs::{ForkName, MultiVersionPublicInputs, Version};

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

/// Generate required staff for chunk proving
pub fn gen_universal_chunk_task(
    task: ChunkProvingTask,
    fork_name: ForkName,
) -> eyre::Result<(B256, ChunkProofMetadata, ProvingTask)> {
    let chunk_total_gas = task.stats().total_gas_used;
    let chunk_info = task.precheck_and_build_metadata()?;
    let proving_task = task.try_into()?;
    let expected_pi_hash = chunk_info.pi_hash_by_fork(fork_name);
    Ok((
        expected_pi_hash,
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
    fork_name: ForkName,
) -> eyre::Result<(B256, BatchProofMetadata, ProvingTask)> {
    let batch_info = task.precheck_and_build_metadata()?;
    let proving_task = task.try_into()?;
    let expected_pi_hash = batch_info.pi_hash_by_fork(fork_name);

    Ok((
        expected_pi_hash,
        BatchProofMetadata {
            batch_info,
            batch_hash: expected_pi_hash,
        },
        proving_task,
    ))
}

/// Generate required staff for bundle proving
pub fn gen_universal_bundle_task(
    task: BundleProvingTask,
    fork_name: ForkName,
) -> eyre::Result<(B256, BundleProofMetadata, ProvingTask)> {
    let bundle_info = task.precheck_and_build_metadata()?;
    let proving_task = task.try_into()?;
    let expected_pi_hash = bundle_info.pi_hash_by_fork(fork_name);

    Ok((
        expected_pi_hash,
        BundleProofMetadata {
            bundle_info,
            bundle_pi_hash: expected_pi_hash,
        },
        proving_task,
    ))
}
