use crate::utils::{c_char_to_str, c_char_to_vec, string_to_c_char, vec_to_c_char, OUTPUT_DIR};
use crate::types::ProofResult;
use libc::c_char;
use prover::{
    utils::init_env_and_log,
    zkevm::{Prover, Verifier},
    ChunkProof,
};
use std::{cell::OnceCell, env, panic, ptr::null};
use types::eth::BlockTrace;

static mut PROVER: OnceCell<Prover> = OnceCell::new();
static mut VERIFIER: OnceCell<Verifier> = OnceCell::new();

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_chunk_prover(params_dir: *const c_char, assets_dir: *const c_char) {
    init_env_and_log("ffi_chunk_prove");

    let params_dir = c_char_to_str(params_dir);
    let assets_dir = c_char_to_str(assets_dir);

    // TODO: add a settings in scroll-prover.
    env::set_var("SCROLL_PROVER_ASSETS_DIR", assets_dir);
    let prover = Prover::from_params_dir(params_dir);

    PROVER.set(prover).unwrap();
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_chunk_verifier(params_dir: *const c_char, assets_dir: *const c_char) {
    init_env_and_log("ffi_chunk_verify");

    let params_dir = c_char_to_str(params_dir);
    let assets_dir = c_char_to_str(assets_dir);

    // TODO: add a settings in scroll-prover.
    env::set_var("SCROLL_PROVER_ASSETS_DIR", assets_dir);
    let verifier = Verifier::from_dirs(params_dir, assets_dir);

    VERIFIER.set(verifier).unwrap();
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn get_chunk_vk() -> *const c_char {
    let vk_result = panic::catch_unwind(|| PROVER.get_mut().unwrap().get_vk());

    vk_result
        .ok()
        .flatten()
        .map_or(null(), |vk| string_to_c_char(base64::encode(vk)))
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn gen_chunk_proof(block_traces: *const c_char) -> *const c_char {
    let proof_result: Result<Vec<u8>, String> = panic::catch_unwind(|| {
        let block_traces = c_char_to_vec(block_traces);
        let block_traces = serde_json::from_slice::<Vec<BlockTrace>>(&block_traces)
            .map_err(|e| format!("Failed to deserialize block traces: {:?}", e))?;

        let proof = PROVER
            .get_mut()
            .map_err(|_| "Failed to get mutable reference to PROVER.".to_string())?
            .gen_chunk_proof(block_traces, None, OUTPUT_DIR.as_deref())
            .map_err(|e| format!("Proof generation failed: {:?}", e))?;

        serde_json::to_vec(&proof)
            .map_err(|e| format!("Failed to serialize the proof: {:?}", e))
    }).unwrap_or_else(|err| Err(format!("Unwind error: {:?}", err)));

    let r = match proof_result {
        Ok(proof_bytes) => ProofResult {
            message: Some(proof_bytes),
            error: None,
        },
        Err(err) => ProofResult {
            message: None,
            error: Some(err),
        },
    };

    serde_json::to_vec(&r).map_or(null(), vec_to_c_char)
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn verify_chunk_proof(proof: *const c_char) -> c_char {
    let proof = c_char_to_vec(proof);
    let proof = serde_json::from_slice::<ChunkProof>(proof.as_slice()).unwrap();

    let verified = panic::catch_unwind(|| VERIFIER.get().unwrap().verify_chunk_proof(proof));
    verified.unwrap_or(false) as c_char
}
