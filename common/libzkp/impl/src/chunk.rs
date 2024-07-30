use crate::utils::{c_char_to_str, c_char_to_vec, panic_catch};
use libc::c_char;
use prover_v3::{zkevm::Verifier as VerifierLoVersion, ChunkProof as ChunkProofLoVersion};
use prover_v4::{
    utils::init_env_and_log, zkevm::Verifier as VerifierHiVersion,
    ChunkProof as ChunkProofHiVersion,
};
use std::{cell::OnceCell, env};

static mut VERIFIER_LO_VERSION: OnceCell<VerifierLoVersion> = OnceCell::new();
static mut VERIFIER_HI_VERSION: OnceCell<VerifierHiVersion> = OnceCell::new();

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_chunk_verifier(
    params_dir: *const c_char,
    v3_assets_dir: *const c_char,
    v4_assets_dir: *const c_char,
) {
    init_env_and_log("ffi_chunk_verify");

    let params_dir = c_char_to_str(params_dir);
    let v3_assets_dir = c_char_to_str(v3_assets_dir);
    let v4_assets_dir = c_char_to_str(v4_assets_dir);

    // TODO: add a settings in scroll-prover.
    env::set_var("SCROLL_PROVER_ASSETS_DIR", v3_assets_dir);
    let verifier_lo = VerifierLoVersion::from_dirs(params_dir, v3_assets_dir);
    env::set_var("SCROLL_PROVER_ASSETS_DIR", v4_assets_dir);
    let verifier_hi = VerifierHiVersion::from_dirs(params_dir, v4_assets_dir);

    VERIFIER_LO_VERSION.set(verifier_lo).unwrap();
    VERIFIER_HI_VERSION.set(verifier_hi).unwrap();
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn verify_chunk_proof(
    proof: *const c_char,
    fork_name: *const c_char,
) -> c_char {
    let proof = c_char_to_vec(proof);

    let fork_name_str = c_char_to_str(fork_name);
    let fork_id = match fork_name_str {
        "curie" => 3,
        "darwin" => 4,
        _ => {
            log::warn!("unexpected fork_name {fork_name_str}, treated as darwin");
            4
        }
    };
    let verified = panic_catch(|| {
        if fork_id == 3 {
            let proof = serde_json::from_slice::<ChunkProofLoVersion>(proof.as_slice()).unwrap();
            VERIFIER_LO_VERSION.get().unwrap().verify_chunk_proof(proof)
        } else {
            let proof = serde_json::from_slice::<ChunkProofHiVersion>(proof.as_slice()).unwrap();
            VERIFIER_HI_VERSION.get().unwrap().verify_chunk_proof(proof)
        }
    });
    verified.unwrap_or(false) as c_char
}
