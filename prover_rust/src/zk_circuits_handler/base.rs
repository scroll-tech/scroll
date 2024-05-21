
use anyhow::{bail, Ok, Result};
use crate::types::ProofType;
use super::CircuitsHandler;
use super::types::{ChunkProof, BatchProof, BlockTrace, ChunkHash};

use prover::zkevm::Prover as ChunkProver;
use prover::aggregator::Prover as BatchProver;

use std::cell::RefCell;

#[derive(Default)]
pub struct BaseCircuitsHandler {
    chunk_prover: Option<RefCell<ChunkProver>>,
    batch_prover: Option<RefCell<BatchProver>>,
}

impl BaseCircuitsHandler {
    pub fn new(proof_type: ProofType, params_dir: &str, assets_dir: &str) -> Result<Self> {
        match proof_type {
            ProofType::ProofTypeChunk => Ok(Self {
                chunk_prover: Some(RefCell::new(ChunkProver::from_dirs(params_dir, assets_dir))),
                ..Default::default()
            }),

            ProofType::ProofTypeBatch => Ok(Self {
                batch_prover: Some(RefCell::new(BatchProver::from_dirs(params_dir, assets_dir))),
                ..Default::default()
            }),
            _ => bail!("proof type invalid")
        }
    }
}

impl CircuitsHandler for BaseCircuitsHandler {
    // api of zkevm::Prover
    fn prover_get_vk(&self) -> Option<Vec<u8>> {
        self.chunk_prover.as_ref().and_then(|prover| prover.borrow().get_vk())
    }

    fn prover_gen_chunk_proof(&self,
        chunk_trace: Vec<BlockTrace>,
        name: Option<&str>,
        inner_id: Option<&str>,
        output_dir: Option<&str>) -> Result<ChunkProof> {
            if let Some(prover) = self.chunk_prover.as_ref() {
                return prover.borrow_mut().gen_chunk_proof(chunk_trace, name, inner_id, output_dir)
            }
            unreachable!("please check errors in proof_type logic")
        }

    // api of aggregator::Prover
    fn aggregator_get_vk(&self) -> Option<Vec<u8>> {
        self.batch_prover.as_ref().and_then(|prover| prover.borrow().get_vk())
    }

    fn aggregator_gen_agg_evm_proof(&self,
        chunk_hashes_proofs: Vec<(ChunkHash, ChunkProof)>,
        name: Option<&str>,
        output_dir: Option<&str>) -> Result<BatchProof> {
            if let Some(prover) = self.batch_prover.as_ref() {
                return prover.borrow_mut().gen_agg_evm_proof(chunk_hashes_proofs, name, output_dir)
            }
            unreachable!("please check errors in proof_type logic")
        }

    fn aggregator_check_chunk_proofs(&self, chunk_proofs: &[ChunkProof]) -> bool {
        if let Some(prover) = self.batch_prover.as_ref() {
            return prover.borrow_mut().check_chunk_proofs(chunk_proofs)
        }
        unreachable!("please check errors in proof_type logic")
    }
}