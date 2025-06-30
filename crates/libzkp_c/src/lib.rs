mod utils;

use std::ffi::{c_char, CString};

use libzkp::TaskType;
use utils::{c_char_to_str, c_char_to_vec};

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_verifier(config: *const c_char) {
    let config_str = c_char_to_str(config);
    libzkp::verifier_init(config_str).unwrap();
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_l2geth(config: *const c_char) {
    let config_str = c_char_to_str(config);
    l2geth::init(config_str).unwrap();
}

fn verify_proof(proof: *const c_char, fork_name: *const c_char, task_type: TaskType) -> c_char {
    let fork_name_str = c_char_to_str(fork_name);
    let proof = c_char_to_vec(proof);

    match libzkp::verify_proof(proof, fork_name_str, task_type) {
        Err(e) => {
            tracing::error!("{:?} verify failed, error: {:#}", task_type, e);
            false as c_char
        }
        Ok(result) => result as c_char,
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn verify_chunk_proof(
    proof: *const c_char,
    fork_name: *const c_char,
) -> c_char {
    verify_proof(proof, fork_name, TaskType::Chunk)
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
    let fork_name_str = c_char_to_str(fork_name);
    let file_str = c_char_to_str(file);
    libzkp::dump_vk(fork_name_str, file_str).unwrap();
}

// Define a struct to hold handling results
#[repr(C)]
pub struct HandlingResult {
    ok: c_char,
    universal_task: *mut c_char,
    metadata: *mut c_char,
    expected_pi_hash: [c_char; 32],
}

fn failed_handling_result() -> HandlingResult {
    HandlingResult {
        ok: false as c_char,
        universal_task: std::ptr::null_mut(),
        metadata: std::ptr::null_mut(),
        expected_pi_hash: Default::default(),
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn gen_universal_task(
    task_type: i32,
    task: *const c_char,
    fork_name: *const c_char,
    expected_vk: *const u8,
    expected_vk_len: usize,
) -> HandlingResult {
    let mut interpreter = None;
    let task_json = if task_type == TaskType::Chunk as i32 {
        let pre_task_str = c_char_to_str(task);
        let cli = l2geth::get_client();
        match libzkp::checkout_chunk_task(pre_task_str, cli) {
            Ok(str) => {
                interpreter.replace(cli);
                str
            }
            Err(e) => {
                tracing::error!("gen_universal_task failed at pre interpret step, error: {e}");
                return failed_handling_result();
            }
        }
    } else {
        c_char_to_str(task).to_string()
    };

    let expected_vk = if expected_vk_len > 0 {
        std::slice::from_raw_parts(expected_vk, expected_vk_len)
    } else {
        &[]
    };

    let ret = libzkp::gen_universal_task(
        task_type,
        &task_json,
        c_char_to_str(fork_name),
        expected_vk,
        interpreter,
    );

    if let Ok((pi_hash, meta_json, task_json)) = ret {
        let expected_pi_hash = pi_hash.0.map(|byte| byte as c_char);
        HandlingResult {
            ok: true as c_char,
            universal_task: CString::new(task_json).unwrap().into_raw(),
            metadata: CString::new(meta_json).unwrap().into_raw(),
            expected_pi_hash,
        }
    } else {
        tracing::error!("gen_universal_task failed, error: {:#}", ret.unwrap_err());
        failed_handling_result()
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn release_task_result(result: HandlingResult) {
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
    proof: *const c_char,
    metadata: *const c_char,
    vk: *const c_char,
    vk_len: usize,
) -> *mut c_char {
    let proof_str = c_char_to_str(proof);
    let metadata_str = c_char_to_str(metadata);
    let vk_data = std::slice::from_raw_parts(vk as *const u8, vk_len);

    match libzkp::gen_wrapped_proof(proof_str, metadata_str, vk_data) {
        Ok(result) => CString::new(result).unwrap().into_raw(),
        Err(e) => {
            tracing::error!("gen_wrapped_proof failed, error: {:#}", e);
            std::ptr::null_mut()
        }
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn release_string(ptr: *mut c_char) {
    if !ptr.is_null() {
        let _ = CString::from_raw(ptr);
    }
}
