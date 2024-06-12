use super::CircuitsHandler;
use crate::{geth_client::GethClient, types::ProofType};
use anyhow::{bail, Context, Ok, Result};
use serde::Deserialize;

use crate::types::{Task, CommonHash};
use std::{cell::RefCell, cmp::Ordering, rc::Rc};

use prover_next::{BlockTrace, ChunkInfo, ChunkProof, ChunkProvingTask, BatchProvingTask};
use prover_next::{aggregator::Prover as BatchProver, check_chunk_hashes, zkevm::Prover as ChunkProver};

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
    pub fn new(proof_type: ProofType, params_dir: &str, assets_dir: &str, geth_client: Option<Rc<RefCell<GethClient>>>) -> Result<Self> {
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

    fn gen_chunk_proof(&self, task: &crate::types::Task) -> Result<String> {
        let chunk_trace = self.gen_chunk_traces(task)?;
        if let Some(prover) = self.chunk_prover.as_ref() {
            let chunk = ChunkProvingTask::from(chunk_trace);

            let chunk_proof = prover
                .borrow_mut()
                .gen_chunk_proof(chunk, None, None, self.get_output_dir())?;

            return serde_json::to_string(&chunk_proof).map_err(|e| anyhow::anyhow!(e));
        }
        unreachable!("please check errors in proof_type logic")
    }

    fn gen_batch_proof(&self, task: &crate::types::Task) -> Result<String> {
        if let Some(prover) = self.batch_prover.as_ref() {
            let chunk_hashes_proofs: Vec<(ChunkInfo, ChunkProof)> =
            self.gen_chunk_hashes_proofs(task)?;
            let chunk_proofs: Vec<ChunkProof> =
                chunk_hashes_proofs.iter().map(|t| t.1.clone()).collect();

            let is_valid = prover
                .borrow_mut()
                .check_protocol_of_chunks(&chunk_proofs);

            if !is_valid {
                bail!("non-match chunk protocol, task-id: {}", &task.id)
            }
            check_chunk_hashes("", &chunk_hashes_proofs).context("failed to check chunk info")?;
            let batch = BatchProvingTask {
                chunk_proofs
            };
            let batch_proof = prover
                .borrow_mut()
                .gen_agg_evm_proof(
                    batch,
                    None,
                    self.get_output_dir(),
                )?;

            return serde_json::to_string(&batch_proof).map_err(|e| anyhow::anyhow!(e));
        }
        unreachable!("please check errors in proof_type logic")
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

    fn get_sorted_traces_by_hashes(
        &self,
        block_hashes: &Vec<CommonHash>,
    ) -> Result<Vec<BlockTrace>> {
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
            block_traces.push(trace.block_trace);
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
                log::error!("[prover] block numbers are not continuous, got {} and {}", block_numbers[i], block_numbers[i + 1]);
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
            ProofType::Chunk => {
                self.chunk_prover.as_ref()
                .and_then(|prover| prover.borrow().get_vk())
            },
            ProofType::Batch => {
                self.batch_prover.as_ref()
                .and_then(|prover| prover.borrow().get_vk())
            },
            _ => unreachable!()
        }
    }

    fn get_proof_data(&self, task_type: ProofType, task: &crate::types::Task) -> Result<String> {
        match task_type {
            ProofType::Chunk => self.gen_chunk_proof(task),
            ProofType::Batch => self.gen_batch_proof(task),
            _ => unreachable!()
        }
    }
}
