use crate::proofs::BatchProof;
use eyre::Result;
use scroll_zkvm_types::{
    bundle::{BundleInfo, BundleWitness, ToArchievedWitness},
    public_inputs::ForkName,
    task::ProvingTask,
    utils::{to_rkyv_bytes, RancorError},
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
        BundleWitness {
            batch_proofs: self.batch_proofs.iter().map(|proof| proof.into()).collect(),
            batch_infos: self
                .batch_proofs
                .iter()
                .map(|wrapped_proof| wrapped_proof.metadata.batch_info.clone())
                .collect(),
        }
    }

    pub fn precheck_and_build_metadata(&self) -> Result<BundleInfo> {
        let fork_name = ForkName::from(self.fork_name.as_str());
        // for every aggregation task, there are two steps needed to build the metadata:
        // 1. generate data for metadata from the witness
        // 2. validate every adjacent proof pair
        let witness = self.build_guest_input();
        let archieved = ToArchievedWitness::create(&witness)
            .map_err(|e| eyre::eyre!("archieve bundle witness fail: {e}"))?;
        let archieved_witness = archieved
            .access()
            .map_err(|e| eyre::eyre!("access archieved bundle witness fail: {e}"))?;
        let metadata: BundleInfo = archieved_witness.into();

        super::check_aggregation_proofs(self.batch_proofs.as_slice(), fork_name)?;

        Ok(metadata)
    }
}

impl TryFrom<BundleProvingTask> for ProvingTask {
    type Error = eyre::Error;

    fn try_from(value: BundleProvingTask) -> Result<Self> {
        let witness = value.build_guest_input();

        Ok(ProvingTask {
            identifier: value.identifier(),
            fork_name: value.fork_name,
            aggregated_proofs: value
                .batch_proofs
                .into_iter()
                .map(|w_proof| w_proof.proof.into_root_proof().expect("expect root proof"))
                .collect(),
            serialized_witness: vec![to_rkyv_bytes::<RancorError>(&witness)?.to_vec()],
            vk: Vec::new(),
        })
    }
}
