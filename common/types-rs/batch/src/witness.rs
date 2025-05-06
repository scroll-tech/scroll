use crate::header::ReferenceHeader;
use types_agg::{AggregationInput, ProgramCommitment, ProofCarryingWitness};
use types_base::public_inputs::{ForkName, chunk::ChunkInfo};

/// Simply rewrap byte48 to avoid unnecessary dep
pub type Bytes48 = [u8; 48];

/// Witness required by applying point evaluation
#[derive(Clone, Debug, rkyv::Archive, rkyv::Deserialize, rkyv::Serialize)]
#[rkyv(derive(Debug))]
pub struct PointEvalWitness {
    /// kzg commitment
    #[rkyv()]
    pub kzg_commitment: Bytes48,
    /// kzg proof
    #[rkyv()]
    pub kzg_proof: Bytes48,
}

/// Witness to the batch circuit.
#[derive(Clone, Debug, rkyv::Archive, rkyv::Deserialize, rkyv::Serialize)]
#[rkyv(derive(Debug))]
pub struct BatchWitness {
    /// Flattened root proofs from all chunks in the batch.
    #[rkyv()]
    pub chunk_proofs: Vec<AggregationInput>,
    /// Chunk infos.
    #[rkyv()]
    pub chunk_infos: Vec<ChunkInfo>,
    /// Blob bytes.
    #[rkyv()]
    pub blob_bytes: Vec<u8>,
    /// Witness for point evaluation
    pub point_eval_witness: PointEvalWitness,
    /// Header for reference.
    #[rkyv()]
    pub reference_header: ReferenceHeader,
    /// The code version specify the chain spec
    #[rkyv()]
    pub fork_name: ForkName,
}

impl ProofCarryingWitness for ArchivedBatchWitness {
    fn get_proofs(&self) -> Vec<AggregationInput> {
        self.chunk_proofs
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
