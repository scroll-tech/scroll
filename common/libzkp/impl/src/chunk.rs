use crate::utils::{c_char_to_str, c_char_to_vec, vec_to_c_char, OUTPUT_DIR};
use libc::c_char;
use prover::{
    utils::init_env_and_log,
    zkevm::{Prover, Verifier},
    ChunkProof,
};
use std::cell::OnceCell;
use types::eth::BlockTrace;

static mut PROVER: OnceCell<Prover> = OnceCell::new();
static mut VERIFIER: OnceCell<Verifier> = OnceCell::new();

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_chunk_prover(params_dir: *const c_char) {
    init_env_and_log("ffi_chunk_prove");

    let params_dir = c_char_to_str(params_dir);
    let prover = Prover::from_params_dir(params_dir);

    PROVER.set(prover).unwrap();
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_chunk_verifier(params_dir: *const c_char, assets_dir: *const c_char) {
    init_env_and_log("ffi_chunk_verify");

    let params_dir = c_char_to_str(params_dir);
    let assets_dir = c_char_to_str(assets_dir);

    let verifier = Verifier::from_dirs(params_dir, assets_dir);

    VERIFIER.set(verifier).unwrap();
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn gen_chunk_proof(block_traces: *const c_char) -> *const c_char {
    let block_traces = c_char_to_vec(block_traces);
    let block_traces = serde_json::from_slice::<Vec<BlockTrace>>(&block_traces).unwrap();

    let proof = panic::catch_unwind(|| {
        PROVER
            .get_mut()
            .unwrap()
            .gen_chunk_proof(block_traces, None, OUTPUT_DIR.as_deref())
            .unwrap();

        serde_json::to_vec(&proof).unwrap()
    });

    proof.map_or(null(), vec_to_c_char)
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn verify_chunk_proof(proof: *const c_char) -> c_char {
    let proof = c_char_to_vec(proof);
    let proof = serde_json::from_slice::<ChunkProof>(proof.as_slice()).unwrap();

    let verified = panic::catch_unwind(|| VERIFIER.get().unwrap().verify_chunk_proof(proof));
    verified.unwrap_or(false) as c_char
}
