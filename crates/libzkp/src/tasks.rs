pub mod batch;
pub mod bundle;
pub mod chunk;
pub mod chunk_interpreter;

pub use batch::BatchProvingTask;
pub use bundle::BundleProvingTask;
pub use chunk::{ChunkProvingTask, ChunkTask};
pub use chunk_interpreter::ChunkInterpreter;
pub use scroll_zkvm_types::task::ProvingTask;

use crate::proofs::{self, BatchProofMetadata, BundleProofMetadata, ChunkProofMetadata};
use sbv_primitives::B256;
use scroll_zkvm_types::public_inputs::{ForkName, MultiVersionPublicInputs};

fn check_aggregation_proofs<Metadata>(
    proofs: &[proofs::WrappedProof<Metadata>],
    fork_name: ForkName,
) -> eyre::Result<()>
where
    Metadata: proofs::ProofMetadata,
{
    use std::panic::{self, AssertUnwindSafe};

    panic::catch_unwind(AssertUnwindSafe(|| {
        for w in proofs.windows(2) {
            w[1].metadata
                .pi_hash_info()
                .validate(w[0].metadata.pi_hash_info(), fork_name);
        }
    }))
    .map_err(|e| {
        let error_msg = if let Some(string) = e.downcast_ref::<String>() {
            string.clone()
        } else if let Some(str) = e.downcast_ref::<&str>() {
            str.to_string()
        } else {
            "Unknown validation error occurred".to_string()
        };
        eyre::eyre!("Chunk data validation failed: {}", error_msg)
    })?;

    Ok(())
}

/// Generate required staff for chunk proving
pub fn gen_universal_chunk_task(
    mut task: ChunkProvingTask,
    fork_name: ForkName,
    interpreter: Option<impl ChunkInterpreter>,
) -> eyre::Result<(B256, ChunkProofMetadata, ProvingTask)> {
    if let Some(interpreter) = interpreter {
        task.prepare_task_via_interpret(interpreter)?;
    }
    let chunk_info = task.precheck_and_build_metadata()?;
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
