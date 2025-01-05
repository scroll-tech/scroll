use super::{common::*, CircuitsHandler};
use crate::types::ProverType;
use async_trait::async_trait;
use scroll_proving_sdk::prover::{
        proving_service::ProveRequest,
        CircuitType,
    };
use anyhow::{bail, Context, Ok, Result};
use once_cell::sync::Lazy;
use serde::Deserialize;
use tokio::sync::RwLock;

use crate::types::CommonHash;
use std::env;

use prover_darwin::{
    aggregator::Prover as BatchProver,
    check_chunk_hashes,
    common::Prover as CommonProver,
    config::{AGG_DEGREES, ZKEVM_DEGREES},
    zkevm::Prover as ChunkProver,
    BatchProof, BatchProvingTask, BlockTrace, BundleProof, BundleProvingTask, ChunkInfo,
    ChunkProof, ChunkProvingTask,
};

// Only used for debugging.
static OUTPUT_DIR: Lazy<Option<String>> = Lazy::new(|| env::var("PROVER_OUTPUT_DIR").ok());

#[derive(Debug, Clone, Deserialize)]
pub struct BatchTaskDetail {
    pub chunk_infos: Vec<ChunkInfo>,
    #[serde(flatten)]
    pub batch_proving_task: BatchProvingTask,
}

type BundleTaskDetail = BundleProvingTask;

#[derive(Debug, Clone, Deserialize)]
pub struct ChunkTaskDetail {
    pub block_hashes: Vec<CommonHash>,
}

fn get_block_number(block_trace: &BlockTrace) -> Option<u64> {
    block_trace.header.number.map(|n| n.as_u64())
}

#[derive(Default)]
pub struct DarwinHandler {
    chunk_prover: Option<RwLock<ChunkProver<'static>>>,
    batch_prover: Option<RwLock<BatchProver<'static>>>,
}

impl DarwinHandler {
    pub fn new_multi(
        prover_types: Vec<ProverType>,
        params_dir: &str,
        assets_dir: &str,
    ) -> Result<Self> {
        let class_name = std::intrinsics::type_name::<Self>();
        let prover_types_set = prover_types
            .into_iter()
            .collect::<std::collections::HashSet<ProverType>>();
        let mut handler = Self {
            batch_prover: None,
            chunk_prover: None,
        };
        let degrees: Vec<u32> = get_degrees(&prover_types_set, |prover_type| match prover_type {
            ProverType::Chunk => ZKEVM_DEGREES.clone(),
            ProverType::Batch => AGG_DEGREES.clone(),
        });
        let params_map = get_params_map_instance(|| {
            log::info!(
                "calling get_params_map from {}, prover_types: {:?}, degrees: {:?}",
                class_name,
                prover_types_set,
                degrees
            );
            CommonProver::load_params_map(params_dir, &degrees)
        });
        for prover_type in prover_types_set {
            match prover_type {
                ProverType::Chunk => {
                    handler.chunk_prover = Some(RwLock::new(ChunkProver::from_params_and_assets(
                        params_map, assets_dir,
                    )));
                }
                ProverType::Batch => {
                    handler.batch_prover = Some(RwLock::new(BatchProver::from_params_and_assets(
                        params_map, assets_dir,
                    )))
                }
            }
        }
        Ok(handler)
    }

    pub fn new(
        prover_type: ProverType,
        params_dir: &str,
        assets_dir: &str,
    ) -> Result<Self> {
        Self::new_multi(vec![prover_type], params_dir, assets_dir)
    }

    async fn gen_chunk_proof_raw(&self, chunk_trace: Vec<BlockTrace>) -> Result<ChunkProof> {
        if let Some(prover) = self.chunk_prover.as_ref() {
            let chunk = ChunkProvingTask::from(chunk_trace);

            let chunk_proof =
                prover
                    .write()
                    .await
                    .gen_chunk_proof(chunk, None, None, self.get_output_dir())?;

            return Ok(chunk_proof);
        }
        unreachable!("please check errors in proof_type logic")
    }

    async fn gen_chunk_proof(&self, prove_request: ProveRequest) -> Result<String> {
        let chunk_traces: Vec<BlockTrace> = serde_json::from_str(&prove_request.input)?;
        let chunk_proof = self.gen_chunk_proof_raw(chunk_traces).await?;
        Ok(serde_json::to_string(&chunk_proof)?)
    }

    async fn gen_batch_proof_raw(&self, batch_task_detail: BatchTaskDetail) -> Result<BatchProof> {
        if let Some(prover) = self.batch_prover.as_ref() {
            let chunk_hashes_proofs: Vec<(ChunkInfo, ChunkProof)> = batch_task_detail
                .chunk_infos
                .clone()
                .into_iter()
                .zip(batch_task_detail.batch_proving_task.chunk_proofs.clone())
                .collect();

            let chunk_proofs: Vec<ChunkProof> =
                chunk_hashes_proofs.iter().map(|t| t.1.clone()).collect();

            let is_valid = prover.read().await.check_protocol_of_chunks(&chunk_proofs);

            if !is_valid {
                bail!("non-match chunk protocol")
            }
            check_chunk_hashes("", &chunk_hashes_proofs).context("failed to check chunk info")?;
            let batch_proof = prover.write().await.gen_batch_proof(
                batch_task_detail.batch_proving_task,
                None,
                self.get_output_dir(),
            )?;

            return Ok(batch_proof);
        }
        unreachable!("please check errors in proof_type logic")
    }

