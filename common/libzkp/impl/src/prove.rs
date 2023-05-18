use crate::utils::{c_char_to_str, c_char_to_vec, vec_to_c_char};
use halo2_proofs::SerdeFormat;
use libc::c_char;
use std::cell::OnceCell;
use std::panic;
use std::ptr::null;
use types::eth::BlockTrace;
use zkevm::circuit::AGG_DEGREE;
use zkevm::utils::{load_params, load_seed};
use zkevm::{circuit::DEGREE, prover::Prover};

static mut PROVER: OnceCell<Prover> = OnceCell::new();

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_prover(params_path: *const c_char, seed_path: *const c_char) {
    env_logger::init();

    let params_path = c_char_to_str(params_path);
    let seed_path = c_char_to_str(seed_path);
    let params = load_params(params_path, *DEGREE, SerdeFormat::RawBytesUnchecked).unwrap();
    let agg_params = load_params(params_path, *AGG_DEGREE, SerdeFormat::RawBytesUnchecked).unwrap();
    let seed = load_seed(seed_path).unwrap();
    let p = Prover::from_params_and_seed(params, agg_params, seed);
    PROVER.set(p).unwrap();
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn create_agg_proof(trace_char: *const c_char) -> *const c_char {
    let trace_vec = c_char_to_vec(trace_char);
    let trace = serde_json::from_slice::<BlockTrace>(&trace_vec).unwrap();
    let proof_result = panic::catch_unwind(|| {
        let proof = PROVER
            .get_mut()
            .unwrap()
            .create_agg_circuit_proof(&trace)
            .unwrap();
        serde_json::to_vec(&proof).unwrap()
    });
    proof_result.map_or(null(), vec_to_c_char)
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn create_agg_proof_multi(trace_char: *const c_char) -> *const c_char {
    let trace_vec = c_char_to_vec(trace_char);
    let traces = serde_json::from_slice::<Vec<BlockTrace>>(&trace_vec).unwrap();
    let proof_result = panic::catch_unwind(|| {
        let proof = PROVER
            .get_mut()
            .unwrap()
            .create_agg_circuit_proof_batch(traces.as_slice())
            .unwrap();
        serde_json::to_vec(&proof).unwrap()
    });
    proof_result.map_or(null(), vec_to_c_char)
}
