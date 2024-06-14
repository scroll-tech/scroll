use super::CircuitsHandler;
use crate::{geth_client::GethClient, types::ProofType};
use anyhow::{bail, Context, Ok, Result};
use serde::Deserialize;

use crate::types::{CommonHash, Task};
use std::{cell::RefCell, cmp::Ordering, rc::Rc};

use prover_next::{
    aggregator::Prover as BatchProver, check_chunk_hashes, zkevm::Prover as ChunkProver,
    BatchProof, BatchProvingTask, BlockTrace, ChunkInfo, ChunkProof, ChunkProvingTask,
};

use super::bernoulli::OUTPUT_DIR;

#[derive(Deserialize)]
pub struct BatchTaskDetail {
    pub chunk_infos: Vec<ChunkInfo>,
    pub chunk_proofs: Vec<ChunkProof>,
}

#[derive(Deserialize)]
pub struct ChunkTaskDetail {
    pub block_hashes: Vec<CommonHash>,
}

fn get_block_number(block_trace: &BlockTrace) -> Option<u64> {
    block_trace.header.number.map(|n| n.as_u64())
}

#[derive(Default)]
pub struct NextCircuitsHandler {
    chunk_prover: Option<RefCell<ChunkProver>>,
    batch_prover: Option<RefCell<BatchProver>>,

    geth_client: Option<Rc<RefCell<GethClient>>>,
}

impl NextCircuitsHandler {
    pub fn new(
        proof_type: ProofType,
        params_dir: &str,
        assets_dir: &str,
        geth_client: Option<Rc<RefCell<GethClient>>>,
    ) -> Result<Self> {
        match proof_type {
            ProofType::Chunk => Ok(Self {
                chunk_prover: Some(RefCell::new(ChunkProver::from_dirs(params_dir, assets_dir))),
                batch_prover: None,
                geth_client,
            }),

            ProofType::Batch => Ok(Self {
                batch_prover: Some(RefCell::new(BatchProver::from_dirs(params_dir, assets_dir))),
                chunk_prover: None,
                geth_client,
            }),
            _ => bail!("proof type invalid"),
        }
    }

    fn gen_chunk_proof_raw(&self, chunk_trace: Vec<BlockTrace>) -> Result<ChunkProof> {
        if let Some(prover) = self.chunk_prover.as_ref() {
            let chunk = ChunkProvingTask::from(chunk_trace);

            let chunk_proof =
                prover
                    .borrow_mut()
                    .gen_chunk_proof(chunk, None, None, self.get_output_dir())?;

            return Ok(chunk_proof);
        }
        unreachable!("please check errors in proof_type logic")
    }

    fn gen_chunk_proof(&self, task: &crate::types::Task) -> Result<String> {
        let chunk_trace = self.gen_chunk_traces(task)?;
        let chunk_proof = self.gen_chunk_proof_raw(chunk_trace)?;
        serde_json::to_string(&chunk_proof).map_err(|e| anyhow::anyhow!(e))
    }

    fn gen_batch_proof_raw(
        &self,
        chunk_hashes_proofs: Vec<(ChunkInfo, ChunkProof)>,
    ) -> Result<BatchProof> {
        if let Some(prover) = self.batch_prover.as_ref() {
            let chunk_proofs: Vec<ChunkProof> =
                chunk_hashes_proofs.iter().map(|t| t.1.clone()).collect();

            let is_valid = prover.borrow_mut().check_protocol_of_chunks(&chunk_proofs);

            if !is_valid {
                bail!("non-match chunk protocol")
            }
            check_chunk_hashes("", &chunk_hashes_proofs).context("failed to check chunk info")?;
            let batch = BatchProvingTask { chunk_proofs };
            let batch_proof =
                prover
                    .borrow_mut()
                    .gen_agg_evm_proof(batch, None, self.get_output_dir())?;

            return Ok(batch_proof);
        }
        unreachable!("please check errors in proof_type logic")
    }

    fn gen_batch_proof(&self, task: &crate::types::Task) -> Result<String> {
        log::info!("[circuit] gen_batch_proof for task {}", task.id);
        let chunk_hashes_proofs: Vec<(ChunkInfo, ChunkProof)> =
            self.gen_chunk_hashes_proofs(task)?;
        let batch_proof = self.gen_batch_proof_raw(chunk_hashes_proofs)?;
        serde_json::to_string(&batch_proof).map_err(|e| anyhow::anyhow!(e))
    }

