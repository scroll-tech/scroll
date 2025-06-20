use crate::proofs::BatchProof;
use eyre::Result;
use scroll_zkvm_types::{
    bundle::{BundleInfo, BundleWitness},
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
        use eyre::eyre;
        let err_prefix = format!("metadata_with_prechecks for task_id={}", self.identifier());

        for w in self.batch_proofs.windows(2) {
            if w[1].metadata.batch_info.chain_id != w[0].metadata.batch_info.chain_id {
                return Err(eyre!("{err_prefix}: chain_id mismatch"));
            }

            if w[1].metadata.batch_info.parent_state_root != w[0].metadata.batch_info.state_root {
                return Err(eyre!("{err_prefix}: state_root not chained"));
            }

            if w[1].metadata.batch_info.parent_batch_hash != w[0].metadata.batch_info.batch_hash {
                return Err(eyre!("{err_prefix}: batch_hash not chained"));
            }
        }

        let (first_batch, last_batch) = (
            &self
                .batch_proofs
                .first()
                .expect("at least one batch in bundle")
                .metadata
                .batch_info,
            &self
                .batch_proofs
                .last()
                .expect("at least one batch in bundle")
                .metadata
                .batch_info,
        );

        let chain_id = first_batch.chain_id;
        let num_batches = u32::try_from(self.batch_proofs.len()).expect("num_batches: u32");
        let prev_state_root = first_batch.parent_state_root;
        let prev_batch_hash = first_batch.parent_batch_hash;
        let post_state_root = last_batch.state_root;
        let batch_hash = last_batch.batch_hash;
        let withdraw_root = last_batch.withdraw_root;
        let msg_queue_hash = last_batch.post_msg_queue_hash;

        Ok(BundleInfo {
            chain_id,
            msg_queue_hash,
            num_batches,
            prev_state_root,
            prev_batch_hash,
            post_state_root,
            batch_hash,
            withdraw_root,
        })
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
