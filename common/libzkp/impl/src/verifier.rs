mod darwin;
mod darwin_v2;

use anyhow::{bail, Result};
use darwin::DarwinVerifier;
use darwin_v2::DarwinV2Verifier;
use halo2_proofs::{halo2curves::bn256::Bn256, poly::kzg::commitment::ParamsKZG};
use prover_v4::utils::load_params;
use serde::{Deserialize, Serialize};
use std::{cell::OnceCell, collections::BTreeMap, rc::Rc};

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
static mut PARAMS_MAP: OnceCell<BTreeMap<u32, ParamsKZG<Bn256>>> = OnceCell::new();

pub fn init(config: VerifierConfig) {
    let low_conf = config.low_version_circuit;

    std::env::set_var("SCROLL_PROVER_ASSETS_DIR", &low_conf.assets_path);
    let params_degrees = [
        *prover_v4::config::LAYER2_DEGREE,
        *prover_v4::config::LAYER4_DEGREE,
    ];

    // params should be shared between low and high
    let mut params_map = BTreeMap::new();
    for degree in params_degrees {
        if let std::collections::btree_map::Entry::Vacant(e) = params_map.entry(degree) {
            match load_params(&low_conf.params_path, degree, None) {
                Ok(params) => {
                    e.insert(params);
                }
                Err(e) => panic!(
                    "failed to load params, degree {}, dir {}, err {}",
                    degree, low_conf.params_path, e
                ),
            }
        }
    }
    unsafe {
        PARAMS_MAP.set(params_map).unwrap_unchecked();
    }

    let verifier = DarwinVerifier::new(unsafe { PARAMS_MAP.get().unwrap() }, &low_conf.assets_path);

    unsafe {
        VERIFIER_LOW
            .set(VerifierPair(
                low_conf.fork_name,
                Rc::new(Box::new(verifier)),
            ))
            .unwrap_unchecked();
    }
    let high_conf = config.high_version_circuit;
    let verifier =
        DarwinV2Verifier::new(unsafe { PARAMS_MAP.get().unwrap() }, &high_conf.assets_path);
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
