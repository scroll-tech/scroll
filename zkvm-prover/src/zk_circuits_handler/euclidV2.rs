use std::{path::Path, sync::Arc};

use super::CircuitsHandler;
use anyhow::{anyhow, Result};
use async_trait::async_trait;
use scroll_proving_sdk::prover::{proving_service::ProveRequest, ProofType};
use scroll_zkvm_prover_euclidv2::{
    task::{batch::BatchProvingTask, bundle::BundleProvingTask, chunk::ChunkProvingTask},
    BatchProver, BundleProver, ChunkProver,
};
use tokio::sync::Mutex;
pub struct EuclidV2Handler {
    chunk_prover: ChunkProver,
    batch_prover: BatchProver,
    bundle_prover: BundleProver,
}

unsafe impl Send for EuclidV2Handler {}

impl EuclidV2Handler {
    pub fn new(workspace_path: &str) -> Self {
        let workspace_path = Path::new(workspace_path);
        println!("ok 1");

        let cache_dir = workspace_path.join("cache");
        let chunk_exe = workspace_path.join("chunk/app.vmexe");
        let chunk_app_config = workspace_path.join("chunk/openvm.toml");
        let chunk_prover = ChunkProver::setup(
            chunk_exe,
            chunk_app_config,
            Some(cache_dir.clone()),
            Default::default(),
        )
        .expect("Failed to setup chunk prover");
        println!("ok 2");

        let batch_exe = workspace_path.join("batch/app.vmexe");
        let batch_app_config = workspace_path.join("batch/openvm.toml");
        let batch_prover = BatchProver::setup(
            batch_exe,
            batch_app_config,
            Some(cache_dir.clone()),
            Default::default(),
        )
        .expect("Failed to setup batch prover");
        println!("ok 3");

        let bundle_exe = workspace_path.join("bundle/app.vmexe");
        let bundle_app_config = workspace_path.join("bundle/openvm.toml");
        let bundle_prover = BundleProver::setup(
            bundle_exe,
            bundle_app_config,
            Some(cache_dir),
            Default::default(),
        )
        .expect("Failed to setup bundle prover");
        println!("ok 4");

        Self {
            chunk_prover,
            batch_prover,
            bundle_prover,
        }
    }
}

#[async_trait]
impl CircuitsHandler for Arc<Mutex<EuclidV2Handler>> {
    async fn get_vk(&self, task_type: ProofType) -> Option<Vec<u8>> {
        Some(match task_type {
            ProofType::Chunk => self.try_lock().unwrap().chunk_prover.get_app_vk(),
            ProofType::Batch => self.try_lock().unwrap().batch_prover.get_app_vk(),
            ProofType::Bundle => self.try_lock().unwrap().bundle_prover.get_app_vk(),
            _ => unreachable!("Unsupported proof type"),
        })
    }

    async fn get_proof_data(&self, prove_request: ProveRequest) -> Result<String> {
        match prove_request.proof_type {
            ProofType::Chunk => {
                let task: ChunkProvingTask = serde_json::from_str(&prove_request.input)?;
                let proof = self.try_lock().unwrap().chunk_prover.gen_proof(&task)?;

                Ok(serde_json::to_string(&proof)?)
            }
            ProofType::Batch => {
                let task: BatchProvingTask = serde_json::from_str(&prove_request.input)?;
                let proof = self.try_lock().unwrap().batch_prover.gen_proof(&task)?;

                Ok(serde_json::to_string(&proof)?)
            }
            ProofType::Bundle => {
                let batch_proofs: BundleProvingTask = serde_json::from_str(&prove_request.input)?;
                let proof = self
                    .try_lock()
                    .unwrap()
                    .bundle_prover
                    .gen_proof_evm(&batch_proofs)?;

                Ok(serde_json::to_string(&proof)?)
            }
            _ => Err(anyhow!("Unsupported proof type")),
        }
    }
}
