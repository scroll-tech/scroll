use openvm_native_recursion::halo2::RawEvmProof;
use scroll_zkvm_circuit_input_types::{bundle::BundleInfo, public_inputs::ForkName};

use crate::{
    Error, Prover, ProverType,
    commitments::{bundle, bundle_euclidv1},
    proof::BundleProofMetadata,
    task::{ProvingTask, bundle::BundleProvingTask},
};

use super::Commitments;

pub struct BundleCircuitV1;
pub struct BundleCircuitV2;

impl Commitments for BundleCircuitV1 {
    const EXE_COMMIT: [u32; 8] = bundle_euclidv1::EXE_COMMIT;
    const LEAF_COMMIT: [u32; 8] = bundle_euclidv1::LEAF_COMMIT;
}

impl Commitments for BundleCircuitV2 {
    const EXE_COMMIT: [u32; 8] = bundle::EXE_COMMIT;
    const LEAF_COMMIT: [u32; 8] = bundle::LEAF_COMMIT;
}

pub type BundleProverTypeEuclidV1 = GenericBundleProverType<BundleCircuitV1>;
pub type BundleProverTypeEuclidV2 = GenericBundleProverType<BundleCircuitV2>;

/// Prover for [`BundleCircuit`].
pub type BundleProverEuclidV1 = Prover<BundleProverTypeEuclidV1>;
pub type BundleProverEuclidV2 = Prover<BundleProverTypeEuclidV2>;

pub struct GenericBundleProverType<C: Commitments>(std::marker::PhantomData<C>);

impl<C: Commitments> ProverType for GenericBundleProverType<C> {
    const NAME: &'static str = "bundle";

    const EVM: bool = true;

    const SEGMENT_SIZE: usize = (1 << 22) - 100;

    const EXE_COMMIT: [u32; 8] = C::EXE_COMMIT;

    const LEAF_COMMIT: [u32; 8] = C::LEAF_COMMIT;

    type ProvingTask = BundleProvingTask;

    type ProofType = RawEvmProof;

    type ProofMetadata = BundleProofMetadata;

    fn metadata_with_prechecks(task: &Self::ProvingTask) -> Result<Self::ProofMetadata, Error> {
        let err_prefix = format!("metadata_with_prechecks for task_id={}", task.identifier());

        for w in task.batch_proofs.windows(2) {
            if w[1].metadata.batch_info.chain_id != w[0].metadata.batch_info.chain_id {
                return Err(Error::GenProof(format!("{err_prefix}: chain_id mismatch")));
            }

            if w[1].metadata.batch_info.parent_state_root != w[0].metadata.batch_info.state_root {
                return Err(Error::GenProof(format!(
                    "{err_prefix}: state_root not chained"
                )));
            }

            if w[1].metadata.batch_info.parent_batch_hash != w[0].metadata.batch_info.batch_hash {
                return Err(Error::GenProof(format!(
                    "{err_prefix}: batch_hash not chained"
                )));
            }
        }

        let (first_batch, last_batch) = (
            &task
                .batch_proofs
                .first()
                .expect("at least one batch in bundle")
                .metadata
                .batch_info,
            &task
                .batch_proofs
                .last()
                .expect("at least one batch in bundle")
                .metadata
                .batch_info,
        );

        let chain_id = first_batch.chain_id;
        let num_batches = u32::try_from(task.batch_proofs.len()).expect("num_batches: u32");
        let prev_state_root = first_batch.parent_state_root;
        let prev_batch_hash = first_batch.parent_batch_hash;
        let post_state_root = last_batch.state_root;
        let batch_hash = last_batch.batch_hash;
        let withdraw_root = last_batch.withdraw_root;
        let msg_queue_hash = last_batch.post_msg_queue_hash;

        let bundle_info = BundleInfo {
            chain_id,
            msg_queue_hash,
            num_batches,
            prev_state_root,
            prev_batch_hash,
            post_state_root,
            batch_hash,
            withdraw_root,
        };

        let fork_name = ForkName::from(task.fork_name.as_str());
        let bundle_pi_hash = bundle_info.pi_hash(fork_name);

        if let Some(checked_bundle_info) = task.bundle_info.as_ref() {
            assert_eq!(
                bundle_pi_hash,
                checked_bundle_info.pi_hash(fork_name),
                "our implement has derived different bundle info with ground truth, got {:?}, expect {:?}",
                bundle_info,
                checked_bundle_info,
            )
        }

        Ok(BundleProofMetadata {
            bundle_info,
            bundle_pi_hash,
        })
    }
}
