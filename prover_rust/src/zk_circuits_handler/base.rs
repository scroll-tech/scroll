
use anyhow::{Ok, Result};
use crate::types::ProofType;
use super::CircuitsHandler;
use super::types::{ChunkProof, BatchProof, BlockTrace, ChunkHash};

use prover::zkevm::{Prover as ChunkProver, Verifier as ChunkVerifier};
use prover::aggregator::{Prover as BatchProver, Verifier as BaatchVerifier};

#[derive(Default)]
pub struct BaseCircuitsHandler {
    chunk_prover: Option<ChunkProver>,
    chunk_verifier: Option<ChunkVerifier>,

    batch_prover: Option<BatchProver>,
    batch_verifier: Option<BaatchVerifier>,
}

impl BaseCircuitsHandler {
    pub fn new(proof_type: ProofType, params_dir: &str, assets_dir: &str) -> Result<Self> {
        match proof_type {
            ProofType::ProofTypeChunk => Ok(Self {
                chunk_prover: Some(ChunkProver::from_dirs(params_dir, assets_dir)),
                chunk_verifier: Some(ChunkVerifier::from_dirs(params_dir, assets_dir)),
                ..Default::default()
            }),

            ProofType::ProofTypeBatch => Ok(Self {
                batch_prover: Some(BatchProver::from_dirs(params_dir, assets_dir)),
                batch_verifier: Some(BaatchVerifier::from_dirs(params_dir, assets_dir)),
                ..Default::default()
            }),
            // TODO: add custom error system and change unreachable to error
            _ => unreachable!()
        }
    }
}

impl CircuitsHandler for BaseCircuitsHandler {
    // api of zkevm::Prover
    fn prover_get_vk(&self) -> Option<Vec<u8>> {
        self.chunk_prover.and_then(|prover| prover.get_vk())
    }

    fn prover_gen_chunk_proof(&self,
        chunk_trace: Vec<BlockTrace>,
        name: Option<&str>,
        inner_id: Option<&str>,
        output_dir: Option<&str>) -> Result<ChunkProof> {
            if let Some(mut prover) = self.chunk_prover {
                return prover.gen_chunk_proof(chunk_trace, name, inner_id, output_dir)
            }
            // TODO: add custom error system and change unreachable to error
            unreachable!()
        }

    // api of aggregator::Prover
    fn aggregator_get_vk(&self) -> Option<Vec<u8>> {
        self.batch_prover.and_then(|prover| prover.get_vk())
    }

    fn aggregator_gen_agg_evm_proof(&self,
        chunk_hashes_proofs: Vec<(ChunkHash, ChunkProof)>,
        name: Option<&str>,
        output_dir: Option<&str>) -> Result<BatchProof> {
            if let Some(mut prover) = self.batch_prover {
                return prover.gen_agg_evm_proof(chunk_hashes_proofs, name, output_dir)
            }
            // TODO: add custom error system and change unreachable to error
            unreachable!()
        }

    fn aggregator_check_chunk_proofs(&self, chunk_proofs: &[ChunkProof]) -> bool {
        if let Some(prover) = self.batch_prover {
            return prover.check_chunk_proofs(chunk_proofs)
        }
        // TODO: add custom error system and change unreachable to error
        unreachable!()
    }
}