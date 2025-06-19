use super::{ProofVerifier, TaskType, VKDump};

use eyre::Result;

use crate::{
    proofs::{AsRootProof, BatchProof, BundleProof, ChunkProof, IntoEvmProof},
    utils::panic_catch,
};
use scroll_zkvm_verifier_euclid::verifier::{BatchVerifier, BundleVerifierEuclidV2, ChunkVerifier};
use std::{fs::File, path::Path};

pub struct EuclidV2Verifier {
    chunk_verifier: ChunkVerifier,
    batch_verifier: BatchVerifier,
    bundle_verifier: BundleVerifierEuclidV2,
}

impl EuclidV2Verifier {
    pub fn new(assets_dir: &str) -> Self {
        let verifier_bin = Path::new(assets_dir).join("verifier.bin");
        let config = Path::new(assets_dir).join("root-verifier-vm-config");
        let exe = Path::new(assets_dir).join("root-verifier-committed-exe");

        Self {
            chunk_verifier: ChunkVerifier::setup(&config, &exe, &verifier_bin)
                .expect("Setting up chunk verifier"),
            batch_verifier: BatchVerifier::setup(&config, &exe, &verifier_bin)
                .expect("Setting up batch verifier"),
            bundle_verifier: BundleVerifierEuclidV2::setup(&config, &exe, &verifier_bin)
                .expect("Setting up bundle verifier"),
        }
    }
}

impl ProofVerifier for EuclidV2Verifier {
    fn verify(&self, task_type: super::TaskType, proof: Vec<u8>) -> Result<bool> {
        panic_catch(|| match task_type {
            TaskType::Chunk => {
                let proof = serde_json::from_slice::<ChunkProof>(proof.as_slice()).unwrap();
                self.chunk_verifier.verify_proof(proof.as_root_proof())
            }
            TaskType::Batch => {
                let proof = serde_json::from_slice::<BatchProof>(proof.as_slice()).unwrap();
                self.batch_verifier.verify_proof(proof.as_root_proof())
            }
            TaskType::Bundle => {
                let proof = serde_json::from_slice::<BundleProof>(proof.as_slice()).unwrap();
                self.bundle_verifier
                    .verify_proof_evm(&proof.into_evm_proof())
            }
        })
        .map_err(|err_str: String| eyre::eyre!("{err_str}"))
    }

    fn dump_vk(&self, file: &Path) {
        use base64::{prelude::BASE64_STANDARD, Engine};
        let f = File::create(file).expect("Failed to open file to dump VK");

        let dump = VKDump {
            chunk_vk: BASE64_STANDARD.encode(self.chunk_verifier.get_app_vk()),
            batch_vk: BASE64_STANDARD.encode(self.batch_verifier.get_app_vk()),
            bundle_vk: BASE64_STANDARD.encode(self.bundle_verifier.get_app_vk()),
        };
        serde_json::to_writer(f, &dump).expect("Failed to dump VK");
    }
}
