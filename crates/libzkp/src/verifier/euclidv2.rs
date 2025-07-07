use super::{ProofVerifier, TaskType};

use eyre::Result;

use crate::{
    proofs::{AsRootProof, BatchProof, BundleProof, ChunkProof, IntoEvmProof},
    utils::panic_catch,
};
use scroll_zkvm_types::public_inputs::ForkName;
use scroll_zkvm_verifier_euclid::verifier::UniversalVerifier;
use std::path::Path;

pub struct EuclidV2Verifier {
    verifier: UniversalVerifier,
    fork: ForkName,
}

impl EuclidV2Verifier {
    pub fn new(assets_dir: &str, fork: ForkName) -> Self {
        let verifier_bin = Path::new(assets_dir).join("verifier.bin");
        let config = Path::new(assets_dir).join("root-verifier-vm-config");
        let exe = Path::new(assets_dir).join("root-verifier-committed-exe");

        Self {
            verifier: UniversalVerifier::setup(&config, &exe, &verifier_bin)
                .expect("Setting up chunk verifier"),
            fork,
        }
    }
}

impl ProofVerifier for EuclidV2Verifier {
    fn verify(&self, task_type: super::TaskType, proof: &[u8]) -> Result<bool> {
        panic_catch(|| match task_type {
            TaskType::Chunk => {
                let proof = serde_json::from_slice::<ChunkProof>(proof).unwrap();
                if !proof.pi_hash_check(self.fork) {
                    return false;
                }
                self.verifier
                    .verify_proof(proof.as_root_proof(), &proof.vk)
                    .unwrap()
            }
            TaskType::Batch => {
                let proof = serde_json::from_slice::<BatchProof>(proof).unwrap();
                if !proof.pi_hash_check(self.fork) {
                    return false;
                }
                self.verifier
                    .verify_proof(proof.as_root_proof(), &proof.vk)
                    .unwrap()
            }
            TaskType::Bundle => {
                let proof = serde_json::from_slice::<BundleProof>(proof).unwrap();
                if !proof.pi_hash_check(self.fork) {
                    return false;
                }
                let vk = proof.vk.clone();
                let evm_proof = proof.into_evm_proof();
                self.verifier.verify_proof_evm(&evm_proof, &vk).unwrap()
            }
        })
        .map_err(|err_str: String| eyre::eyre!("{err_str}"))
    }

    fn dump_vk(&self, _file: &Path) {
        panic!("dump vk has been deprecated");
    }
}
