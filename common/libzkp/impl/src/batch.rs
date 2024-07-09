use crate::{
    types::{CheckChunkProofsResponse, ProofResult},
    utils::{
        c_char_to_str, c_char_to_vec, file_exists, panic_catch, string_to_c_char, vec_to_c_char,
        OUTPUT_DIR,
    },
};
use libc::c_char;
use prover_v3::BatchProof as BatchProofV3;
use prover_v4::{
    aggregator::{Prover, Verifier as VerifierV4},
    check_chunk_hashes,
    consts::BATCH_VK_FILENAME,
    utils::{chunk_trace_to_witness_block, init_env_and_log},
    BatchHeader, BatchProof as BatchProofV4, BatchProvingTask, BlockTrace, BundleProof,
    BundleProvingTask, ChunkInfo, ChunkProof,
};
use snark_verifier_sdk::verify_evm_calldata;
use std::{cell::OnceCell, env, ptr::null};

static mut PROVER: OnceCell<Prover> = OnceCell::new();
static mut VERIFIER_V4: OnceCell<VerifierV4> = OnceCell::new();

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_batch_prover(params_dir: *const c_char, assets_dir: *const c_char) {
    init_env_and_log("ffi_batch_prove");

    let params_dir = c_char_to_str(params_dir);
    let assets_dir = c_char_to_str(assets_dir);

    // TODO: add a settings in scroll-prover.
    env::set_var("SCROLL_PROVER_ASSETS_DIR", assets_dir);

    // VK file must exist, it is optional and logged as a warning in prover.
    if !file_exists(assets_dir, &BATCH_VK_FILENAME) {
        panic!("{} must exist in folder {}", *BATCH_VK_FILENAME, assets_dir);
    }

    let prover = Prover::from_dirs(params_dir, assets_dir);

    PROVER.set(prover).unwrap();
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_batch_verifier(params_dir: *const c_char, assets_dir: *const c_char) {
    init_env_and_log("ffi_batch_verify");

    let params_dir = c_char_to_str(params_dir);
    let assets_dir = c_char_to_str(assets_dir);

    // TODO: add a settings in scroll-prover.
    env::set_var("SCROLL_PROVER_ASSETS_DIR", assets_dir);
    let verifier_v4 = VerifierV4::from_dirs(params_dir, assets_dir);

    VERIFIER_V4.set(verifier_v4).unwrap();
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn get_batch_vk() -> *const c_char {
    let vk_result = panic_catch(|| PROVER.get_mut().unwrap().get_batch_vk());

    vk_result
        .ok()
        .flatten()
        .map_or(null(), |vk| string_to_c_char(base64::encode(vk)))
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn get_bundle_vk() -> *const c_char {
    let vk_result = panic_catch(|| PROVER.get_mut().unwrap().get_bundle_vk());

    vk_result
        .ok()
        .flatten()
        .map_or(null(), |vk| string_to_c_char(base64::encode(vk)))
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn check_chunk_proofs(chunk_proofs: *const c_char) -> *const c_char {
    let check_result: Result<bool, String> = panic_catch(|| {
        let chunk_proofs = c_char_to_vec(chunk_proofs);
        let chunk_proofs = serde_json::from_slice::<Vec<ChunkProof>>(&chunk_proofs)
            .map_err(|e| format!("failed to deserialize chunk proofs: {e:?}"))?;

        if chunk_proofs.is_empty() {
            return Err("provided chunk proofs are empty.".to_string());
        }

        let prover_ref = PROVER.get().expect("failed to get reference to PROVER.");

        let valid = prover_ref.check_protocol_of_chunks(&chunk_proofs);
        Ok(valid)
    })
    .unwrap_or_else(|e| Err(format!("unwind error: {e:?}")));

    let r = match check_result {
        Ok(valid) => CheckChunkProofsResponse {
            ok: valid,
            error: None,
        },
        Err(err) => CheckChunkProofsResponse {
            ok: false,
            error: Some(err),
        },
    };

    serde_json::to_vec(&r).map_or(null(), vec_to_c_char)
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn gen_batch_proof(
    chunk_hashes: *const c_char,
    chunk_proofs: *const c_char,
    batch_header: *const c_char,
) -> *const c_char {
    let proof_result: Result<Vec<u8>, String> = panic_catch(|| {
        let chunk_hashes = c_char_to_vec(chunk_hashes);
        let chunk_proofs = c_char_to_vec(chunk_proofs);
        let batch_header = c_char_to_vec(batch_header);

        let chunk_hashes = serde_json::from_slice::<Vec<ChunkInfo>>(&chunk_hashes)
            .map_err(|e| format!("failed to deserialize chunk hashes: {e:?}"))?;
        let chunk_proofs = serde_json::from_slice::<Vec<ChunkProof>>(&chunk_proofs)
            .map_err(|e| format!("failed to deserialize chunk proofs: {e:?}"))?;
        let batch_header = serde_json::from_slice::<BatchHeader>(&batch_header)
            .map_err(|e| format!("failed to deserialize batch header: {e:?}"))?;

        if chunk_hashes.len() != chunk_proofs.len() {
            return Err(format!("chunk hashes and chunk proofs lengths mismatch: chunk_hashes.len() = {}, chunk_proofs.len() = {}",
                chunk_hashes.len(), chunk_proofs.len()));
        }

        let chunk_hashes_proofs: Vec<(_,_)> = chunk_hashes
            .into_iter()
            .zip(chunk_proofs.clone())
            .collect();
        check_chunk_hashes("", &chunk_hashes_proofs).map_err(|e| format!("failed to check chunk info: {e:?}"))?;

        let batch = BatchProvingTask {
            chunk_proofs,
            batch_header,
        };
        let proof = PROVER
            .get_mut()
            .expect("failed to get mutable reference to PROVER.")
            .gen_batch_proof(batch, None, OUTPUT_DIR.as_deref())
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
pub unsafe extern "C" fn verify_batch_proof(
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
            // As of upgrade #3 (Curie), we verify batch proofs on-chain (EVM).
            let proof = serde_json::from_slice::<BatchProofV3>(proof.as_slice()).unwrap();
            verify_evm_calldata(
                include_bytes!("plonk_verifier_0.11.4.bin").to_vec(),
                proof.calldata(),
            )
        } else {
            // Post upgrade #4 (Darwin), batch proofs are not EVM-verifiable. Instead they are
            // halo2 proofs meant to be bundled recursively.
            let proof = serde_json::from_slice::<BatchProofV4>(proof.as_slice()).unwrap();
            VERIFIER_V4.get().unwrap().verify_batch_proof(proof)
        }
    });
    verified.unwrap_or(false) as c_char
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn gen_bundle_proof(batch_proofs: *const c_char) -> *const c_char {
    let proof_result: Result<Vec<u8>, String> = panic_catch(|| {
        let batch_proofs = c_char_to_vec(batch_proofs);
        let batch_proofs = serde_json::from_slice::<Vec<BatchProofV4>>(&batch_proofs)
            .map_err(|e| format!("failed to deserialize batch proofs: {e:?}"))?;

        let bundle = BundleProvingTask { batch_proofs };
        let proof = PROVER
            .get_mut()
            .expect("failed to get mutable reference to PROVER.")
            .gen_bundle_proof(bundle, None, OUTPUT_DIR.as_deref())
            .map_err(|e| format!("failed to generate bundle proof: {e:?}"))?;

        serde_json::to_vec(&proof)
            .map_err(|e| format!("failed to serialize the bundle proof: {e:?}"))
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
pub unsafe extern "C" fn verify_bundle_proof(proof: *const c_char) -> c_char {
    let proof = c_char_to_vec(proof);
    let proof = serde_json::from_slice::<BundleProof>(proof.as_slice()).unwrap();
    let verified = panic_catch(|| VERIFIER_V4.get().unwrap().verify_bundle_proof(proof));
    verified.unwrap_or(false) as c_char
}

// This function is only used for debugging on Go side.
/// # Safety
#[no_mangle]
pub unsafe extern "C" fn block_traces_to_chunk_info(block_traces: *const c_char) -> *const c_char {
    let block_traces = c_char_to_vec(block_traces);
    let block_traces = serde_json::from_slice::<Vec<BlockTrace>>(&block_traces).unwrap();

    let witness_block = chunk_trace_to_witness_block(block_traces).unwrap();
    let chunk_info = ChunkInfo::from_witness_block(&witness_block, false);

    let chunk_info_bytes = serde_json::to_vec(&chunk_info).unwrap();
    vec_to_c_char(chunk_info_bytes)
}
