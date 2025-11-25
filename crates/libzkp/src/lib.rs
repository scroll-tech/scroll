pub mod proofs;
pub mod tasks;
pub use tasks::ProvingTaskExt;
pub mod verifier;
use verifier::HardForkName;
pub use verifier::{TaskType, VerifierConfig};
mod utils;

use sbv_primitives::B256;
use scroll_zkvm_types::{utils::vec_as_base64, version::Version};
use serde::{Deserialize, Serialize};
use serde_json::value::RawValue;
use std::{collections::HashMap, path::Path, sync::OnceLock};
use tasks::chunk_interpreter::{ChunkInterpreter, TryFromWithInterpreter};

pub(crate) fn witness_use_legacy_mode(fork_name: &str) -> eyre::Result<bool> {
    ADDITIONAL_FEATURES
        .get()
        .and_then(|features| features.get(fork_name))
        .map(|cfg| cfg.legacy_witness_encoding)
        .ok_or_else(|| {
            eyre::eyre!(
                "can not find features setting for unrecognized fork {}",
                fork_name
            )
        })
}

#[derive(Debug, Default, Clone)]
struct FeatureOptions {
    legacy_witness_encoding: bool,
    for_openvm_13_prover: bool,
}

static ADDITIONAL_FEATURES: OnceLock<HashMap<HardForkName, FeatureOptions>> = OnceLock::new();

impl FeatureOptions {
    pub fn new(feats: &str) -> Self {
        let mut ret: Self = Default::default();

        for feat_s in feats.split(':') {
            match feat_s.trim().to_lowercase().as_str() {
                "legacy_witness" => {
                    tracing::info!("set witness encoding for legacy mode");
                    ret.legacy_witness_encoding = true;
                }
                "openvm_13" => {
                    tracing::info!("set prover should use openvm 13");
                    ret.for_openvm_13_prover = true;
                }
                s => tracing::warn!("unrecognized dynamic feature: {s}"),
            }
        }
        ret
    }
}

/// Turn the coordinator's chunk task into a json string for formal chunk proving
/// task (with full witnesses)
pub fn checkout_chunk_task(
    task_json: &str,
    decryption_key: Option<&[u8]>,
    interpreter: impl ChunkInterpreter,
) -> eyre::Result<String> {
    let chunk_task = serde_json::from_str::<tasks::ChunkTask>(task_json)?;
    Ok(serde_json::to_string(
        &tasks::ChunkProvingTask::try_from_with_interpret(chunk_task, decryption_key, interpreter)?,
    )?)
}

/// Convert the universal task json into compatible form for old prover
pub fn univ_task_compatibility_fix(task_json: &str) -> eyre::Result<String> {
    use scroll_zkvm_types::proof::VmInternalStarkProof;

    let task: tasks::ProvingTask = serde_json::from_str(task_json)?;
    let aggregated_proofs: Vec<VmInternalStarkProof> = task
        .aggregated_proofs
        .into_iter()
        .map(|proof| VmInternalStarkProof {
            proofs: proof.proofs,
            public_values: proof.public_values,
        })
        .collect();

    #[derive(Serialize)]
    struct CompatibleProvingTask {
        /// seralized witness which should be written into stdin first
        pub serialized_witness: Vec<Vec<u8>>,
        /// aggregated proof carried by babybear fields, should be written into stdin
        /// followed `serialized_witness`
        pub aggregated_proofs: Vec<VmInternalStarkProof>,
        /// Fork name specify
        pub fork_name: String,
        /// The vk of app which is expcted to prove this task
        pub vk: Vec<u8>,
        /// An identifier assigned by coordinator, it should be kept identify for the
        /// same task (for example, using chunk, batch and bundle hashes)
        pub identifier: String,
    }

    let compatible_u_task = CompatibleProvingTask {
        serialized_witness: task.serialized_witness,
        aggregated_proofs,
        fork_name: task.fork_name,
        vk: task.vk,
        identifier: task.identifier,
    };

    Ok(serde_json::to_string(&compatible_u_task)?)
}

