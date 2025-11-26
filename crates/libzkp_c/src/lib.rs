mod utils;

use std::ffi::{CString, c_char};

use libzkp::TaskType;
use utils::{c_char_to_str, c_char_to_vec};

use std::sync::OnceLock;

static LOG_SETTINGS: OnceLock<Result<(), String>> = OnceLock::new();

fn enable_dump() -> bool {
    static ZKVM_DEBUG_DUMP: OnceLock<bool> = OnceLock::new();
    *ZKVM_DEBUG_DUMP.get_or_init(|| {
        std::env::var("ZKVM_DEBUG")
            .or_else(|_| std::env::var("ZKVM_DEBUG_PROOF"))
            .map(|s| s.to_lowercase() == "true")
            .unwrap_or(false)
    })
}

/// # Safety
#[unsafe(no_mangle)]
pub unsafe extern "C" fn init_tracing() {
    use tracing_subscriber::filter::{EnvFilter, LevelFilter};

    LOG_SETTINGS
        .get_or_init(|| {
            tracing_subscriber::fmt()
                .with_env_filter(
                    EnvFilter::builder()
                        .with_default_directive(LevelFilter::INFO.into())
                        .from_env_lossy(),
                )
                .with_ansi(false)
                .with_level(true)
                .with_target(true)
                .try_init()
                .map_err(|e| format!("{e}"))?;

            Ok(())
        })
        .clone()
        .expect("Failed to initialize tracing subscriber");

    tracing::info!("Tracing has been initialized normally");
}

/// # Safety
#[unsafe(no_mangle)]
pub unsafe extern "C" fn init_verifier(config: *const c_char) {
    let config_str = c_char_to_str(config);
    libzkp::verifier_init(config_str).unwrap();
}

/// # Safety
#[unsafe(no_mangle)]
pub unsafe extern "C" fn init_l2geth(config: *const c_char) {
    let config_str = c_char_to_str(config);
    l2geth::init(config_str).unwrap();
}

fn verify_proof(proof: *const c_char, fork_name: *const c_char, task_type: TaskType) -> c_char {
    let fork_name_str = c_char_to_str(fork_name);
    let proof_str = proof;
    let proof = c_char_to_vec(proof);

    match libzkp::verify_proof(proof, fork_name_str, task_type) {
        Err(e) => {
            tracing::error!("{:?} verify failed, error: {:#}", task_type, e);
            false as c_char
        }
        Ok(result) => {
            if !result && enable_dump() {
                use std::time::{SystemTime, UNIX_EPOCH};
                // Dump req.input to a temporary file
                let timestamp = SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .unwrap_or_default()
                    .as_secs();
                let filename = format!("/tmp/proof_{}.json", timestamp);
                let cstr = unsafe { std::ffi::CStr::from_ptr(proof_str) };
                if let Err(e) = std::fs::write(&filename, cstr.to_bytes()) {
                    eprintln!("Failed to write proof to file {}: {}", filename, e);
                } else {
                    println!("Dumped failed proof to {}", filename);
                }
            }
            result as c_char
        }
    }
}

/// # Safety
#[unsafe(no_mangle)]
pub unsafe extern "C" fn verify_chunk_proof(
    proof: *const c_char,
    fork_name: *const c_char,
) -> c_char {
    verify_proof(proof, fork_name, TaskType::Chunk)
}

/// # Safety
#[unsafe(no_mangle)]
pub unsafe extern "C" fn verify_batch_proof(
    proof: *const c_char,
    fork_name: *const c_char,
) -> c_char {
    verify_proof(proof, fork_name, TaskType::Batch)
}

/// # Safety
#[unsafe(no_mangle)]
pub unsafe extern "C" fn verify_bundle_proof(
    proof: *const c_char,
    fork_name: *const c_char,
) -> c_char {
    verify_proof(proof, fork_name, TaskType::Bundle)
}

