mod utils;
mod verifier;

use crate::utils::{c_char_to_str, c_char_to_vec};
use libc::c_char;
use prover_v5::utils::init_env_and_log;
use verifier::{TaskType, VerifierConfig};

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init(config: *const c_char) {
    init_env_and_log("ffi_init");

    let config_str = c_char_to_str(config);
    let verifier_config = serde_json::from_str::<VerifierConfig>(config_str).unwrap();
    verifier::init(verifier_config);
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn verify_chunk_proof(
    proof: *const c_char,
    fork_name: *const c_char,
) -> c_char {
    verify_proof(proof, fork_name, TaskType::Chunk)
}

fn verify_proof(proof: *const c_char, fork_name: *const c_char, task_type: TaskType) -> c_char {
    let proof = c_char_to_vec(proof);

    let fork_name_str = c_char_to_str(fork_name);
    let verifier = verifier::get_verifier(fork_name_str);

    if let Err(e) = verifier {
        log::warn!("failed to get verifier, error: {:#}", e);
        return 0 as c_char;
    }
    match verifier.unwrap().verify(task_type, proof) {
        Err(e) => {
            log::error!("{:?} verify failed, error: {:#}", task_type, e);
            false as c_char
        }
        Ok(result) => result as c_char,
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn verify_batch_proof(
    proof: *const c_char,
    fork_name: *const c_char,
) -> c_char {
    verify_proof(proof, fork_name, TaskType::Batch)
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn verify_bundle_proof(
    proof: *const c_char,
    fork_name: *const c_char,
) -> c_char {
    verify_proof(proof, fork_name, TaskType::Bundle)
}
