use std::{
    collections::HashMap,
    path::Path,
    sync::{Arc, OnceLock},
};

use super::{CircuitsHandler, Phase};
use crate::prover::CircuitConfig;
use async_trait::async_trait;
use base64::{prelude::BASE64_STANDARD, Engine};
use eyre::Result;
use scroll_proving_sdk::prover::{proving_service::ProveRequest, ProofType};
use scroll_zkvm_prover_euclid::{BatchProver, BundleProverEuclidV2, ChunkProver};
use scroll_zkvm_types::ProvingTask;
use tokio::sync::Mutex;
pub struct EuclidV2Handler {
    chunk_prover: ChunkProver,
    batch_prover: BatchProver,
    bundle_prover: BundleProverEuclidV2,
    cached_vks: HashMap<ProofType, OnceLock<String>>,
}

unsafe impl Send for EuclidV2Handler {}

impl EuclidV2Handler {
    pub fn new(cfg: &CircuitConfig) -> Self {
        let workspace_path = &cfg.workspace_path;
        let p = Phase::EuclidV2;
        let workspace_path = Path::new(workspace_path);
        let chunk_prover = ChunkProver::setup(p.phase_spec_chunk(workspace_path))
            .expect("Failed to setup chunk prover");

        let batch_prover = BatchProver::setup(p.phase_spec_batch(workspace_path))
            .expect("Failed to setup batch prover");

        let bundle_prover = BundleProverEuclidV2::setup(p.phase_spec_bundle(workspace_path))
            .expect("Failed to setup bundle prover");

        let build_vk_cache = |proof_type: ProofType| {
            let vk = if let Some(vk) = cfg.vks.get(&proof_type) {
                OnceLock::from(vk.clone())
            } else {
                OnceLock::new()
            };
            (proof_type, vk)
        };

        Self {
            chunk_prover,
            batch_prover,
            bundle_prover,
            cached_vks: HashMap::from([
                build_vk_cache(ProofType::Chunk),
                build_vk_cache(ProofType::Batch),
                build_vk_cache(ProofType::Bundle),
            ]),
        }
    }

    pub fn get_vk_and_cache(&self, task_type: ProofType) -> String {
        match task_type {
            ProofType::Chunk => self.cached_vks[&ProofType::Chunk]
                .get_or_init(|| BASE64_STANDARD.encode(self.chunk_prover.get_app_vk())),
            ProofType::Batch => self.cached_vks[&ProofType::Batch]
                .get_or_init(|| BASE64_STANDARD.encode(self.batch_prover.get_app_vk())),
            ProofType::Bundle => self.cached_vks[&ProofType::Bundle]
                .get_or_init(|| BASE64_STANDARD.encode(self.bundle_prover.get_evm_vk())),
            _ => unreachable!("Unsupported proof type {:?}", task_type),
        }
        .clone()
    }
}

#[async_trait]
impl CircuitsHandler for Arc<Mutex<EuclidV2Handler>> {
    fn get_vk(&self, task_type: ProofType) -> String {
        self.try_lock()
            .expect("get vk is on called before other entry is used")
            .get_vk_and_cache(task_type)
    }

    async fn get_proof_data(&self, prove_request: ProveRequest) -> Result<String> {
        let handler_self = self.lock().await;
        let u_task: ProvingTask = serde_json::from_str(&prove_request.input)?;
        let expected_vk = handler_self.get_vk_and_cache(prove_request.proof_type);
        if BASE64_STANDARD.encode(&u_task.vk) != expected_vk {
            eyre::bail!(
                "vk is not match!, prove type {:?}, expected {}, get {}",
                prove_request.proof_type,
                expected_vk,
                BASE64_STANDARD.encode(&u_task.vk),
            );
        }

        let proof = match prove_request.proof_type {
            ProofType::Chunk => handler_self
                .chunk_prover
                .gen_proof_universal(&u_task, false)?,
            ProofType::Batch => handler_self
                .batch_prover
                .gen_proof_universal(&u_task, false)?,
            ProofType::Bundle => handler_self
                .bundle_prover
                .gen_proof_universal(&u_task, true)?,
            _ => {
                return Err(eyre::eyre!(
                    "Unsupported proof type {:?}",
                    prove_request.proof_type
                ))
            }
        };
        //TODO: check expected PI
        Ok(serde_json::to_string(&proof)?)
    }
}
