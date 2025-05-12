pub mod io;
pub use io::read_witnesses;

use alloy_primitives::B256;
use itertools::Itertools;
use types_agg::{AggregationInput, ProgramCommitment, ProofCarryingWitness, verify_proof};

pub use types_base::public_inputs::PublicInputs;

/// Reveal the public-input values as openvm public values.
pub fn reveal_pi_hash(pi_hash: B256) {
    openvm::io::println(format!("pi_hash = {pi_hash:?}"));
    openvm::io::reveal_bytes32(*pi_hash);
}

/// Circuit defines the higher-level behaviour to be observed by a [`openvm`] guest program.
pub trait Circuit {
    /// The witness provided to the circuit.
    type Witness;

    /// The public-input values for the circuit.
    type PublicInputs: PublicInputs;

    /// Setup openvm extensions as a preliminary step.
    fn setup();

    /// Reads bytes from openvm StdIn.
    fn read_witness_bytes() -> Vec<u8>;

    /// Deserialize raw bytes into the circuit's witness type.
    fn deserialize_witness(witness_bytes: &[u8]) -> &Self::Witness;

    /// Validate the witness to produce the circuit's public inputs.
    fn validate(witness: &Self::Witness) -> Self::PublicInputs;

    /// Reveal the public inputs.
    fn reveal_pi(pi: &Self::PublicInputs) {
        reveal_pi_hash(pi.pi_hash())
    }
}

/// Circuit that additional aggregates proofs from other [`Circuits`][Circuit].
pub trait AggCircuit: Circuit
where
    Self::Witness: ProofCarryingWitness,
{
    /// The public-input values of the proofs being aggregated.
    type AggregatedPublicInputs: PublicInputs;

    /// Check if the commitment in proof is valid (from program(s)
    /// we have expected)
    fn verify_commitments(commitment: &ProgramCommitment);

    /// Verify the proofs being aggregated.
    ///
    /// Also returns the root proofs being aggregated.
    fn verify_proofs(witness: &Self::Witness) -> Vec<AggregationInput> {
        let proofs = witness.get_proofs();

        for proof in proofs.iter() {
            Self::verify_commitments(&proof.commitment);
            verify_proof(&proof.commitment, proof.public_values.as_slice());
        }

        proofs
    }

    /// Derive the public-input values of the proofs being aggregated from the witness.
    fn aggregated_public_inputs(witness: &Self::Witness) -> Vec<Self::AggregatedPublicInputs>;

    /// Derive the public-input hashes of the aggregated proofs from the proofs itself.
    fn aggregated_pi_hashes(proofs: &[AggregationInput]) -> Vec<B256>;

    /// Validate that the public-input values of the aggregated proofs are well-formed.
    ///
    /// - That the public-inputs of contiguous chunks/batches are valid
    /// - That the public-input values in fact hash to the pi_hash values from the root proofs.
    fn validate_aggregated_pi(agg_pis: &[Self::AggregatedPublicInputs], agg_pi_hashes: &[B256]) {
        // There should be at least a single proof being aggregated.
        assert!(!agg_pis.is_empty(), "at least 1 pi to aggregate");

        // Validation for the contiguous public-input values.
        for w in agg_pis.windows(2) {
            w[1].validate(&w[0]);
        }

        // Validation for public-input values hash being the pi_hash from root proof.
        for (agg_pi, &agg_pi_hash) in agg_pis.iter().zip_eq(agg_pi_hashes.iter()) {
            assert_eq!(
                agg_pi.pi_hash(),
                agg_pi_hash,
                "pi hash mismatch between proofs and witness computed"
            );
        }
    }
}
