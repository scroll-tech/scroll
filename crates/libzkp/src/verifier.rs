#![allow(static_mut_refs)]

mod euclidv2;
use euclidv2::EuclidV2Verifier;
use eyre::Result;
use serde::{Deserialize, Serialize};
use std::{
    path::Path,
    sync::{Arc, Mutex, OnceLock},
};

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
    fn verify(&self, task_type: TaskType, proof: Vec<u8>) -> Result<bool>;
    fn dump_vk(&self, file: &Path);
}

#[derive(Debug, Serialize, Deserialize)]
pub struct CircuitConfig {
    pub fork_name: String,
    pub assets_path: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct VerifierConfig {
    pub high_version_circuit: CircuitConfig,
}

type HardForkName = String;

struct VerifierPair(HardForkName, Arc<Mutex<dyn ProofVerifier + Send>>);
static VERIFIER_HIGH: OnceLock<VerifierPair> = OnceLock::new();

pub fn init(config: VerifierConfig) {
    let verifier = EuclidV2Verifier::new(&config.high_version_circuit.assets_path);

    let ret = VERIFIER_HIGH
        .set(VerifierPair(
            config.high_version_circuit.fork_name.to_lowercase(),
            Arc::new(Mutex::new(verifier)),
        ))
        .is_ok();

    assert!(ret);
}

pub fn get_verifier(fork_name: &str) -> Result<Arc<Mutex<dyn ProofVerifier>>> {
    if let Some(verifier) = VERIFIER_HIGH.get() {
        if verifier.0 == fork_name {
            return Ok(verifier.1.clone());
        }
        Err(eyre::eyre!(
            "failed to get verifier, key not found: {}, expected {}",
            fork_name,
            verifier.0,
        ))
    } else {
        Err(eyre::eyre!(
            "failed to get verifier, not inited {}",
            fork_name
        ))
    }
}