    fn get_output_dir(&self) -> Option<&str> {
        OUTPUT_DIR.as_deref()
    }

    fn gen_chunk_traces(&self, task: &Task) -> Result<Vec<BlockTrace>> {
        let chunk_task_detail: ChunkTaskDetail = serde_json::from_str(&task.task_data)?;
        self.get_sorted_traces_by_hashes(&chunk_task_detail.block_hashes)
    }

    fn gen_chunk_hashes_proofs(&self, task: &Task) -> Result<Vec<(ChunkInfo, ChunkProof)>> {
        let batch_task_detail: BatchTaskDetail = serde_json::from_str(&task.task_data)?;

        Ok(batch_task_detail
            .chunk_infos
            .clone()
            .into_iter()
            .zip(batch_task_detail.chunk_proofs.clone())
            .collect())
    }

    fn get_sorted_traces_by_hashes(&self, block_hashes: &[CommonHash]) -> Result<Vec<BlockTrace>> {
        if block_hashes.is_empty() {
            log::error!("[prover] failed to get sorted traces: block_hashes are empty");
            bail!("block_hashes are empty")
        }

        let mut block_traces = Vec::new();
        for hash in block_hashes.iter() {
            let trace = self
                .geth_client
                .as_ref()
                .unwrap()
                .borrow_mut()
                .get_block_trace_by_hash(hash)?;
            block_traces.push(trace);
        }

        block_traces.sort_by(|a, b| {
            if get_block_number(a).is_none() {
                Ordering::Less
            } else if get_block_number(b).is_none() {
                Ordering::Greater
            } else {
                get_block_number(a)
                    .unwrap()
                    .cmp(&get_block_number(b).unwrap())
            }
        });

        let block_numbers: Vec<u64> = block_traces
            .iter()
            .map(|trace| get_block_number(trace).unwrap_or(0))
            .collect();
        let mut i = 0;
        while i < block_numbers.len() - 1 {
            if block_numbers[i] + 1 != block_numbers[i + 1] {
                log::error!(
                    "[prover] block numbers are not continuous, got {} and {}",
                    block_numbers[i],
                    block_numbers[i + 1]
                );
                bail!(
                    "block numbers are not continuous, got {} and {}",
                    block_numbers[i],
                    block_numbers[i + 1]
                )
            }
            i += 1;
        }

        Ok(block_traces)
    }
}

impl CircuitsHandler for NextCircuitsHandler {
    fn get_vk(&self, task_type: ProofType) -> Option<Vec<u8>> {
        match task_type {
            ProofType::Chunk => self
                .chunk_prover
                .as_ref()
                .and_then(|prover| prover.borrow().get_vk()),
            ProofType::Batch => self
                .batch_prover
                .as_ref()
                .and_then(|prover| prover.borrow().get_vk()),
            _ => unreachable!(),
        }
    }

    fn get_proof_data(&self, task_type: ProofType, task: &crate::types::Task) -> Result<String> {
        match task_type {
            ProofType::Chunk => self.gen_chunk_proof(task),
            ProofType::Batch => self.gen_batch_proof(task),
            _ => unreachable!(),
        }
    }
}

// =================================== tests module ========================================

#[cfg(test)]
mod tests {
    use super::*;
    use crate::zk_circuits_handler::utils::encode_vk;
    use prover_next::utils::chunk_trace_to_witness_block;
    use std::{path::PathBuf, sync::LazyLock};

    #[ctor::ctor]
    fn init() {
        crate::utils::log_init(None);
        log::info!("logger initialized");
    }

