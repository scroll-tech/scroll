use crate::utils::{c_char_to_str, c_char_to_vec, panic_catch};
use libc::c_char;
use prover_v3::BatchProof as BatchProofLoVersion;
use prover_v4::{
    aggregator::Verifier as VerifierHiVersion, utils::init_env_and_log,
    BatchProof as BatchProofHiVersion, BundleProof,
};
use snark_verifier_sdk::verify_evm_calldata;
use std::{cell::OnceCell, env};

static mut VERIFIER: OnceCell<VerifierHiVersion> = OnceCell::new();

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn init_batch_verifier(params_dir: *const c_char, assets_dir: *const c_char) {
    init_env_and_log("ffi_batch_verify");

    let params_dir = c_char_to_str(params_dir);
    let assets_dir = c_char_to_str(assets_dir);

    // TODO: add a settings in scroll-prover.
    env::set_var("SCROLL_PROVER_ASSETS_DIR", assets_dir);
    let verifier_hi = VerifierHiVersion::from_dirs(params_dir, assets_dir);

    VERIFIER.set(verifier_hi).unwrap();
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
            let proof = serde_json::from_slice::<BatchProofLoVersion>(proof.as_slice()).unwrap();
            verify_evm_calldata(
                include_bytes!("plonk_verifier_0.11.4.bin").to_vec(),
                proof.calldata(),
            )
        } else {
            // Post upgrade #4 (Darwin), batch proofs are not EVM-verifiable. Instead they are
            // halo2 proofs meant to be bundled recursively.
            let proof = serde_json::from_slice::<BatchProofHiVersion>(proof.as_slice()).unwrap();
            VERIFIER.get().unwrap().verify_batch_proof(&proof)
        }
    });
    verified.unwrap_or(false) as c_char
}

/// # Safety
#[no_mangle]
pub unsafe extern "C" fn verify_bundle_proof(proof: *const c_char) -> c_char {
    let proof = c_char_to_vec(proof);
    let proof = serde_json::from_slice::<BundleProof>(proof.as_slice()).unwrap();
    let verified = panic_catch(|| VERIFIER.get().unwrap().verify_bundle_proof(proof));
    verified.unwrap_or(false) as c_char
}
