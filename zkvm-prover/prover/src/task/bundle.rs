use openvm_native_recursion::hints::Hintable;
use openvm_sdk::StdIn;
use scroll_zkvm_circuit_input_types::{
    bundle::{BundleInfo, BundleWitness},
    public_inputs::ForkName,
};

use crate::{
    BatchProof,
    task::{ProvingTask, flatten_wrapped_proof},
};

/// Message indicating a sanity check failure.
const BUNDLE_SANITY_MSG: &str = "bundle must have at least one batch";

#[derive(Clone, serde::Deserialize, serde::Serialize)]
pub struct BundleProvingTask {
    pub batch_proofs: Vec<BatchProof>,
    /// for sanity check
    pub bundle_info: Option<BundleInfo>,
    /// Fork name specify
    pub fork_name: String,
}

impl ProvingTask for BundleProvingTask {
    fn identifier(&self) -> String {
        assert!(!self.batch_proofs.is_empty(), "{BUNDLE_SANITY_MSG}",);

        let (first, last) = (
            self.batch_proofs
                .first()
                .expect(BUNDLE_SANITY_MSG)
                .metadata
                .batch_hash,
            self.batch_proofs
                .last()
                .expect(BUNDLE_SANITY_MSG)
                .metadata
                .batch_hash,
        );

        format!("{first}-{last}")
    }

    fn fork_name(&self) -> ForkName {
        ForkName::from(self.fork_name.as_str())
    }

    fn build_guest_input(&self) -> Result<StdIn, rkyv::rancor::Error> {
        let witness = BundleWitness {
            batch_proofs: self
                .batch_proofs
                .iter()
                .map(flatten_wrapped_proof)
                .collect(),
            batch_infos: self
                .batch_proofs
                .iter()
                .map(|wrapped_proof| wrapped_proof.metadata.batch_info.clone())
                .collect(),
        };
        let serialized = rkyv::to_bytes::<rkyv::rancor::Error>(&witness)?;
        let mut stdin = StdIn::default();
        stdin.write_bytes(&serialized);
        for batch_proof in &self.batch_proofs {
            let root_input = &batch_proof.as_proof();
            let streams = root_input.write();
            for s in &streams {
                stdin.write_field(s);
            }
        }
        Ok(stdin)
    }
}
