use crate::{
    Error, Prover, ProverType,
    commitments::batch::{EXE_COMMIT as BATCH_EXE_COMMIT, LEAF_COMMIT as BATCH_LEAF_COMMIT},
    proof::{BatchProofMetadata, RootProof},
    task::batch::BatchProvingTask,
};

/// Prover for [`BatchCircuit`].
pub type BatchProver = Prover<BatchProverType>;

pub struct BatchProverType;

impl ProverType for BatchProverType {
    const NAME: &'static str = "batch";

    const EVM: bool = false;

    const SEGMENT_SIZE: usize = (1 << 22) - 100;

    const EXE_COMMIT: [u32; 8] = BATCH_EXE_COMMIT;

    const LEAF_COMMIT: [u32; 8] = BATCH_LEAF_COMMIT;

    type ProvingTask = BatchProvingTask;

    type ProofType = RootProof;

    type ProofMetadata = BatchProofMetadata;

    fn metadata_with_prechecks(task: &Self::ProvingTask) -> Result<Self::ProofMetadata, Error> {
        let batch_info = task.into();
        let batch_hash = task.batch_header.batch_hash();

        Ok(BatchProofMetadata {
            batch_info,
            batch_hash,
        })
    }
}
