#![allow(static_mut_refs)]

mod darwin_v2;
mod euclid;

use anyhow::{bail, Result};
use darwin_v2::DarwinV2Verifier;
use euclid::EuclidVerifier;
use halo2_proofs::{halo2curves::bn256::Bn256, poly::kzg::commitment::ParamsKZG};
use prover_v5::utils::load_params;
use serde::{Deserialize, Serialize};
use std::{cell::OnceCell, collections::BTreeMap, path::Path, rc::Rc};

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

static mut VERIFIER_ZKVM: OnceCell<VerifierPair> = OnceCell::new();
static mut VERIFIER_HALO2: OnceCell<VerifierPair> = OnceCell::new();
static mut PARAMS_MAP: OnceCell<BTreeMap<u32, ParamsKZG<Bn256>>> = OnceCell::new();

pub fn init(config: VerifierConfig) {
    std::env::set_var(
        "SCROLL_PROVER_ASSETS_DIR",
        &config.low_version_circuit.assets_path,
    );
    let params_degrees = [
        *prover_v5::config::LAYER2_DEGREE,
        *prover_v5::config::LAYER4_DEGREE,
    ];

    // params should be shared between low and high
    let mut params_map = BTreeMap::new();
    for degree in params_degrees {
        if let std::collections::btree_map::Entry::Vacant(e) = params_map.entry(degree) {
            match load_params(&config.low_version_circuit.params_path, degree, None) {
                Ok(params) => {
                    e.insert(params);
                }
                Err(e) => panic!(
                    "failed to load params, degree {}, dir {}, err {}",
                    degree, config.low_version_circuit.params_path, e
                ),
            }
        }
    }
    unsafe {
        PARAMS_MAP.set(params_map).unwrap_unchecked();
    }

    let verifier = DarwinV2Verifier::new(
        unsafe { PARAMS_MAP.get().unwrap() },
        &config.low_version_circuit.assets_path,
    );
    unsafe {
        VERIFIER_HALO2
            .set(VerifierPair(
                config.low_version_circuit.fork_name.clone(),
                Rc::new(Box::new(verifier)),
            ))
            .unwrap_unchecked();
    }

    let verifier = EuclidVerifier::new(
        unsafe { PARAMS_MAP.get().unwrap() },
        &config.high_version_circuit.assets_path,
    );
    unsafe {
        VERIFIER_ZKVM
            .set(VerifierPair(
                config.high_version_circuit.fork_name,
                Rc::new(Box::new(verifier)),
            ))
            .unwrap_unchecked();
    }
}

pub fn get_verifier(fork_name: &str) -> Result<Rc<Box<dyn ProofVerifier>>> {
    unsafe {
        if let Some(verifier) = VERIFIER_HALO2.get() {
            if verifier.0 == fork_name {
                return Ok(verifier.1.clone());
            }
        }

        if let Some(verifier) = VERIFIER_ZKVM.get() {
            if verifier.0 == fork_name {
                return Ok(verifier.1.clone());
            }
        }
    }
    bail!("failed to get verifier, key not found, {}", fork_name)
}
