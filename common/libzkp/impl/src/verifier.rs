#![allow(static_mut_refs)]

mod euclid;
mod euclidv2;

use anyhow::{bail, Result};
use euclid::EuclidVerifier;
use euclidv2::EuclidV2Verifier;
use serde::{Deserialize, Serialize};
use std::{cell::OnceCell, path::Path, rc::Rc};

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum TaskType {
    Chunk,
    Batch,
    Bundle,
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
    pub params_path: String,
    pub assets_path: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct VerifierConfig {
    pub low_version_circuit: CircuitConfig,
    pub high_version_circuit: CircuitConfig,
}

type HardForkName = String;

struct VerifierPair(HardForkName, Rc<Box<dyn ProofVerifier>>);

static mut VERIFIER_LOW: OnceCell<VerifierPair> = OnceCell::new();
static mut VERIFIER_HIGH: OnceCell<VerifierPair> = OnceCell::new();

pub fn init(config: VerifierConfig) {
    let verifier = EuclidVerifier::new(&config.high_version_circuit.assets_path);
    unsafe {
        VERIFIER_LOW
            .set(VerifierPair(
                "euclid".to_string(),
                Rc::new(Box::new(verifier)),
            ))
            .unwrap_unchecked();
    }

    let verifier = EuclidV2Verifier::new(&config.high_version_circuit.assets_path);
    unsafe {
        VERIFIER_HIGH
            .set(VerifierPair(
                "euclidV2".to_string(),
                Rc::new(Box::new(verifier)),
            ))
            .unwrap_unchecked();
    }
}

pub fn get_verifier(fork_name: &str) -> Result<Rc<Box<dyn ProofVerifier>>> {
    unsafe {
        if let Some(verifier) = VERIFIER_LOW.get() {
            if verifier.0 == fork_name {
                return Ok(verifier.1.clone());
            }
        }

        if let Some(verifier) = VERIFIER_HIGH.get() {
            if verifier.0 == fork_name {
                return Ok(verifier.1.clone());
            }
        }
    }
    bail!("failed to get verifier, key not found, {}", fork_name)
}
