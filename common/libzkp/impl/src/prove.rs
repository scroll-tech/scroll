use crate::utils::{c_char_to_str, c_char_to_vec, vec_to_c_char};
use libc::c_char;
use std::cell::OnceCell;
use types::eth::BlockTrace;
use zkevm::circuit::AGG_DEGREE;
use zkevm::utils::{load_or_create_params, load_or_create_seed};
use zkevm::{circuit::DEGREE, prover::Prover};

static mut PROVER: OnceCell<Prover> = OnceCell::new();

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_prover(params_path: *const c_char, seed_path: *const c_char) {
    env_logger::init();

    let params_path = c_char_to_str(params_path);
    let seed_path = c_char_to_str(seed_path);
    let params = load_or_create_params(params_path, *DEGREE).unwrap();
    let agg_params = load_or_create_params(params_path, *AGG_DEGREE).unwrap();
    let seed = load_or_create_seed(seed_path).unwrap();
    let p = Prover::from_params_and_seed(params, agg_params, seed);
    PROVER.set(p).unwrap();
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn create_agg_proof(trace_char: *const c_char) -> *const c_char {
    env_logger::init();

    let trace_vec = c_char_to_vec(trace_char);
    let trace = serde_json::from_slice::<BlockTrace>(&trace_vec).unwrap();
    let proof = PROVER
        .get_mut()
        .unwrap()
        .create_agg_circuit_proof(&trace)
        .unwrap();
    let proof_bytes = serde_json::to_vec(&proof).unwrap();
    vec_to_c_char(proof_bytes)
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn create_agg_proof_multi(trace_char: *const c_char) -> *const c_char {
    env_logger::init();

    let trace_vec = c_char_to_vec(trace_char);
    let traces = serde_json::from_slice::<Vec<BlockTrace>>(&trace_vec).unwrap();
    let proof = PROVER
        .get_mut()
        .unwrap()
        .create_agg_circuit_proof_multi(traces.as_slice())
        .unwrap();
    let proof_bytes = serde_json::to_vec(&proof).unwrap();
    vec_to_c_char(proof_bytes)
}
