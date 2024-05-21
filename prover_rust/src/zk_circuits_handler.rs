mod base;
mod types;

use anyhow::Result;
use base::BaseCircuitsHandler;
use std::collections::HashMap;
use types::{BatchProof, BlockTrace, ChunkHash, ChunkProof};

use crate::types::ProofType;

type CiruitsVersion = String;

pub mod utils {
    pub fn encode_vk(vk: Vec<u8>) -> String {
        base64::encode(vk)
    }
}

pub trait CircuitsHandler {
    // api of zkevm::Prover
    fn prover_get_vk(&self) -> Option<Vec<u8>>;
    fn prover_gen_chunk_proof(
        &self,
        chunk_trace: Vec<BlockTrace>,
        name: Option<&str>,
        inner_id: Option<&str>,
        output_dir: Option<&str>,
    ) -> Result<ChunkProof>;

    // api of aggregator::Prover
    fn aggregator_get_vk(&self) -> Option<Vec<u8>>;
    fn aggregator_gen_agg_evm_proof(
        &self,
        chunk_hashes_proofs: Vec<(ChunkHash, ChunkProof)>,
        name: Option<&str>,
        output_dir: Option<&str>,
    ) -> Result<BatchProof>;
    fn aggregator_check_chunk_proofs(&self, chunk_proofs: &[ChunkProof]) -> bool;
}

pub struct CircuitsHandlerProvider {
    proof_type: ProofType,
    circuits_handler_map: HashMap<CiruitsVersion, Box<dyn CircuitsHandler>>,
}

impl CircuitsHandlerProvider {
    pub fn new(proof_type: ProofType, params_dir: &str, assets_dir: &str) -> Result<Self> {
        let mut m: HashMap<CiruitsVersion, Box<dyn CircuitsHandler>> = HashMap::new();
        let handler = BaseCircuitsHandler::new(proof_type, params_dir, assets_dir)?;
        m.insert("".to_string(), Box::new(handler));

        Ok(CircuitsHandlerProvider {
            proof_type: proof_type,
            circuits_handler_map: m,
        })
    }

    pub fn get_circuits_client(&self, version: String) -> Option<&Box<dyn CircuitsHandler>> {
        self.circuits_handler_map.get(&version)
    }

    pub fn get_vks(&self) -> Vec<String> {
        match self.proof_type {
            ProofType::ProofTypeBatch => self
                .circuits_handler_map
                .values()
                .map(|h| {
                    h.aggregator_get_vk()
                        .map_or("".to_string(), |vk| utils::encode_vk(vk))
                })
                .collect::<Vec<String>>(),
            ProofType::ProofTypeChunk => self
                .circuits_handler_map
                .values()
                .map(|h| {
                    h.prover_get_vk()
                        .map_or("".to_string(), |vk| utils::encode_vk(vk))
                })
                .collect::<Vec<String>>(),
            _ => unreachable!(),
        }
    }
}