    static DEFAULT_WORK_DIR: &str = "/assets";
    static WORK_DIR: LazyLock<String> = LazyLock::new(|| {
        std::env::var("CURIE_TEST_DIR")
            .unwrap_or(String::from(DEFAULT_WORK_DIR))
            .trim_end_matches('/')
            .to_string()
    });
    static PARAMS_PATH: LazyLock<String> =
        LazyLock::new(|| format!("{}/test_params", WORK_DIR.clone()));
    static ASSETS_PATH: LazyLock<String> =
        LazyLock::new(|| format!("{}/test_assets", WORK_DIR.clone()));
    static PROOF_DUMP_PATH: LazyLock<String> =
        LazyLock::new(|| format!("{}/proof_data", WORK_DIR.clone()));
    static BATCH_DIR_PATH: LazyLock<String> =
        LazyLock::new(|| format!("{}/traces/batch_24", WORK_DIR.clone()));
    static BATCH_VK_PATH: LazyLock<String> =
        LazyLock::new(|| format!("{}/test_assets/agg_vk.vkey", WORK_DIR.clone()));
    static CHUNK_VK_PATH: LazyLock<String> =
        LazyLock::new(|| format!("{}/test_assets/chunk_vk.vkey", WORK_DIR.clone()));

    #[test]
    fn it_works() {
        let result = true;
        assert!(result);
    }

    #[test]
    fn test_circuits() -> Result<()> {
        let chunk_handler =
            NextCircuitsHandler::new(ProofType::Chunk, &PARAMS_PATH, &ASSETS_PATH, None)?;

        let chunk_vk = chunk_handler.get_vk(ProofType::Chunk).unwrap();

        check_vk(ProofType::Chunk, chunk_vk, "chunk vk must be available");
        let chunk_dir_paths = get_chunk_dir_paths()?;
        log::info!("chunk_dir_paths, {:?}", chunk_dir_paths);
        let mut chunk_infos = vec![];
        let mut chunk_proofs = vec![];
        for (id, chunk_path) in chunk_dir_paths.into_iter().enumerate() {
            let chunk_id = format!("chunk_proof{id}");
            log::info!("start to process {chunk_id}");
            let chunk_trace = read_chunk_trace(chunk_path)?;

            let chunk_info = traces_to_chunk_info(chunk_trace.clone())?;
            chunk_infos.push(chunk_info);

            log::info!("start to prove {chunk_id}");
            let chunk_proof = chunk_handler.gen_chunk_proof_raw(chunk_trace)?;
            let proof_data = serde_json::to_string(&chunk_proof)?;
            dump_proof(chunk_id, proof_data)?;
            chunk_proofs.push(chunk_proof);
        }

        let batch_handler =
            NextCircuitsHandler::new(ProofType::Batch, &PARAMS_PATH, &ASSETS_PATH, None)?;
        let batch_vk = batch_handler.get_vk(ProofType::Batch).unwrap();
        check_vk(ProofType::Batch, batch_vk, "batch vk must be available");
        let chunk_hashes_proofs = chunk_infos.into_iter().zip(chunk_proofs).collect();
        log::info!("start to prove batch");
        let batch_proof = batch_handler.gen_batch_proof_raw(chunk_hashes_proofs)?;
        let proof_data = serde_json::to_string(&batch_proof)?;
        dump_proof("batch_proof".to_string(), proof_data)?;

        Ok(())
    }

    fn check_vk(proof_type: ProofType, vk: Vec<u8>, info: &str) {
        log::info!("check_vk, {:?}", proof_type);
        let vk_from_file = read_vk(proof_type).unwrap();
        assert_eq!(vk_from_file, encode_vk(vk), "{info}")
    }

    fn read_vk(proof_type: ProofType) -> Result<String> {
        log::info!("read_vk, {:?}", proof_type);
        let vk_file = match proof_type {
            ProofType::Chunk => CHUNK_VK_PATH.clone(),
            ProofType::Batch => BATCH_VK_PATH.clone(),
            ProofType::Undefined => unreachable!(),
        };

        let data = std::fs::read(vk_file)?;
        Ok(encode_vk(data))
    }

    fn read_chunk_trace(path: PathBuf) -> Result<Vec<BlockTrace>> {
        log::info!("read_chunk_trace, {:?}", path);
        let mut chunk_trace: Vec<BlockTrace> = vec![];

        fn read_block_trace(file: &PathBuf) -> Result<BlockTrace> {
            let f = std::fs::File::open(file)?;
            serde_json::from_reader(&f).map_err(|e| anyhow::anyhow!(e))
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
        std::fs::write(dump_path.join(id), proof_data).map_err(|e| anyhow::anyhow!(e))
    }
}
