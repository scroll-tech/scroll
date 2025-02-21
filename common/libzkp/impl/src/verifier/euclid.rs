use super::{ProofVerifier, TaskType, VKDump};

use anyhow::Result;

use crate::utils::panic_catch;
use prover_v7::{BatchProof, BatchProver, BundleProof, BundleProver, ChunkProof, ChunkProver};
use std::{env, fs::File, path::Path};

pub struct EuclidVerifier {
    chunk_verifier: ChunkProver,
    batch_verifier: BatchProver,
    bundle_verifier: BundleProver,
}

impl EuclidVerifier {
    pub fn new(assets_dir: &str) -> Self {
        env::set_var("SCROLL_PROVER_ASSETS_DIR", assets_dir);
        let zkvm_release_path = Path::new(assets_dir).join("scroll-zkvm").join("current");
        let chunk_exe = zkvm_release_path.join("chunk/app.vmexe");
        let chunk_app_config = zkvm_release_path.join("chunk/openvm.toml");
        let chunk_verifier = ChunkProver::setup(chunk_exe, chunk_app_config, None).unwrap();

        let batch_exe = zkvm_release_path.join("batch/app.vmexe");
        let batch_app_config = zkvm_release_path.join("batch/openvm.toml");
        let batch_verifier = BatchProver::setup(batch_exe, batch_app_config, None).unwrap();

        let bundle_exe = zkvm_release_path.join("bundle/app.vmexe");
        let bundle_app_config = zkvm_release_path.join("bundle/openvm.toml");
        let bundle_verifier = BundleProver::setup(bundle_exe, bundle_app_config, None).unwrap();

        Self {
            chunk_verifier,
            batch_verifier,
            bundle_verifier,
        }
    }
}

impl ProofVerifier for EuclidVerifier {
    fn verify(&self, task_type: super::TaskType, proof: Vec<u8>) -> Result<bool> {
        panic_catch(|| match task_type {
            TaskType::Chunk => {
                let proof = serde_json::from_slice::<ChunkProof>(proof.as_slice()).unwrap();
                self.chunk_verifier.verify_proof(&proof).is_ok()
            }
            TaskType::Batch => {
                let proof = serde_json::from_slice::<BatchProof>(proof.as_slice()).unwrap();
                self.batch_verifier.verify_proof(&proof).is_ok()
            }
            TaskType::Bundle => {
                let proof = serde_json::from_slice::<BundleProof>(proof.as_slice()).unwrap();
                self.bundle_verifier.verify_proof_evm(&proof).is_ok()
            }
        })
        .map_err(|err_str| anyhow::anyhow!(err_str))
    }

    fn dump_vk(&self, file: &Path) {
        let f = File::create(file).expect("Failed to open file to dump VK");

        let dump = VKDump {
            chunk_vk: base64::encode(self.chunk_verifier.get_app_vk()),
            batch_vk: base64::encode(self.batch_verifier.get_app_vk()),
            bundle_vk: base64::encode(self.bundle_verifier.get_app_vk()),
        };
        serde_json::to_writer(f, &dump).expect("Failed to dump VK");
    }
}
