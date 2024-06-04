mod base;
mod next;
mod types;

use anyhow::Result;
use base::BaseCircuitsHandler;
use std::collections::HashMap;
use types::{BatchProof, BlockTrace, ChunkHash, ChunkProof};

use crate::{config::Config, types::ProofType};

type HardForkName = String;

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
    fn aggregator_check_chunk_proofs(&self, chunk_proofs: &[ChunkProof]) -> Result<bool>;
}

type CircuitsHandlerBuilder = fn(proof_type: ProofType, config: &Config) -> Result<Box<dyn CircuitsHandler>>;

pub struct CircuitsHandlerProvider<'a> {
    proof_type: ProofType,
    config: &'a Config,
    circuits_handler_builder_map: HashMap<HardForkName, CircuitsHandlerBuilder>,

    current_hard_fork_name: Option<HardForkName>,
    current_circuit: Option<Box<dyn CircuitsHandler>>,
    vks: Vec<String>,
}

impl<'a> CircuitsHandlerProvider<'a> {
    pub fn new(proof_type: ProofType, config: &'a Config) -> Result<Self> {
        let mut m: HashMap<HardForkName, CircuitsHandlerBuilder> = HashMap::new();

        fn handler_builder(proof_type: ProofType, config: &Config) -> Result<Box<dyn CircuitsHandler>> {
            BaseCircuitsHandler::new(proof_type,
                &config.low_version_circuit.params_path,
                &config.low_version_circuit.assets_path)
                .map(|handler| Box::new(handler) as Box<dyn CircuitsHandler>)
        }
        m.insert(config.low_version_circuit.hard_fork_name.clone(), handler_builder);

        fn next_handler_builder(proof_type: ProofType, config: &Config) -> Result<Box<dyn CircuitsHandler>> {
            BaseCircuitsHandler::new(proof_type,
                &config.high_version_circuit.params_path,
                &config.high_version_circuit.assets_path)
                .map(|handler| Box::new(handler) as Box<dyn CircuitsHandler>)
        }

        m.insert(config.high_version_circuit.hard_fork_name.clone(), next_handler_builder);

        let vks = CircuitsHandlerProvider::init_vks(proof_type, config, &m);

        let mut provider = CircuitsHandlerProvider {
            proof_type,
            config,
            circuits_handler_builder_map: m,
            current_hard_fork_name: None,
            current_circuit: None,
            vks,
        };

        // initialize current_circuit
        provider.get_circuits_handler(&config.low_version_circuit.hard_fork_name);
        Ok(provider)
    }

    pub fn get_circuits_handler(&mut self, hard_fork_name: &String) -> Option<&Box<dyn CircuitsHandler>> {
        match &self.current_hard_fork_name {
            Some(name) if name == hard_fork_name => {
                (&self.current_circuit).as_ref()
            },
            _ => {
                let builder = self.circuits_handler_builder_map.get(hard_fork_name);
                builder.and_then(|build| {
                    log::info!("building circuits handler for {hard_fork_name}");
                    let handler = build(self.proof_type, &self.config).expect("failed to build circuits handler");
                    self.current_hard_fork_name = Some(hard_fork_name.clone());
                    self.current_circuit = Some(handler);
                    (&self.current_circuit).as_ref()
                } )
            }
        }
    }

    fn init_vks(proof_type: ProofType, config: &'a Config, circuits_handler_builder_map: &HashMap<HardForkName, CircuitsHandlerBuilder>) -> Vec<String> {
        match proof_type {
            ProofType::ProofTypeBatch => circuits_handler_builder_map
                .values()
                .map(|build| {
                    let handler = build(proof_type, config).expect("failed to build circuits handler");
                    handler.aggregator_get_vk()
                        .map_or("".to_string(), |vk| utils::encode_vk(vk))
                })
                .collect::<Vec<String>>(),
            ProofType::ProofTypeChunk => circuits_handler_builder_map
                .values()
                .map(|build| {
                    let handler = build(proof_type, config).expect("failed to build circuits handler");
                    handler.prover_get_vk()
                        .map_or("".to_string(), |vk| utils::encode_vk(vk))
                })
                .collect::<Vec<String>>(),
            _ => unreachable!(),
        }
    }

    pub fn get_vks(&self) -> Vec<String> {
        self.vks.clone()
    }
}