    async fn gen_batch_proof(&self, prove_request: ProveRequest) -> Result<String> {
        let batch_task_detail: BatchTaskDetail = serde_json::from_str(&prove_request.input)?;
        let batch_proof = self.gen_batch_proof_raw(batch_task_detail).await?;
        Ok(serde_json::to_string(&batch_proof)?)
    }

    async fn gen_bundle_proof_raw(&self, bundle_task_detail: BundleTaskDetail) -> Result<BundleProof> {
        if let Some(prover) = self.batch_prover.as_ref() {
            let bundle_proof = prover.write().await.gen_bundle_proof(
                bundle_task_detail,
                None,
                self.get_output_dir(),
            )?;

            return Ok(bundle_proof);
        }
        unreachable!("please check errors in proof_type logic")
    }

    async fn gen_bundle_proof(&self, prove_request: ProveRequest) -> Result<String> {
        let bundle_task_detail: BundleTaskDetail = serde_json::from_str(&prove_request.input)?;
        let bundle_proof = self.gen_bundle_proof_raw(bundle_task_detail).await?;
        Ok(serde_json::to_string(&bundle_proof)?)
    }

    fn get_output_dir(&self) -> Option<&str> {
        OUTPUT_DIR.as_deref()
    }
}

#[async_trait]
impl CircuitsHandler for DarwinHandler {
    async fn get_vk(&self, task_type: CircuitType) -> Option<Vec<u8>> {
        match task_type {
            CircuitType::Chunk => self.chunk_prover.as_ref().unwrap().read().await.get_vk(),
            CircuitType::Batch => self.batch_prover.as_ref().unwrap().read().await.get_batch_vk(),
            CircuitType::Bundle => self.batch_prover.as_ref().unwrap().read().await.get_bundle_vk(),
            _ => unreachable!(),
        }
    }

    async fn get_proof_data(&self, prove_request: ProveRequest) -> Result<String> {
        match prove_request.circuit_type {
            CircuitType::Chunk => self.gen_chunk_proof(prove_request).await,
            CircuitType::Batch => self.gen_batch_proof(prove_request).await,
            CircuitType::Bundle => self.gen_bundle_proof(prove_request).await,
            _ => unreachable!(),
        }
    }
}

// =================================== tests module ========================================

#[cfg(test)]
mod tests {
    use super::*;
    use crate::zk_circuits_handler::utils::encode_vk;
    use prover_darwin::utils::chunk_trace_to_witness_block;
    use std::{path::PathBuf, sync::LazyLock};

    #[ctor::ctor]
    fn init() {
        crate::utils::log_init(None);
        log::info!("logger initialized");
    }

    static DEFAULT_WORK_DIR: &str = "/assets";
    static WORK_DIR: LazyLock<String> = LazyLock::new(|| {
        std::env::var("DARWIN_TEST_DIR")
            .unwrap_or(String::from(DEFAULT_WORK_DIR))
            .trim_end_matches('/')
            .to_string()
    });
    static PARAMS_PATH: LazyLock<String> = LazyLock::new(|| format!("{}/test_params", *WORK_DIR));
    static ASSETS_PATH: LazyLock<String> = LazyLock::new(|| format!("{}/test_assets", *WORK_DIR));
    static PROOF_DUMP_PATH: LazyLock<String> =
        LazyLock::new(|| format!("{}/proof_data", *WORK_DIR));
    static BATCH_DIR_PATH: LazyLock<String> =
        LazyLock::new(|| format!("{}/traces/batch_24", *WORK_DIR));
    static BATCH_VK_PATH: LazyLock<String> =
        LazyLock::new(|| format!("{}/test_assets/vk_batch.vkey", *WORK_DIR));
    static CHUNK_VK_PATH: LazyLock<String> =
        LazyLock::new(|| format!("{}/test_assets/vk_chunk.vkey", *WORK_DIR));

    #[test]
    fn it_works() {
        let result = true;
        assert!(result);
    }

