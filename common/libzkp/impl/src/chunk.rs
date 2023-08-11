use crate::utils::{c_char_to_str, c_char_to_vec, vec_to_c_char, OUTPUT_DIR};
use libc::c_char;
use prover::{
    utils::init_env_and_log,
    zkevm::{Prover, Verifier},
    ChunkProof,
};
use std::{cell::OnceCell, panic, ptr::null};
use types::eth::BlockTrace;

static mut PROVER: OnceCell<Prover> = OnceCell::new();
static mut VERIFIER: OnceCell<Verifier> = OnceCell::new();

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_chunk_prover(params_dir: *const c_char) {
    init_env_and_log("ffi_chunk_prove");

    let params_dir = c_char_to_str(params_dir);

    let result = panic::catch_unwind(|| Prover::from_params_dir(params_dir));
    match result {
        Ok(prover) => PROVER.set(prover).unwrap(),
        Err(err) => log::error!("Failed to init chunk-prover: {err:?}"),
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_chunk_verifier(params_dir: *const c_char, assets_dir: *const c_char) {
    init_env_and_log("ffi_chunk_verify");

    let params_dir = c_char_to_str(params_dir);
    let assets_dir = c_char_to_str(assets_dir);

    let result = panic::catch_unwind(|| Verifier::from_dirs(params_dir, assets_dir));
    match result {
        Ok(verifier) => VERIFIER.set(verifier).unwrap(),
        Err(err) => log::error!("Failed to init chunk-verifier: {err:?}"),
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn gen_chunk_proof(block_traces: *const c_char) -> *const c_char {
    let block_traces = c_char_to_vec(block_traces);
    let block_traces = serde_json::from_slice::<Vec<BlockTrace>>(&block_traces).unwrap();

    let result = panic::catch_unwind(|| {
        let proof = PROVER
            .get_mut()
            .unwrap()
            .gen_chunk_proof(block_traces, None, OUTPUT_DIR.as_deref())
            .unwrap();

        serde_json::to_vec(&proof).unwrap()
    });

    match result {
        Ok(result) => vec_to_c_char(result),
        Err(err) => {
            log::error!("Failed to gen chunk-proof: {err:?}");
            null()
        }
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn verify_chunk_proof(proof: *const c_char) -> c_char {
    let proof = c_char_to_vec(proof);
    let proof = serde_json::from_slice::<ChunkProof>(proof.as_slice()).unwrap();

    let result = panic::catch_unwind(|| VERIFIER.get().unwrap().verify_chunk_proof(proof));
    let verified = match result {
        Ok(verified) => verified,
        Err(err) => {
            log::error!("Failed to verify chunk-proof: {err:?}");
            false
        }
    };

    verified as c_char
}
