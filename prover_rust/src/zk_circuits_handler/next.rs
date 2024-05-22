use std::cell::RefCell;
use anyhow::{bail, Ok, Result};
use crate::types::ProofType;
use super::{
    types::*,
    CircuitsHandler,
};
use prover_next::{aggregator::Prover as NextBatchProver, zkevm::Prover as NextChunkProver};

#[derive(Default)]
pub struct NextCircuitsHandler {
    chunk_prover: Option<RefCell<NextChunkProver>>,
    batch_prover: Option<RefCell<NextBatchProver>>,
}

impl NextCircuitsHandler {
    pub fn new(proof_type: ProofType, params_dir: &str, assets_dir: &str) -> Result<Self> {
        match proof_type {
            ProofType::ProofTypeChunk => Ok(Self {
                chunk_prover: Some(RefCell::new(NextChunkProver::from_dirs(params_dir, assets_dir))),
                ..Default::default()
            }),

            ProofType::ProofTypeBatch => Ok(Self {
                batch_prover: Some(RefCell::new(NextBatchProver::from_dirs(params_dir, assets_dir))),
                ..Default::default()
            }),
            _ => bail!("proof type invalid"),
        }
    }
}

impl CircuitsHandler for NextCircuitsHandler {
    // api of zkevm::Prover
    fn prover_get_vk(&self) -> Option<Vec<u8>> {
        self.chunk_prover
            .as_ref()
            .and_then(|prover| prover.borrow().get_vk())
    }

    fn prover_gen_chunk_proof(
        &self,
        chunk_trace: Vec<BlockTrace>,
        name: Option<&str>,
        inner_id: Option<&str>,
        output_dir: Option<&str>,
    ) -> Result<ChunkProof> {
        if let Some(prover) = self.chunk_prover.as_ref() {
            let next_chunk_trace = chunk_trace.into_iter()
            .map(|block_trace| block_trace_base_to_next(block_trace))
            .collect::<Result<Vec<NextBlockTrace>>>()?;

            let next_chunk_proof = prover
            .borrow_mut()
            .gen_chunk_proof(next_chunk_trace, name, inner_id, output_dir)?;

            return chunk_proof_next_to_base(next_chunk_proof);
        }
        unreachable!("please check errors in proof_type logic")
    }

    // api of aggregator::Prover
    fn aggregator_get_vk(&self) -> Option<Vec<u8>> {
        self.batch_prover
            .as_ref()
            .and_then(|prover| prover.borrow().get_vk())
    }

    fn aggregator_gen_agg_evm_proof(
        &self,
        chunk_hashes_proofs: Vec<(ChunkHash, ChunkProof)>,
        name: Option<&str>,
        output_dir: Option<&str>,
    ) -> Result<BatchProof> {
        if let Some(prover) = self.batch_prover.as_ref() {
            let next_chunk_hashes_proofs = chunk_hashes_proofs.into_iter().map(|t| {
                let next_chunk_hash = chunk_hash_base_to_next(t.0);
                let next_chunk_proof = chunk_proof_base_to_next(&t.1);
                match next_chunk_proof {
                    Result::Ok(proof) => Ok((next_chunk_hash, proof)),
                    Err(err) => Err(err)
                }
            }).collect::<Result<Vec<(NextChunkHash, NextChunkProof)>>>()?;

            let next_batch_proof = prover
            .borrow_mut()
            .gen_agg_evm_proof(next_chunk_hashes_proofs, name, output_dir)?;

            return batch_proof_next_to_base(next_batch_proof);
        }
        unreachable!("please check errors in proof_type logic")
    }

    fn aggregator_check_chunk_proofs(&self, chunk_proofs: &[ChunkProof]) -> Result<bool> {
        if let Some(prover) = self.batch_prover.as_ref() {
            let next_chunk_proofs = chunk_proofs.into_iter().map(|chunk_proof| {
                chunk_proof_base_to_next(chunk_proof)
            }).collect::<Result<Vec<NextChunkProof>>>()?;

            return Ok(prover.borrow_mut().check_chunk_proofs(&next_chunk_proofs));
        }
        unreachable!("please check errors in proof_type logic")
    }
}