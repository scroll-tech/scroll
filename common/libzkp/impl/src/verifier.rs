mod darwin;
mod darwin_v2;

use anyhow::{bail, Result};
use darwin::DarwinVerifier;
use darwin_v2::DarwinV2Verifier;
use serde::{Deserialize, Serialize};
use std::{cell::OnceCell, rc::Rc};

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum TaskType {
    Chunk,
    Batch,
    Bundle,
}

pub trait ProofVerifier {
    fn verify(&self, task_type: TaskType, proof: Vec<u8>) -> Result<bool>;
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

static mut VERIFIER_HIGH: OnceCell<VerifierPair> = OnceCell::new();
static mut VERIFIER_LOW: OnceCell<VerifierPair> = OnceCell::new();

pub fn init(config: VerifierConfig) {
    let low_conf = config.low_version_circuit;
    let verifier = DarwinVerifier::new(&low_conf.params_path, &low_conf.assets_path);

    unsafe {
        VERIFIER_LOW
            .set(VerifierPair(
                low_conf.fork_name,
                Rc::new(Box::new(verifier)),
            ))
            .unwrap_unchecked();
    }
    let high_conf = config.high_version_circuit;
    let verifier = DarwinV2Verifier::new(&high_conf.params_path, &high_conf.assets_path);
    unsafe {
        VERIFIER_HIGH
            .set(VerifierPair(
                high_conf.fork_name,
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
