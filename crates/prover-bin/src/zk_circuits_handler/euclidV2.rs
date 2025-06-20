use std::{path::Path, sync::Arc};

use super::{CircuitsHandler, Phase};
use async_trait::async_trait;
use eyre::Result;
use scroll_proving_sdk::prover::{proving_service::ProveRequest, ProofType};
use scroll_zkvm_prover_euclid::{BatchProver, BundleProverEuclidV2, ChunkProver};
use scroll_zkvm_types::ProvingTask;
use tokio::sync::Mutex;
pub struct EuclidV2Handler {
    chunk_prover: ChunkProver,
    batch_prover: BatchProver,
    bundle_prover: BundleProverEuclidV2,
}

unsafe impl Send for EuclidV2Handler {}

impl EuclidV2Handler {
    pub fn new(workspace_path: &str) -> Self {
        let p = Phase::EuclidV2;
        let workspace_path = Path::new(workspace_path);
        let chunk_prover = ChunkProver::setup(p.phase_spec_chunk(workspace_path))
            .expect("Failed to setup chunk prover");

        let batch_prover = BatchProver::setup(p.phase_spec_batch(workspace_path))
            .expect("Failed to setup batch prover");

        let bundle_prover = BundleProverEuclidV2::setup(p.phase_spec_bundle(workspace_path))
            .expect("Failed to setup bundle prover");

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
        let u_task: ProvingTask = serde_json::from_str(&prove_request.input)?;

        let proof = match prove_request.proof_type {
            ProofType::Chunk => self
                .try_lock()
                .unwrap()
                .chunk_prover
                .gen_proof_universal(&u_task, false)?,
            ProofType::Batch => self
                .try_lock()
                .unwrap()
                .batch_prover
                .gen_proof_universal(&u_task, false)?,
            ProofType::Bundle => self
                .try_lock()
                .unwrap()
                .bundle_prover
                .gen_proof_universal(&u_task, true)?,
            _ => return Err(eyre::eyre!("Unsupported proof type")),
        };
        Ok(serde_json::to_string(&proof)?)
    }
}
