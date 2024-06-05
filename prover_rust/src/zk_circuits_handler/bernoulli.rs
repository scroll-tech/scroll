use super::CircuitsHandler;
use once_cell::sync::Lazy;
use crate::{geth_client::GethClient, types::ProofType};
use anyhow::{bail, Ok, Result};
use serde::Deserialize;

use crate::types::{Task, CommonHash};
use std::{cell::RefCell, env, cmp::Ordering, rc::Rc};
use prover::{aggregator::Prover as BatchProver, zkevm::Prover as ChunkProver};
use prover::{BlockTrace, ChunkHash, ChunkProof};

// Only used for debugging.
pub(crate) static OUTPUT_DIR: Lazy<Option<String>> =
    Lazy::new(|| env::var("PROVER_OUTPUT_DIR").ok());


#[derive(Deserialize)]
pub struct BatchTaskDetail {
    pub chunk_infos: Vec<ChunkHash>,
    pub chunk_proofs: Vec<ChunkProof>,
}

#[derive(Deserialize)]
pub struct ChunkTaskDetail {
    pub block_hashes: Vec<CommonHash>,
}

fn get_block_number(block_trace: &BlockTrace) -> Option<u64> {
    block_trace.header.number.map(|n| n.as_u64())
}

pub struct BaseCircuitsHandler {
    chunk_prover: Option<RefCell<ChunkProver>>,
    batch_prover: Option<RefCell<BatchProver>>,

    geth_client: Option<Rc<RefCell<GethClient>>>,
}

impl BaseCircuitsHandler {
    pub fn new(proof_type: ProofType, params_dir: &str, assets_dir: &str, geth_client: Option<Rc<RefCell<GethClient>>>) -> Result<Self> {
        match proof_type {
            ProofType::ProofTypeChunk => Ok(Self {
                chunk_prover: Some(RefCell::new(ChunkProver::from_dirs(params_dir, assets_dir))),
                batch_prover: None,
                geth_client,
            }),

            ProofType::ProofTypeBatch => Ok(Self {
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
            let chunk_proof = prover
                .borrow_mut()
                .gen_chunk_proof(chunk_trace, None, None, self.get_output_dir())?;

            return serde_json::to_string(&chunk_proof).map_err(|e| anyhow::anyhow!(e));
        }
        unreachable!("please check errors in proof_type logic")
    }

    fn gen_batch_proof(&self, task: &crate::types::Task) -> Result<String> {
        let chunk_hashes_proofs: Vec<(ChunkHash, ChunkProof)> =
            self.gen_chunk_hashes_proofs(task)?;
        let chunk_proofs: Vec<ChunkProof> =
            chunk_hashes_proofs.iter().map(|t| t.1.clone()).collect();

        if let Some(prover) = self.batch_prover.as_ref() {
            let is_valid = prover
                .borrow_mut()
                .check_chunk_proofs(&chunk_proofs);

            if !is_valid {
                bail!("non-match chunk protocol, task-id: {}", &task.id)
            }
            let batch_proof = prover
                .borrow_mut()
                .gen_agg_evm_proof(
                    chunk_hashes_proofs,
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

    fn gen_chunk_hashes_proofs(&self, task: &Task) -> Result<Vec<(ChunkHash, ChunkProof)>> {
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
        if block_hashes.len() == 0 {
            log::error!("[prover] failed to get sorted traces: block_hashes are empty");
            bail!("block_hashes are empty")
        }

        let mut block_traces = Vec::new();
        for (_, hash) in block_hashes.into_iter().enumerate() {
            let trace = self
                .geth_client
                .as_ref()
                .unwrap()
                .borrow_mut()
                .get_block_trace_by_hash(hash)?;
            block_traces.push(trace.block_trace);
        }

        block_traces.sort_by(|a, b| {
            if get_block_number(a) == None {
                Ordering::Less
            } else if get_block_number(b) == None {
                Ordering::Greater
            } else {
                get_block_number(a)
                    .unwrap()
                    .cmp(&get_block_number(b).unwrap())
            }
        });

        let block_numbers: Vec<u64> = block_traces
            .iter()
            .map(|trace| match get_block_number(trace) {
                Some(v) => v,
                None => 0,
            })
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

impl CircuitsHandler for BaseCircuitsHandler {
    fn get_vk(&self, task_type: ProofType) -> Option<Vec<u8>> {
        match task_type {
            ProofType::ProofTypeChunk => {
                self.chunk_prover.as_ref()
                .and_then(|prover| prover.borrow().get_vk())
            },
            ProofType::ProofTypeBatch => {
                self.batch_prover.as_ref()
                .and_then(|prover| prover.borrow().get_vk())
            },
            _ => unreachable!()
        }
    }

    fn get_proof_data(&self, task_type: ProofType, task: &crate::types::Task) -> Result<String> {
        match task_type {
            ProofType::ProofTypeChunk => self.gen_chunk_proof(task),
            ProofType::ProofTypeBatch => self.gen_batch_proof(task),
            _ => unreachable!()
        }
    }
}
