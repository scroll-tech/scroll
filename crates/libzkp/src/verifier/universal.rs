use super::{ProofVerifier, TaskType};

use eyre::Result;

use crate::{
    proofs::{AsRootProof, BatchProof, BundleProof, ChunkProof, IntoEvmProof},
    utils::panic_catch,
};
use scroll_zkvm_types::version::Version;
use scroll_zkvm_verifier::verifier::UniversalVerifier;
use std::path::Path;

pub struct Verifier {
    verifier: UniversalVerifier,
    version: Version,
}

impl Verifier {
    pub fn new(assets_dir: &str, ver_n: u8) -> Self {
        let verifier_bin = Path::new(assets_dir);

        Self {
            verifier: UniversalVerifier::setup(verifier_bin).expect("Setting up chunk verifier"),
            version:  Version::from(ver_n),
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
                self.verifier
                    .verify_stark_proof(proof.as_root_proof(), &proof.vk)
                    .unwrap()
            }
            TaskType::Bundle => {
                let proof = serde_json::from_slice::<BundleProof>(proof).unwrap();
                assert!(proof.pi_hash_check(self.version));
                let vk = proof.vk.clone();
                let evm_proof = proof.into_evm_proof();
                self.verifier.verify_evm_proof(&evm_proof, &vk).unwrap()
            }
        })
        .map(|_| true)
        .map_err(|err_str: String| eyre::eyre!("{err_str}"))
    }

    fn dump_vk(&self, _file: &Path) {
        panic!("dump vk has been deprecated");
    }
}
