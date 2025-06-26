pub mod batch;
pub mod bundle;
pub mod chunk;
pub mod chunk_interpreter;

pub use batch::BatchProvingTask;
pub use bundle::BundleProvingTask;
pub use chunk::{ChunkProvingTask, ChunkTask};
pub use chunk_interpreter::ChunkInterpreter;
pub use scroll_zkvm_types::task::ProvingTask;

use crate::proofs::{BatchProofMetadata, BundleProofMetadata, ChunkProofMetadata};
use chunk_interpreter::{DummyInterpreter, TryFromWithInterpreter};
use sbv_primitives::B256;
use scroll_zkvm_types::{
    chunk::ChunkInfo,
    public_inputs::{ForkName, MultiVersionPublicInputs},
};

/// Generate required staff for chunk proving
pub fn gen_universal_chunk_task(
    mut task: ChunkProvingTask,
    fork_name: ForkName,
    interpreter: Option<impl ChunkInterpreter>,
) -> eyre::Result<(B256, ChunkProofMetadata, ProvingTask)> {
    let chunk_info = if let Some(interpreter) = interpreter {
        ChunkInfo::try_from_with_interpret(&mut task, interpreter)
    } else {
        ChunkInfo::try_from_with_interpret(&mut task, DummyInterpreter {})
    }?;
    let proving_task = task.try_into()?;
    let expected_pi_hash = chunk_info.pi_hash_by_fork(fork_name);
    Ok((
        expected_pi_hash,
        ChunkProofMetadata { chunk_info },
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
