use std::{collections::HashMap, path::Path, sync::OnceLock};

use super::Phase;
use crate::prover::CircuitConfig;
use base64::{prelude::BASE64_STANDARD, Engine};
use scroll_proving_sdk::prover::ProofType;
use scroll_zkvm_prover::Prover;
pub struct AssetsHandler {
    chunk_prover: Prover,
    batch_prover: Prover,
    bundle_prover: Prover,
    cached_vks: HashMap<ProofType, OnceLock<String>>,
}

impl AssetsHandler {
    pub fn new(cfg: &CircuitConfig) -> Self {
        let workspace_path = &cfg.workspace_path;
        let p = Phase::EuclidV2;
        let workspace_path = Path::new(workspace_path);
        let chunk_prover = Prover::setup(p.phase_spec_chunk(workspace_path), false, None)
            .expect("Failed to setup chunk prover");

        let batch_prover = Prover::setup(p.phase_spec_batch(workspace_path), false, None)
            .expect("Failed to setup batch prover");

        let bundle_prover = Prover::setup(p.phase_spec_bundle(workspace_path), true, None)
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

    /// get the inner prover for evm
    pub fn get_evm_prover(&self) -> &Prover {
        &self.bundle_prover
    }

    pub fn get_vk_and_cache(&self, task_type: ProofType) -> String {
        match task_type {
            ProofType::Chunk => self.cached_vks[&ProofType::Chunk]
                .get_or_init(|| BASE64_STANDARD.encode(self.chunk_prover.get_app_vk())),
            ProofType::Batch => self.cached_vks[&ProofType::Batch]
                .get_or_init(|| BASE64_STANDARD.encode(self.batch_prover.get_app_vk())),
            ProofType::Bundle => self.cached_vks[&ProofType::Bundle]
                .get_or_init(|| BASE64_STANDARD.encode(self.bundle_prover.get_app_vk())),
            _ => unreachable!("Unsupported proof type {:?}", task_type),
        }
        .clone()
    }
}
