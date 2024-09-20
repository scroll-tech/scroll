use super::{common::*, CircuitsHandler};
use crate::{
    geth_client::GethClient,
    types::{ProverType, TaskType},
};
use anyhow::{bail, Context, Ok, Result};
use once_cell::sync::Lazy;
use serde::Deserialize;

use crate::types::{CommonHash, Task};
use std::{cell::RefCell, cmp::Ordering, env, rc::Rc};

use prover_darwin_v2::{
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
pub struct DarwinV2Handler {
    chunk_prover: Option<RefCell<ChunkProver<'static>>>,
    batch_prover: Option<RefCell<BatchProver<'static>>>,

    geth_client: Option<Rc<RefCell<GethClient>>>,
}

impl DarwinV2Handler {
    pub fn new_multi(
        prover_types: Vec<ProverType>,
        params_dir: &str,
        assets_dir: &str,
        geth_client: Option<Rc<RefCell<GethClient>>>,
    ) -> Result<Self> {
        let class_name = std::intrinsics::type_name::<Self>();
        let prover_types_set = prover_types
            .into_iter()
            .collect::<std::collections::HashSet<ProverType>>();
        let mut handler = Self {
            batch_prover: None,
            chunk_prover: None,
            geth_client,
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
                    handler.chunk_prover = Some(RefCell::new(ChunkProver::from_params_and_assets(
                        params_map, assets_dir,
                    )));
                }
                ProverType::Batch => {
                    handler.batch_prover = Some(RefCell::new(BatchProver::from_params_and_assets(
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
        geth_client: Option<Rc<RefCell<GethClient>>>,
    ) -> Result<Self> {
        Self::new_multi(vec![prover_type], params_dir, assets_dir, geth_client)
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
        Ok(serde_json::to_string(&chunk_proof)?)
    }

    fn gen_batch_proof_raw(&self, batch_task_detail: BatchTaskDetail) -> Result<BatchProof> {
        if let Some(prover) = self.batch_prover.as_ref() {
            let chunk_hashes_proofs: Vec<(ChunkInfo, ChunkProof)> = batch_task_detail
                .chunk_infos
                .clone()
                .into_iter()
                .zip(batch_task_detail.batch_proving_task.chunk_proofs.clone())
                .collect();

            let chunk_proofs: Vec<ChunkProof> =
                chunk_hashes_proofs.iter().map(|t| t.1.clone()).collect();

            let is_valid = prover.borrow_mut().check_protocol_of_chunks(&chunk_proofs);

            if !is_valid {
                bail!("non-match chunk protocol")
            }
            check_chunk_hashes("", &chunk_hashes_proofs).context("failed to check chunk info")?;
            let batch_proof = prover.borrow_mut().gen_batch_proof(
                batch_task_detail.batch_proving_task,
                None,
                self.get_output_dir(),
            )?;

            return Ok(batch_proof);
        }
        unreachable!("please check errors in proof_type logic")
    }

    fn gen_batch_proof(&self, task: &crate::types::Task) -> Result<String> {
        log::info!("[circuit] gen_batch_proof for task {}", task.id);

        let batch_task_detail: BatchTaskDetail = serde_json::from_str(&task.task_data)?;
        let batch_proof = self.gen_batch_proof_raw(batch_task_detail)?;
        Ok(serde_json::to_string(&batch_proof)?)
    }

    fn gen_bundle_proof_raw(&self, bundle_task_detail: BundleTaskDetail) -> Result<BundleProof> {
        if let Some(prover) = self.batch_prover.as_ref() {
            let bundle_proof = prover.borrow_mut().gen_bundle_proof(
                bundle_task_detail,
                None,
                self.get_output_dir(),
            )?;

            return Ok(bundle_proof);
        }
        unreachable!("please check errors in proof_type logic")
    }

    fn gen_bundle_proof(&self, task: &crate::types::Task) -> Result<String> {
        log::info!("[circuit] gen_bundle_proof for task {}", task.id);
        let bundle_task_detail: BundleTaskDetail = serde_json::from_str(&task.task_data)?;
        let bundle_proof = self.gen_bundle_proof_raw(bundle_task_detail)?;
        Ok(serde_json::to_string(&bundle_proof)?)
    }

    fn get_output_dir(&self) -> Option<&str> {
        OUTPUT_DIR.as_deref()
    }

    fn gen_chunk_traces(&self, task: &Task) -> Result<Vec<BlockTrace>> {
        let chunk_task_detail: ChunkTaskDetail = serde_json::from_str(&task.task_data)?;
        self.get_sorted_traces_by_hashes(&chunk_task_detail.block_hashes)
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

impl CircuitsHandler for DarwinV2Handler {
    fn get_vk(&self, task_type: TaskType) -> Option<Vec<u8>> {
        match task_type {
            TaskType::Chunk => self
                .chunk_prover
                .as_ref()
                .and_then(|prover| prover.borrow().get_vk()),
            TaskType::Batch => self
                .batch_prover
                .as_ref()
                .and_then(|prover| prover.borrow().get_batch_vk()),
            TaskType::Bundle => self
                .batch_prover
                .as_ref()
                .and_then(|prover| prover.borrow().get_bundle_vk()),
            _ => unreachable!(),
        }
    }

    fn get_proof_data(&self, task_type: TaskType, task: &crate::types::Task) -> Result<String> {
        match task_type {
            TaskType::Chunk => self.gen_chunk_proof(task),
            TaskType::Batch => self.gen_batch_proof(task),
            TaskType::Bundle => self.gen_bundle_proof(task),
            _ => unreachable!(),
        }
    }
}