    #[tokio::test]
    async fn test_circuits() -> Result<()> {
        let bi_handler = DarwinHandler::new_multi(
            vec![ProverType::Chunk, ProverType::Batch],
            &PARAMS_PATH,
            &ASSETS_PATH,
        )?;

        let chunk_handler = bi_handler;
        let chunk_vk = chunk_handler.get_vk(CircuitType::Chunk).await.unwrap();

        check_vk(CircuitType::Chunk, chunk_vk, "chunk vk must be available");
        let chunk_dir_paths = get_chunk_dir_paths()?;
        log::info!("chunk_dir_paths, {:?}", chunk_dir_paths);
        let mut chunk_infos = vec![];
        let mut chunk_proofs = vec![];
        for (id, chunk_path) in chunk_dir_paths.into_iter().enumerate() {
            let chunk_id = format!("chunk_proof{}", id + 1);
            log::info!("start to process {chunk_id}");
            let chunk_trace = read_chunk_trace(chunk_path)?;

            let chunk_info = traces_to_chunk_info(chunk_trace.clone())?;
            chunk_infos.push(chunk_info);

            log::info!("start to prove {chunk_id}");
            let chunk_proof = chunk_handler.gen_chunk_proof_raw(chunk_trace).await?;
            let proof_data = serde_json::to_string(&chunk_proof)?;
            dump_proof(chunk_id, proof_data)?;
            chunk_proofs.push(chunk_proof);
        }

        let batch_handler = chunk_handler;
        let batch_vk = batch_handler.get_vk(CircuitType::Batch).await.unwrap();
        check_vk(CircuitType::Batch, batch_vk, "batch vk must be available");
        let batch_task_detail = make_batch_task_detail(chunk_infos, chunk_proofs);
        log::info!("start to prove batch");
        let batch_proof = batch_handler.gen_batch_proof_raw(batch_task_detail).await?;
        let proof_data = serde_json::to_string(&batch_proof)?;
        dump_proof("batch_proof".to_string(), proof_data)?;

        Ok(())
    }

    fn make_batch_task_detail(_: Vec<ChunkInfo>, _: Vec<ChunkProof>) -> BatchTaskDetail {
        todo!();
        // BatchTaskDetail {
        //     chunk_infos,
        //     batch_proving_task: BatchProvingTask {
        //         parent_batch_hash: todo!(),
        //         parent_state_root: todo!(),
        //         batch_header: todo!(),
        //         chunk_proofs,
        //     },
        // }
    }

    fn check_vk(proof_type: CircuitType, vk: Vec<u8>, info: &str) {
        log::info!("check_vk, {:?}", proof_type);
        let vk_from_file = read_vk(proof_type).unwrap();
        assert_eq!(vk_from_file, encode_vk(vk), "{info}")
    }

    fn read_vk(proof_type: CircuitType) -> Result<String> {
        log::info!("read_vk, {:?}", proof_type);
        let vk_file = match proof_type {
            CircuitType::Chunk => CHUNK_VK_PATH.clone(),
            CircuitType::Batch => BATCH_VK_PATH.clone(),
            CircuitType::Bundle => todo!(),
            CircuitType::Undefined => unreachable!(),
        };

        let data = std::fs::read(vk_file)?;
        Ok(encode_vk(data))
    }

    fn read_chunk_trace(path: PathBuf) -> Result<Vec<BlockTrace>> {
        log::info!("read_chunk_trace, {:?}", path);
        let mut chunk_trace: Vec<BlockTrace> = vec![];

        fn read_block_trace(file: &PathBuf) -> Result<BlockTrace> {
            let f = std::fs::File::open(file)?;
            Ok(serde_json::from_reader(&f)?)
        }

        if path.is_dir() {
            let entries = std::fs::read_dir(&path)?;
            let mut files: Vec<String> = entries
                .into_iter()
                .filter_map(|e| {
                    if e.is_err() {
                        return None;
                    }
                    let entry = e.unwrap();
                    if entry.path().is_dir() {
                        return None;
                    }
                    if let Result::Ok(file_name) = entry.file_name().into_string() {
                        Some(file_name)
                    } else {
                        None
                    }
                })
                .collect();
            files.sort();

            log::info!("files in chunk {:?} is {:?}", path, files);
            for file in files {
                let block_trace = read_block_trace(&path.join(file))?;
                chunk_trace.push(block_trace);
            }
        } else {
            let block_trace = read_block_trace(&path)?;
            chunk_trace.push(block_trace);
        }
        Ok(chunk_trace)
    }

    fn get_chunk_dir_paths() -> Result<Vec<PathBuf>> {
        let batch_path = PathBuf::from(BATCH_DIR_PATH.clone());
        let entries = std::fs::read_dir(&batch_path)?;
        let mut files: Vec<String> = entries
            .filter_map(|e| {
                if e.is_err() {
                    return None;
                }
                let entry = e.unwrap();
                if entry.path().is_dir() {
                    if let Result::Ok(file_name) = entry.file_name().into_string() {
                        Some(file_name)
                    } else {
                        None
                    }
                } else {
                    None
                }
            })
            .collect();
        files.sort();
        log::info!("files in batch {:?} is {:?}", batch_path, files);
        Ok(files.into_iter().map(|f| batch_path.join(f)).collect())
    }

    fn traces_to_chunk_info(chunk_trace: Vec<BlockTrace>) -> Result<ChunkInfo> {
        let witness_block = chunk_trace_to_witness_block(chunk_trace)?;
        Ok(ChunkInfo::from_witness_block(&witness_block, false))
    }

    fn dump_proof(id: String, proof_data: String) -> Result<()> {
        let dump_path = PathBuf::from(PROOF_DUMP_PATH.clone());
        Ok(std::fs::write(dump_path.join(id), proof_data)?)
    }
}
