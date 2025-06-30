pub mod proofs;
pub mod tasks;
pub mod verifier;
pub use verifier::{TaskType, VerifierConfig};
mod utils;

use sbv_primitives::B256;
use scroll_zkvm_types::util::vec_as_base64;
use serde::{Deserialize, Serialize};
use serde_json::value::RawValue;
use std::path::Path;
use tasks::chunk_interpreter::{ChunkInterpreter, TryFromWithInterpreter};

/// Turn the coordinator's chunk task into a json string for formal chunk proving
/// task (with full witnesses)
pub fn checkout_chunk_task(
    task_json: &str,
    interpreter: impl ChunkInterpreter,
) -> eyre::Result<String> {
    let chunk_task = serde_json::from_str::<tasks::ChunkTask>(task_json)?;
    let ret = serde_json::to_string(&tasks::ChunkProvingTask::try_from_with_interpret(
        chunk_task,
        interpreter,
    )?)?;
    Ok(ret)
}

/// Generate required staff for proving tasks
/// return (pi_hash, metadata, task)
pub fn gen_universal_task(
    task_type: i32,
    task_json: &str,
    fork_name: &str,
    expected_vk: &[u8],
    interpreter: Option<impl ChunkInterpreter>,
) -> eyre::Result<(B256, String, String)> {
    use proofs::*;
    use tasks::*;

    /// Wrapper for metadata
    #[derive(Clone, Debug, Serialize, Deserialize)]
    #[serde(untagged)]
    enum AnyMetaData {
        Chunk(ChunkProofMetadata),
        Batch(BatchProofMetadata),
        Bundle(BundleProofMetadata),
    }

    let (pi_hash, metadata, mut u_task) = match task_type {
        x if x == TaskType::Chunk as i32 => {
            let task = serde_json::from_str::<ChunkProvingTask>(task_json)?;
            let (pi_hash, metadata, u_task) =
                gen_universal_chunk_task(task, fork_name.into(), interpreter)?;
            (pi_hash, AnyMetaData::Chunk(metadata), u_task)
        }
        x if x == TaskType::Batch as i32 => {
            let task = serde_json::from_str::<BatchProvingTask>(task_json)?;
            let (pi_hash, metadata, u_task) = gen_universal_batch_task(task, fork_name.into())?;
            (pi_hash, AnyMetaData::Batch(metadata), u_task)
        }
        x if x == TaskType::Bundle as i32 => {
            let task = serde_json::from_str::<BundleProvingTask>(task_json)?;
            let (pi_hash, metadata, u_task) = gen_universal_bundle_task(task, fork_name.into())?;
            (pi_hash, AnyMetaData::Bundle(metadata), u_task)
        }
        _ => return Err(eyre::eyre!("unrecognized task type {task_type}")),
    };

    u_task.vk = Vec::from(expected_vk);

    Ok((
        pi_hash,
        serde_json::to_string(&metadata)?,
        serde_json::to_string(&u_task)?,
    ))
}

/// helper to rearrange the proof return by universal prover into corresponding wrapped proof
pub fn gen_wrapped_proof(proof_json: &str, metadata: &str, vk: &[u8]) -> eyre::Result<String> {
    #[derive(Serialize)]
    struct RearrangeWrappedProofJson<'a> {
        #[serde(borrow)]
        pub metadata: &'a RawValue,
        #[serde(borrow)]
        pub proof: &'a RawValue,
        #[serde(with = "vec_as_base64", default)]
        pub vk: Vec<u8>,
        pub git_version: String,
    }

    let re_arrange = RearrangeWrappedProofJson {
        metadata: serde_json::from_str(metadata)?,
        proof: serde_json::from_str(proof_json)?,
        vk: vk.to_vec(),
        git_version: utils::short_git_version(),
    };

    let ret = serde_json::to_string(&re_arrange)?;
    Ok(ret)
}

/// init verifier
pub fn verifier_init(config: &str) -> eyre::Result<()> {
    let cfg: VerifierConfig = serde_json::from_str(config)?;
    verifier::init(cfg);
    Ok(())
}

/// verify proof
pub fn verify_proof(proof: Vec<u8>, fork_name: &str, task_type: TaskType) -> eyre::Result<bool> {
    let verifier = verifier::get_verifier(fork_name)?;

    let ret = verifier.lock().unwrap().verify(task_type, &proof)?;

    if let Ok(debug_value) = std::env::var("ZKVM_DEBUG_PROOF") {
        use std::time::{SystemTime, UNIX_EPOCH};
        if !ret && debug_value.to_lowercase() == "true" {
            // Dump req.input to a temporary file
            let timestamp = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap_or_default()
                .as_secs();
            let filename = format!("/tmp/proof_{}.json", timestamp);
            if let Err(e) = std::fs::write(&filename, &proof) {
                eprintln!("Failed to write proof to file {}: {}", filename, e);
            } else {
                println!("Dumped failed proof to {}", filename);
            }
        }
    }

    Ok(ret)
}

/// dump vk
pub fn dump_vk(fork_name: &str, file: &str) -> eyre::Result<()> {
    let verifier = verifier::get_verifier(fork_name)?;

    verifier.lock().unwrap().dump_vk(Path::new(file));

    Ok(())
}
