/// Represents an openvm program commitments and public values.
#[derive(
    Clone,
    Debug,
    rkyv::Archive,
    rkyv::Deserialize,
    rkyv::Serialize,
    serde::Deserialize,
    serde::Serialize,
)]
#[rkyv(derive(Debug))]
pub struct AggregationInput {
    /// Public values.
    pub public_values: Vec<u32>,
    /// Represent the commitment needed to verify a root proof
    pub commitment: ProgramCommitment,
}

/// Represent the commitment needed to verify a [`RootProof`].
#[derive(
    Clone,
    Debug,
    Default,
    rkyv::Archive,
    rkyv::Deserialize,
    rkyv::Serialize,
    serde::Deserialize,
    serde::Serialize,
)]
#[rkyv(derive(Debug))]
pub struct ProgramCommitment {
    /// The commitment to the child program exe.
    pub exe: [u32; 8],
    /// The commitment to the child program leaf.
    pub leaf: [u32; 8],
}

impl ProgramCommitment {
    pub fn deserialize(commitment_bytes: &[u8]) -> Self {
        // TODO: temporary skip deserialize if no vk is provided
        if commitment_bytes.is_empty() {
            return Default::default();
        }

        let archived_data =
            rkyv::access::<ArchivedProgramCommitment, rkyv::rancor::BoxedError>(commitment_bytes)
                .unwrap();

        Self {
            exe: archived_data.exe.map(|u32_le| u32_le.to_native()),
            leaf: archived_data.leaf.map(|u32_le| u32_le.to_native()),
        }
    }

    pub fn serialize(&self) -> Vec<u8> {
        rkyv::to_bytes::<rkyv::rancor::BoxedError>(self)
            .map(|v| v.to_vec())
            .unwrap()
    }
}

impl From<&ArchivedProgramCommitment> for ProgramCommitment {
    fn from(archived: &ArchivedProgramCommitment) -> Self {
        Self {
            exe: archived.exe.map(|u32_le| u32_le.to_native()),
            leaf: archived.leaf.map(|u32_le| u32_le.to_native()),
        }
    }
}

/// Number of public-input values, i.e. [u32; N].
///
/// Note that the actual value for each u32 is a byte.
pub const NUM_PUBLIC_VALUES: usize = 32;

/// Witness for an [`AggregationCircuit`][AggCircuit] that also carries proofs that are being
/// aggregated.
pub trait ProofCarryingWitness {
    /// Get the root proofs from the witness.
    fn get_proofs(&self) -> Vec<AggregationInput>;
}
