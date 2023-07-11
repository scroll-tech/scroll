use crate::utils::{c_char_to_str, c_char_to_vec};
use halo2_proofs::SerdeFormat;
use libc::c_char;
use std::fs::File;
use std::io::Read;
use std::panic;
use zkevm::circuit::{AGG_DEGREE, DEGREE};
use zkevm::prover::AggCircuitProof;
use zkevm::utils::load_params;
use zkevm::verifier::Verifier;

static mut VERIFIER: Option<&Verifier> = None;

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_verifier(params_path: *const c_char, agg_vk_path: *const c_char) {
    env_logger::init();

    let params_path = c_char_to_str(params_path);
    let agg_vk_path = c_char_to_str(agg_vk_path);
    let mut f = File::open(agg_vk_path).unwrap();
    let mut agg_vk = vec![];
    f.read_to_end(&mut agg_vk).unwrap();

    let params = load_params(params_path, *DEGREE, SerdeFormat::RawBytesUnchecked).unwrap();
    let agg_params = load_params(params_path, *AGG_DEGREE, SerdeFormat::RawBytesUnchecked).unwrap();

    let v = Box::new(Verifier::from_params(params, agg_params, Some(agg_vk)));
    VERIFIER = Some(Box::leak(v))
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn verify_agg_proof(proof: *const c_char) -> c_char {
    let proof_vec = c_char_to_vec(proof);
    let agg_proof = serde_json::from_slice::<AggCircuitProof>(proof_vec.as_slice()).unwrap();
    let verified = panic::catch_unwind(|| {
        VERIFIER
            .unwrap()
            .verify_agg_circuit_proof(agg_proof)
            .is_ok()
    });
    verified.unwrap_or(false) as c_char
}