/// Generate required staff for proving tasks
/// return (pi_hash, metadata, task)
pub fn gen_universal_task(
    task_type: i32,
    task_json: &str,
    fork_name_str: &str,
    expected_vk: &[u8],
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
            let mut task = serde_json::from_str::<ChunkProvingTask>(task_json)?;
            // normailze fork name field in task
            task.fork_name = task.fork_name.to_lowercase();
            let version = Version::from(task.version);
            // always respect the fork_name_str (which has been normalized) being passed
            // if the fork_name wrapped in task is not match, consider it a malformed task
            if fork_name_str != task.fork_name.as_str() {
                eyre::bail!("fork name in chunk task not match the calling arg, expected {fork_name_str}, get {}", task.fork_name);
            }
            if fork_name_str != version.fork.as_str() {
                eyre::bail!(
                    "given task version, expected fork={fork_name_str}, got={version_fork}",
                    version_fork = version.fork.as_str()
                );
            }
            let (pi_hash, metadata, u_task) =
                utils::panic_catch(move || gen_universal_chunk_task(task))
                    .map_err(|e| eyre::eyre!("caught panic in chunk task{e}"))??;
            (pi_hash, AnyMetaData::Chunk(metadata), u_task)
        }
        x if x == TaskType::Batch as i32 => {
            let mut task = serde_json::from_str::<BatchProvingTask>(task_json)?;
            task.fork_name = task.fork_name.to_lowercase();
            let version = Version::from(task.version);
            if fork_name_str != task.fork_name.as_str() {
                eyre::bail!("fork name in batch task not match the calling arg, expected {fork_name_str}, get {}", task.fork_name);
            }
            if fork_name_str != version.fork.as_str() {
                eyre::bail!(
                    "given task version, expected fork={fork_name_str}, got={version_fork}",
                    version_fork = version.fork.as_str()
                );
            }
            let (pi_hash, metadata, u_task) =
                utils::panic_catch(move || gen_universal_batch_task(task))
                    .map_err(|e| eyre::eyre!("caught panic in chunk task{e}"))??;
            (pi_hash, AnyMetaData::Batch(metadata), u_task)
        }
        x if x == TaskType::Bundle as i32 => {
            let mut task = serde_json::from_str::<BundleProvingTask>(task_json)?;
            task.fork_name = task.fork_name.to_lowercase();
            let version = Version::from(task.version);
            if fork_name_str != task.fork_name.as_str() {
                eyre::bail!("fork name in bundle task not match the calling arg, expected {fork_name_str}, get {}", task.fork_name);
            }
            if fork_name_str != version.fork.as_str() {
                eyre::bail!(
                    "given task version, expected fork={fork_name_str}, got={version_fork}",
                    version_fork = version.fork.as_str()
                );
            }
            let (pi_hash, metadata, u_task) =
                utils::panic_catch(move || gen_universal_bundle_task(task))
                    .map_err(|e| eyre::eyre!("caught panic in chunk task{e}"))??;
            (pi_hash, AnyMetaData::Bundle(metadata), u_task)
        }
        _ => return Err(eyre::eyre!("unrecognized task type {task_type}")),
    };

    u_task.vk = Vec::from(expected_vk);
    let fork_name = u_task.fork_name.clone();
    let mut u_task_ext = ProvingTaskExt::new(u_task);

    // set additional settings from global features
    if let Some(cfg) = ADDITIONAL_FEATURES
        .get()
        .and_then(|features| features.get(&fork_name))
    {
        u_task_ext.use_openvm_13 = cfg.for_openvm_13_prover;
    } else {
        tracing::warn!(
            "can not found features setting for unrecognized fork {}",
            fork_name
        );
    }

    Ok((
        pi_hash,
        serde_json::to_string(&metadata)?,
        serde_json::to_string(&u_task_ext)?,
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
    ADDITIONAL_FEATURES
        .set(HashMap::from_iter(cfg.circuits.iter().map(|config| {
            tracing::info!(
                "start setting features [{:?}] for fork {}",
                config.features,
                config.fork_name
            );
            (
                config.fork_name.to_lowercase(),
                config
                    .features
                    .as_ref()
                    .map(|features| FeatureOptions::new(features.as_str()))
                    .unwrap_or_default(),
            )
        })))
        .map_err(|c| eyre::eyre!("Fail to init additional features: {c:?}"))?;

    verifier::init(cfg);

    Ok(())
}

/// verify proof
pub fn verify_proof(proof: Vec<u8>, fork_name: &str, task_type: TaskType) -> eyre::Result<bool> {
    let verifier = verifier::get_verifier(fork_name)?;

    let ret = verifier.lock().unwrap().verify(task_type, &proof)?;
    Ok(ret)
}

/// dump vk
pub fn dump_vk(fork_name: &str, file: &str) -> eyre::Result<()> {
    let verifier = verifier::get_verifier(fork_name)?;

    verifier.lock().unwrap().dump_vk(Path::new(file));

    Ok(())
}
