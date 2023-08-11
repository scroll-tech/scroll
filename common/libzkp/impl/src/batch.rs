use crate::utils::{c_char_to_str, c_char_to_vec, vec_to_c_char, OUTPUT_DIR};
use libc::c_char;
use prover::{
    aggregator::{Prover, Verifier},
    utils::{chunk_trace_to_witness_block, init_env_and_log},
    BatchProof, ChunkHash, ChunkProof,
};
use std::{cell::OnceCell, panic, ptr::null};
use types::eth::BlockTrace;

static mut PROVER: OnceCell<Prover> = OnceCell::new();
static mut VERIFIER: OnceCell<Verifier> = OnceCell::new();

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_batch_prover(params_dir: *const c_char) {
    init_env_and_log("ffi_batch_prove");

    let params_dir = c_char_to_str(params_dir);

    let result = panic::catch_unwind(|| Prover::from_params_dir(params_dir));
    match result {
        Ok(prover) => PROVER.set(prover).unwrap(),
        Err(err) => log::error!("Failed to init batch-prover: {err:?}"),
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_batch_verifier(params_dir: *const c_char, assets_dir: *const c_char) {
    init_env_and_log("ffi_batch_verify");

    let params_dir = c_char_to_str(params_dir);
    let assets_dir = c_char_to_str(assets_dir);

    let result = panic::catch_unwind(|| Verifier::from_dirs(params_dir, assets_dir));
    match result {
        Ok(verifier) => VERIFIER.set(verifier).unwrap(),
        Err(err) => log::error!("Failed to init batch-verifier: {err:?}"),
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn gen_batch_proof(
    chunk_hashes: *const c_char,
    chunk_proofs: *const c_char,
) -> *const c_char {
    let chunk_hashes = c_char_to_vec(chunk_hashes);
    let chunk_proofs = c_char_to_vec(chunk_proofs);

    let chunk_hashes = serde_json::from_slice::<Vec<ChunkHash>>(&chunk_hashes).unwrap();
    let chunk_proofs = serde_json::from_slice::<Vec<ChunkProof>>(&chunk_proofs).unwrap();
    assert_eq!(chunk_hashes.len(), chunk_proofs.len());

    let chunk_hashes_proofs = chunk_hashes
        .into_iter()
        .zip(chunk_proofs.into_iter())
        .collect();

    let result = panic::catch_unwind(|| {
        let proof = PROVER
            .get_mut()
            .unwrap()
            .gen_agg_evm_proof(chunk_hashes_proofs, None, OUTPUT_DIR.as_deref())
            .unwrap();

        serde_json::to_vec(&proof).unwrap()
    });

    match result {
        Ok(result) => vec_to_c_char(result),
        Err(err) => {
            log::error!("Failed to gen batch-proof: {err:?}");
            null()
        }
    }
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn verify_batch_proof(proof: *const c_char) -> c_char {
    let proof = c_char_to_vec(proof);
    let proof = serde_json::from_slice::<BatchProof>(proof.as_slice()).unwrap();

    let result = panic::catch_unwind(|| VERIFIER.get().unwrap().verify_agg_evm_proof(proof));
    let verified = match result {
        Ok(verified) => verified,
        Err(err) => {
            log::error!("Failed to verify batch-proof: {err:?}");
            false
        }
    };

    verified as c_char
}

// This function is only used for debugging on Go side.
/// # Safety
#[no_mangle]
pub unsafe extern "C" fn block_traces_to_chunk_info(block_traces: *const c_char) -> *const c_char {
    let block_traces = c_char_to_vec(block_traces);
    let block_traces = serde_json::from_slice::<Vec<BlockTrace>>(&block_traces).unwrap();

    let witness_block = chunk_trace_to_witness_block(block_traces).unwrap();
    let chunk_info = ChunkHash::from_witness_block(&witness_block, false);

    let chunk_info_bytes = serde_json::to_vec(&chunk_info).unwrap();
    vec_to_c_char(chunk_info_bytes)
}