/// # Safety
#[unsafe(no_mangle)]
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
#[unsafe(no_mangle)]
pub unsafe extern "C" fn gen_universal_task(
    task_type: i32,
    task: *const c_char,
    fork_name: *const c_char,
    expected_vk: *const u8,
    expected_vk_len: usize,
    decryption_key: *const u8,
    decryption_key_len: usize,
) -> HandlingResult {
    let task_json = if task_type == TaskType::Chunk as i32 {
        let pre_task_str = c_char_to_str(task);
        let cli = l2geth::get_client();
        let decryption_key = if decryption_key_len > 0 {
            if decryption_key_len != 32 {
                tracing::error!(
                    "gen_universal_task received {}-byte decryption key; expected 32",
                    decryption_key_len
                );
                return failed_handling_result();
            }
            Some(unsafe { std::slice::from_raw_parts(decryption_key, decryption_key_len) })
        } else {
            None
        };
        match libzkp::checkout_chunk_task(pre_task_str, decryption_key, cli) {
            Ok(str) => str,
            Err(e) => {
                tracing::error!("gen_universal_task failed at pre interpret step, error: {e}");
                return failed_handling_result();
            }
        }
    } else {
        c_char_to_str(task).to_string()
    };

    let expected_vk = if expected_vk_len > 0 {
        unsafe { std::slice::from_raw_parts(expected_vk, expected_vk_len) }
    } else {
        &[]
    };

    let ret =
        libzkp::gen_universal_task(task_type, &task_json, c_char_to_str(fork_name), expected_vk);

    if let Ok((pi_hash, meta_json, task_json)) = ret {
        let expected_pi_hash = pi_hash.0.map(|byte| byte as c_char);
        HandlingResult {
            ok: true as c_char,
            universal_task: CString::new(task_json).unwrap().into_raw(),
            metadata: CString::new(meta_json).unwrap().into_raw(),
            expected_pi_hash,
        }
    } else {
        if enable_dump() {
            use std::time::{SystemTime, UNIX_EPOCH};
            // Dump req.input to a temporary file
            let timestamp = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap_or_default()
                .as_secs();
            let c_str = unsafe { std::ffi::CStr::from_ptr(fork_name) };
            let filename = format!("/tmp/task_{}_{}.json", c_str.to_str().unwrap(), timestamp);
            if let Err(e) = std::fs::write(&filename, task_json.as_bytes()) {
                eprintln!("Failed to write task to file {}: {}", filename, e);
            } else {
                println!("Dumped failed task to {}", filename);
            }
        }

        tracing::error!("gen_universal_task failed, error: {:#}", ret.unwrap_err());
        failed_handling_result()
    }
}

/// # Safety
#[unsafe(no_mangle)]
pub unsafe extern "C" fn release_task_result(result: HandlingResult) {
    if !result.universal_task.is_null() {
        let _ = unsafe { CString::from_raw(result.universal_task) };
    }
    if !result.metadata.is_null() {
        let _ = unsafe { CString::from_raw(result.metadata) };
    }
}

/// # Safety
#[unsafe(no_mangle)]
pub unsafe extern "C" fn gen_wrapped_proof(
    proof: *const c_char,
    metadata: *const c_char,
    vk: *const c_char,
    vk_len: usize,
) -> *mut c_char {
    let proof_str = c_char_to_str(proof);
    let metadata_str = c_char_to_str(metadata);
    let vk_data = unsafe { std::slice::from_raw_parts(vk as *const u8, vk_len) };

    match libzkp::gen_wrapped_proof(proof_str, metadata_str, vk_data) {
        Ok(result) => CString::new(result).unwrap().into_raw(),
        Err(e) => {
            tracing::error!("gen_wrapped_proof failed, error: {:#}", e);
            std::ptr::null_mut()
        }
    }
}

/// # Safety
#[unsafe(no_mangle)]
pub unsafe extern "C" fn univ_task_compatibility_fix(task_json: *const c_char) -> *mut c_char {
    let task_json_str = c_char_to_str(task_json);
    match libzkp::univ_task_compatibility_fix(task_json_str) {
        Ok(result) => CString::new(result).unwrap().into_raw(),
        Err(e) => {
            tracing::error!("univ_task_compability_fix failed, error: {:#}", e);
            std::ptr::null_mut()
        }
    }
}

/// # Safety
#[unsafe(no_mangle)]
pub unsafe extern "C" fn release_string(ptr: *mut c_char) {
    if !ptr.is_null() {
        let _ = unsafe { CString::from_raw(ptr) };
    }
}
