mod utils;
mod verifier;

use std::{
    ffi::{c_char, c_int, CString},
    path::Path,
};

use crate::utils::{c_char_to_str, c_char_to_vec};
use verifier::{TaskType, VerifierConfig};

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_verifier(config: *const c_char) {
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
    let fork_name_str = c_char_to_str(fork_name);
    let proof = c_char_to_vec(proof);
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

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn dump_vk(fork_name: *const c_char, file: *const c_char) {
    _dump_vk(fork_name, file);
}

fn _dump_vk(fork_name: *const c_char, file: *const c_char) {
    let fork_name_str = c_char_to_str(fork_name);
    let verifier = verifier::get_verifier(fork_name_str);

    if let Ok(verifier) = verifier {
        verifier.as_ref().dump_vk(Path::new(c_char_to_str(file)));
    }
}

/// Represents the result of generating a universal task
#[repr(C)]
pub struct HandlingResult {
    ok: c_char,
    universal_task: *mut c_char,
    metadata: *mut c_char,
    expected_pi_hash: [c_char; 32],
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn gen_universal_task(
    _task_type: c_int,
    _task: *const c_char,
    _fork_name: *const c_char,
) -> HandlingResult {
    unimplemented!("next phase");
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn release_task_result(result: HandlingResult) {
    // Free the allocated strings
    if !result.universal_task.is_null() {
        let _ = CString::from_raw(result.universal_task);
    }

    if !result.metadata.is_null() {
        let _ = CString::from_raw(result.metadata);
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn gen_wrapped_proof(
    _proof_json: *const c_char,
    _metadata: *const c_char,
    _vk: *const c_char,
    _vk_len: usize,
) -> *mut c_char {
    unimplemented!("next phase");
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn release_string(string_ptr: *mut c_char) {
    if !string_ptr.is_null() {
        let _ = CString::from_raw(string_ptr);
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_l2geth(_config: *const c_char) {
    unimplemented!("next phase");
}
