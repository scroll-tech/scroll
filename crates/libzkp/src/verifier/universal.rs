use super::{ProofVerifier, TaskType};

use eyre::Result;

use crate::{
    proofs::{AsRootProof, BatchProof, BundleProof, ChunkProof, IntoEvmProof},
    utils::panic_catch,
};
use openvm_sdk::SC;
use scroll_zkvm_types::version::Version;
use scroll_zkvm_verifier::verifier::UniversalVerifier;
use std::path::Path;

pub struct Verifier {
    verifier: UniversalVerifier,
    /// Deferral-enabled root verifier VK for batch proofs (v0.9.0+).
    /// Loaded from `batch_root_verifier_vk` if present in the assets directory.
    batch_mvk: Option<openvm_stark_sdk::openvm_stark_backend::keygen::types::MultiStarkVerifyingKey<SC>>,
    version: Version,
}

impl Verifier {
    pub fn new(assets_dir: &str, ver_n: u8) -> Self {
        let verifier_bin = Path::new(assets_dir);

        let verifier =
            UniversalVerifier::setup(verifier_bin).expect("Setting up universal verifier");

        let batch_mvk_path = verifier_bin.join("batch_root_verifier_vk");
        let batch_mvk = if batch_mvk_path.exists() {
            Some(
                openvm_sdk::fs::read_object_from_file(&batch_mvk_path)
                    .expect("Reading batch root verifier vk"),
            )
        } else {
            None
        };

        Self {
            verifier,
            batch_mvk,
            version: Version::from(ver_n),
        }
    }
}

impl ProofVerifier for Verifier {
    fn verify(&self, task_type: super::TaskType, proof: &[u8]) -> Result<bool> {
        panic_catch(|| match task_type {
            TaskType::Chunk => {
                let proof = serde_json::from_slice::<ChunkProof>(proof).unwrap();
                assert!(proof.pi_hash_check(self.version));
                self.verifier
                    .verify_stark_proof(proof.as_root_proof(), &proof.vk)
                    .unwrap()
            }
            TaskType::Batch => {
                let proof = serde_json::from_slice::<BatchProof>(proof).unwrap();
                assert!(proof.pi_hash_check(self.version));
                let mvk = self
                    .batch_mvk
                    .as_ref()
                    .expect("batch_root_verifier_vk missing from assets");
                UniversalVerifier::verify_stark_proof_with_vk(
                    mvk,
                    proof.as_root_proof(),
                    &proof.vk,
                )
                .unwrap()
            }
            TaskType::Bundle => {
                let proof = serde_json::from_slice::<BundleProof>(proof).unwrap();
                assert!(proof.pi_hash_check(self.version));
                let vk = proof.vk.clone();
                let evm_proof = proof.into_evm_proof();
                self.verifier.verify_evm_proof(&evm_proof, &vk).unwrap();
            }
        })
        .map(|_| true)
        .map_err(|err_str: String| eyre::eyre!("{err_str}"))
    }

    fn dump_vk(&self, _file: &Path) {
        panic!("dump vk has been deprecated");
    }
}
