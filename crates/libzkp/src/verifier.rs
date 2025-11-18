#![allow(static_mut_refs)]

mod universal;
use eyre::Result;
use serde::{Deserialize, Serialize};
use std::{
    collections::HashMap,
    path::Path,
    sync::{Arc, Mutex, OnceLock},
};
use universal::Verifier;

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum TaskType {
    Chunk,
    Batch,
    Bundle,
}

impl std::fmt::Display for TaskType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Chunk => write!(f, "chunk"),
            Self::Batch => write!(f, "batch"),
            Self::Bundle => write!(f, "bundle"),
        }
    }
}

#[derive(Debug, Serialize, Deserialize)]
pub struct VKDump {
    pub chunk_vk: String,
    pub batch_vk: String,
    pub bundle_vk: String,
}

pub trait ProofVerifier {
    fn verify(&self, task_type: TaskType, proof: &[u8]) -> Result<bool>;
    fn dump_vk(&self, file: &Path);
}

#[derive(Debug, Serialize, Deserialize)]
pub struct CircuitConfig {
    pub version: u8,
    pub fork_name: String,
    pub assets_path: String,
    #[serde(default)]
    pub features: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct VerifierConfig {
    pub circuits: Vec<CircuitConfig>,
}

pub(crate) type HardForkName = String;

type VerifierType = Arc<Mutex<dyn ProofVerifier + Send>>;
static VERIFIERS: OnceLock<HashMap<HardForkName, VerifierType>> = OnceLock::new();

pub fn init(config: VerifierConfig) {
    let mut verifiers: HashMap<HardForkName, VerifierType> = Default::default();

    for cfg in &config.circuits {
        let canonical_fork_name = cfg.fork_name.to_lowercase();

        let verifier = Verifier::new(&cfg.assets_path, cfg.version);
        let ret = verifiers.insert(canonical_fork_name, Arc::new(Mutex::new(verifier)));
        assert!(
            ret.is_none(),
            "DO NOT init the same fork {} twice",
            cfg.fork_name
        );
        tracing::info!(
            "load verifier config for fork {} (ver {})",
            cfg.fork_name,
            cfg.version
        );
    }

    let ret = VERIFIERS.set(verifiers).is_ok();
    assert!(ret);
}

pub fn get_verifier(fork_name: &str) -> Result<Arc<Mutex<dyn ProofVerifier>>> {
    if let Some(verifiers) = VERIFIERS.get() {
        if let Some(verifier) = verifiers.get(fork_name) {
            return Ok(verifier.clone());
        }

        Err(eyre::eyre!(
            "failed to get verifier, key not found: {}, has {:?}",
            fork_name,
            verifiers.keys().collect::<Vec<_>>(),
        ))
    } else {
        Err(eyre::eyre!(
            "failed to get verifier, not inited {}",
            fork_name
        ))
    }
}
