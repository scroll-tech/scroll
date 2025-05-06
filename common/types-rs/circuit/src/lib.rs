pub mod io;
pub use io::read_witnesses;

use alloy_primitives::B256;
use types_base::public_inputs::PublicInputs;

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
