use eyre::Result;
use scroll_zkvm_types::{
    bundle::{BundleInfo, BundleWitness, LegacyBundleWitness},
    public_inputs::Version,
    task::ProvingTask,
    utils::{to_rkyv_bytes, RancorError},
};

use crate::proofs::BatchProof;

/// Message indicating a sanity check failure.
const BUNDLE_SANITY_MSG: &str = "bundle must have at least one batch";

#[derive(Clone, serde::Deserialize, serde::Serialize)]
pub struct BundleProvingTask {
    /// The version of batches in the bundle.
    pub version: u8,
    /// The STARK proofs of each batch in the bundle.
    pub batch_proofs: Vec<BatchProof>,
    /// for sanity check
    pub bundle_info: Option<BundleInfo>,
    /// Fork name specify
    pub fork_name: String,
}

impl BundleProvingTask {
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

    fn build_guest_input(&self) -> BundleWitness {
        let version = Version::from(self.version);
        BundleWitness {
            version: version.as_version_byte(),
            batch_proofs: self.batch_proofs.iter().map(|proof| proof.into()).collect(),
            batch_infos: self
                .batch_proofs
                .iter()
                .map(|wrapped_proof| wrapped_proof.metadata.batch_info.clone())
                .collect(),
            fork_name: version.fork,
        }
    }

    pub fn precheck_and_build_metadata(&self) -> Result<BundleInfo> {
        // for every aggregation task, there are two steps needed to build the metadata:
        // 1. generate data for metadata from the witness
        // 2. validate every adjacent proof pair
        let witness = self.build_guest_input();
        let metadata = BundleInfo::from(&witness);
        super::check_aggregation_proofs(self.batch_proofs.as_slice(), Version::from(self.version))?;

        Ok(metadata)
    }
}

impl TryFrom<BundleProvingTask> for ProvingTask {
    type Error = eyre::Error;

    fn try_from(value: BundleProvingTask) -> Result<Self> {
        let witness = value.build_guest_input();
        let serialized_witness = if crate::witness_use_legacy_mode() {
            let legacy = LegacyBundleWitness::from(witness);
            to_rkyv_bytes::<RancorError>(&legacy)?.into_vec()
        } else {
            super::encode_task_to_witness(&witness)?
        };

        Ok(ProvingTask {
            identifier: value.identifier(),
            fork_name: value.fork_name,
            aggregated_proofs: value
                .batch_proofs
                .into_iter()
                .map(|w_proof| w_proof.proof.into_stark_proof().expect("expect root proof"))
                .collect(),
            serialized_witness: vec![serialized_witness],
            vk: Vec::new(),
        })
    }
}
