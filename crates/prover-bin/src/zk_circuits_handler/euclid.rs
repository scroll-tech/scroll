use std::{path::Path, sync::Arc};

use super::CircuitsHandler;
use anyhow::{anyhow, Result};
use async_trait::async_trait;
use scroll_proving_sdk::prover::{proving_service::ProveRequest, ProofType};
use scroll_zkvm_prover_euclid::{
    task::{batch::BatchProvingTask, bundle::BundleProvingTask, chunk::ChunkProvingTask},
    BatchProver, BundleProverEuclidV1, ChunkProver, ProverConfig,
};
use tokio::sync::Mutex;
pub struct EuclidHandler {
    chunk_prover: ChunkProver,
    batch_prover: BatchProver,
    bundle_prover: BundleProverEuclidV1,
}

#[derive(Clone, Copy)]
pub(crate) enum Phase {
    EuclidV1,
    EuclidV2,
}

impl Phase {
    pub fn as_str(&self) -> &str {
        match self {
            Phase::EuclidV1 => "euclidv1",
            Phase::EuclidV2 => "euclidv2",
        }
    }

    pub fn phase_spec_chunk(&self, workspace_path: &Path) -> ProverConfig {
        let dir_cache = Some(workspace_path.join("cache"));
        let path_app_exe = workspace_path.join("chunk/app.vmexe");
        let path_app_config = workspace_path.join("chunk/openvm.toml");
        let segment_len = Some((1 << 22) - 100);
        ProverConfig {
            dir_cache,
            path_app_config,
            path_app_exe,
            segment_len,
            ..Default::default()
        }
    }

    pub fn phase_spec_batch(&self, workspace_path: &Path) -> ProverConfig {
        let dir_cache = Some(workspace_path.join("cache"));
        let path_app_exe = workspace_path.join("batch/app.vmexe");
        let path_app_config = workspace_path.join("batch/openvm.toml");
        let segment_len = Some((1 << 22) - 100);
        ProverConfig {
            dir_cache,
            path_app_config,
            path_app_exe,
            segment_len,
            ..Default::default()
        }
    }

    pub fn phase_spec_bundle(&self, workspace_path: &Path) -> ProverConfig {
        let dir_cache = Some(workspace_path.join("cache"));
        let path_app_config = workspace_path.join("bundle/openvm.toml");
        let segment_len = Some((1 << 22) - 100);
        match self {
            Phase::EuclidV1 => ProverConfig {
                dir_cache,
                path_app_config,
                segment_len,
                path_app_exe: workspace_path.join("bundle/app_euclidv1.vmexe"),
                ..Default::default()
            },
            Phase::EuclidV2 => ProverConfig {
                dir_cache,
                path_app_config,
                segment_len,
                path_app_exe: workspace_path.join("bundle/app.vmexe"),
                ..Default::default()
            },
        }
    }
}

unsafe impl Send for EuclidHandler {}

impl EuclidHandler {
    pub fn new(workspace_path: &str) -> Self {
        let p = Phase::EuclidV1;
        let workspace_path = Path::new(workspace_path);
        let chunk_prover = ChunkProver::setup(p.phase_spec_chunk(workspace_path))
            .expect("Failed to setup chunk prover");

        let batch_prover = BatchProver::setup(p.phase_spec_batch(workspace_path))
            .expect("Failed to setup batch prover");

        let bundle_prover = BundleProverEuclidV1::setup(p.phase_spec_bundle(workspace_path))
            .expect("Failed to setup bundle prover");

        Self {
            chunk_prover,
            batch_prover,
            bundle_prover,
        }
    }
}

#[async_trait]
impl CircuitsHandler for Arc<Mutex<EuclidHandler>> {
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
