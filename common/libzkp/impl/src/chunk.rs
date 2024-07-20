use crate::{
    types::ProofResult,
    utils::{
        c_char_to_str, c_char_to_vec, file_exists, panic_catch, string_to_c_char, vec_to_c_char,
        OUTPUT_DIR,
    },
};
use libc::c_char;
use prover_v3::{zkevm::Verifier as VerifierV3, ChunkProof as ChunkProofV3};
use prover_v4::{
    consts::CHUNK_VK_FILENAME,
    utils::init_env_and_log,
    zkevm::{Prover, Verifier as VerifierV4},
    BlockTrace, ChunkProof as ChunkProofV4, ChunkProvingTask,
};
use std::{cell::OnceCell, env, ptr::null};

static mut PROVER: OnceCell<Prover> = OnceCell::new();
static mut VERIFIER_V3: OnceCell<VerifierV3> = OnceCell::new();
static mut VERIFIER_V4: OnceCell<VerifierV4> = OnceCell::new();

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

    PROVER.set(prover).unwrap();
}

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
    let verifier_v3 = VerifierV3::from_dirs(params_dir, v3_assets_dir);
    env::set_var("SCROLL_PROVER_ASSETS_DIR", v4_assets_dir);
    let verifier_v4 = VerifierV4::from_dirs(params_dir, v4_assets_dir);

    VERIFIER_V3.set(verifier_v3).unwrap();
    VERIFIER_V4.set(verifier_v4).unwrap();
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn get_chunk_vk() -> *const c_char {
    let vk_result = panic_catch(|| PROVER.get_mut().unwrap().get_vk());

    vk_result
        .ok()
        .flatten()
        .map_or(null(), |vk| string_to_c_char(base64::encode(vk)))
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn gen_chunk_proof(block_traces: *const c_char) -> *const c_char {
    let proof_result: Result<Vec<u8>, String> = panic_catch(|| {
        let block_traces = c_char_to_vec(block_traces);
        let block_traces = serde_json::from_slice::<Vec<BlockTrace>>(&block_traces)
            .map_err(|e| format!("failed to deserialize block traces: {e:?}"))?;
        let chunk = ChunkProvingTask::from(block_traces);

        let proof = PROVER
            .get_mut()
            .expect("failed to get mutable reference to PROVER.")
            .gen_chunk_proof(chunk, None, None, OUTPUT_DIR.as_deref())
            .map_err(|e| format!("failed to generate proof: {e:?}"))?;

        serde_json::to_vec(&proof).map_err(|e| format!("failed to serialize the proof: {e:?}"))
    })
    .unwrap_or_else(|e| Err(format!("unwind error: {e:?}")));

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
            let proof = serde_json::from_slice::<ChunkProofV3>(proof.as_slice()).unwrap();
            VERIFIER_V3.get().unwrap().verify_chunk_proof(proof)
        } else {
            let proof = serde_json::from_slice::<ChunkProofV4>(proof.as_slice()).unwrap();
            VERIFIER_V4.get().unwrap().verify_chunk_proof(proof)
        }
    });
    verified.unwrap_or(false) as c_char
}
