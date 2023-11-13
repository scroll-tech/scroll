use crate::{
    types::ProofResult,
    utils::{
        c_char_to_str, c_char_to_vec, file_exists, panic_catch, string_to_c_char, vec_to_c_char,
        OUTPUT_DIR,
    },
};
use libc::c_char;
use prover::{
    consts::CHUNK_VK_FILENAME,
    utils::{get_block_trace_from_file, init_env_and_log},
    zkevm::{Prover, Verifier},
    BlockTrace, ChunkProof,
};
use std::{
    env,
    process::{Command, Stdio},
    ptr::null,
};

#[no_mangle]
static mut PROVER: Option<Prover> = None;
#[no_mangle]
static mut VERIFIER: Option<Verifier> = None;

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn test_chunk(params_dir: *const c_char, assets_dir: *const c_char) {
    let params_dir = c_char_to_str(params_dir);
    let assets_dir = c_char_to_str(assets_dir);

    init_env_and_log("chunk_tests");

    env::set_var("SCROLL_PROVER_ASSETS_DIR", assets_dir);

    let prover = Prover::from_dirs(params_dir, assets_dir);
    PROVER = Some(prover);
    log::info!("Constructed chunk prover");

    let chunk_trace = vec![get_block_trace_from_file("/assets/traces/1_transfer.json")];
    log::info!("Loaded chunk trace");

    for i in 0..50 {
        log::info!("Proof-{i} BEGIN mem: {}", mem_usage());
        PROVER
            .as_mut()
            .unwrap()
            .gen_chunk_proof(chunk_trace.clone(), None, None, None);
        log::info!("Proof-{i} END mem: {}", mem_usage());
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_chunk_prover(params_dir: *const c_char, assets_dir: *const c_char) {
    init_env_and_log("ffi_chunk_prove");

    let params_dir = c_char_to_str(params_dir);
    let assets_dir = c_char_to_str(assets_dir);

    // TODO: add a settings in scroll-prover.
    env::set_var("SCROLL_PROVER_ASSETS_DIR", assets_dir);

    // VK file must exist, it is optional and logged as a warning in prover.
    if !file_exists(assets_dir, &CHUNK_VK_FILENAME) {
        panic!("{} must exist in folder {}", *CHUNK_VK_FILENAME, assets_dir);
    }

    let prover = Prover::from_dirs(params_dir, assets_dir);

    PROVER = Some(prover);
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

    VERIFIER = Some(verifier);
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn get_chunk_vk() -> *const c_char {
    let vk_result = panic_catch(|| PROVER.as_mut().unwrap().get_vk());

    vk_result
        .ok()
        .flatten()
        .map_or(null(), |vk| string_to_c_char(base64::encode(vk)))
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn gen_chunk_proof(
    block_traces: *mut u8,
    len: libc::c_uint,
) -> *const c_char {
    log::warn!("gupeng - aaaaa");
    // return null();

    // let block_traces1 = c_char_to_vec(block_traces);
    // let block_traces1 = std::slice::from_raw_parts(block_traces, len as usize);

    // let block_traces = serde_json::from_slice::<Vec<BlockTrace>>(block_traces1).unwrap();

    let block_traces = vec![get_block_trace_from_file("/assets/traces/1_transfer.json")];
    log::info!("Loaded chunk trace");

    // let block_traces = vec![get_block_trace_from_file("/assets/traces/1_transfer.json")];
    // log::info!("Loaded chunk trace");

    let prover = PROVER
        .as_mut()
        .expect("failed to get mutable reference to PROVER.");

    prover.gen_chunk_proof(block_traces, None, None, None); // OUTPUT_DIR.as_deref());
    log::warn!("gupeng - kkkkk");

    null()
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn verify_chunk_proof(proof: *const c_char) -> c_char {
    let proof = c_char_to_vec(proof);
    let proof = serde_json::from_slice::<ChunkProof>(proof.as_slice()).unwrap();

    let verified = panic_catch(|| VERIFIER.as_mut().unwrap().verify_chunk_proof(proof));
    verified.unwrap_or(false) as c_char
}

fn mem_usage() -> String {
    let cmd = "echo \"$(date '+%Y-%m-%d %H:%M:%S') $(free -g | grep Mem: | sed 's/Mem://g')\"";
    let output = Command::new("bash").arg("-c").arg(cmd).output().unwrap();
    String::from_utf8_lossy(&output.stdout).to_string()
}
