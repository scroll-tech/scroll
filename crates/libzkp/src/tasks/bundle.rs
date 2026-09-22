use eyre::Result;
use sbv_primitives::B256;
use scroll_zkvm_types::{
    public_inputs::{MultiVersionPublicInputs, Version},
    scroll::bundle::{BundleInfo, BundleWitness},
    task::ProvingTask,
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
    pub fn into_proving_task_with_precheck(self) -> Result<(ProvingTask, BundleInfo, B256)> {
        let (witness, bundle_info, bundle_pi_hash) = self.precheck()?;
        let serialized_witness = super::encode_task_to_witness(&witness)?;

        let proving_task = ProvingTask {
            identifier: self.identifier(),
            fork_name: self.fork_name,
            aggregated_proofs: self
                .batch_proofs
                .into_iter()
                .map(|w_proof| w_proof.proof.into_stark_proof().expect("expect root proof"))
                .collect(),
            serialized_witness: vec![serialized_witness],
            vk: Vec::new(),
            input_commits: Vec::new(),
        };

        Ok((proving_task, bundle_info, bundle_pi_hash))
    }

    fn identifier(&self) -> String {
        assert!(!self.batch_proofs.is_empty(), "{BUNDLE_SANITY_MSG}",);

        let (first, last) = (
            self.batch_proofs
                .first()
                .expect(BUNDLE_SANITY_MSG)
                .metadata
                .batch_info
                .batch_hash,
            self.batch_proofs
                .last()
                .expect(BUNDLE_SANITY_MSG)
                .metadata
                .batch_info
                .batch_hash,
        );

        format!("{first}-{last}")
    }

    fn build_guest_input(&self, version: Version) -> BundleWitness {
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

    fn precheck(&self) -> Result<(BundleWitness, BundleInfo, B256)> {
        // for every aggregation task, there are two steps needed to build the metadata:
        // 1. generate data for metadata from the witness
        // 2. validate every adjacent proof pair
        let version = Version::from(self.version);
        let witness = self.build_guest_input(version);
        let metadata = BundleInfo::from(&witness);
        super::check_aggregation_proofs(
            witness.batch_infos.as_slice(),
            Version::from(self.version),
        )?;
        let pi_hash = metadata.pi_hash_by_version(version);

        Ok((witness, metadata, pi_hash))
    }
}
