use types_agg::{AggregationInput, ProgramCommitment, ProofCarryingWitness};
use types_base::public_inputs::batch::BatchInfo;

/// The witness for the bundle circuit.
#[derive(Clone, Debug, rkyv::Archive, rkyv::Deserialize, rkyv::Serialize)]
#[rkyv(derive(Debug))]
pub struct BundleWitness {
    /// Batch proofs being aggregated in the bundle.
    #[rkyv()]
    pub batch_proofs: Vec<AggregationInput>,
    /// Public-input values for the corresponding batch proofs.
    #[rkyv()]
    pub batch_infos: Vec<BatchInfo>,
}

impl ProofCarryingWitness for ArchivedBundleWitness {
    fn get_proofs(&self) -> Vec<AggregationInput> {
        self.batch_proofs
            .iter()
            .map(|archived| AggregationInput {
                public_values: archived
                    .public_values
                    .iter()
                    .map(|u32_le| u32_le.to_native())
                    .collect(),
                commitment: ProgramCommitment::from(&archived.commitment),
            })
            .collect()
    }
}
