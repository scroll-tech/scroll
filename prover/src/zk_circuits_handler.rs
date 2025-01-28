mod common;
mod darwin;
mod darwin_v2;

use crate::{config::AssetsDirEnvConfig, prover::LocalProverConfig};
use anyhow::{bail, Result};
use async_trait::async_trait;
use darwin::DarwinHandler;
use darwin_v2::DarwinV2Handler;
use scroll_proving_sdk::prover::{proving_service::ProveRequest, ProofType};
use std::{collections::HashMap, sync::Arc};

type HardForkName = String;

pub mod utils {
    pub fn encode_vk(vk: Vec<u8>) -> String {
        base64::encode(vk)
    }
}

#[async_trait]
pub trait CircuitsHandler: Send + Sync {
    async fn get_vk(&self, task_type: ProofType) -> Option<Vec<u8>>;

    async fn get_proof_data(&self, prove_request: ProveRequest) -> Result<String>;
}

type CircuitsHandlerBuilder =
    fn(proof_types: Vec<ProofType>, config: &LocalProverConfig) -> Result<Box<dyn CircuitsHandler>>;

pub struct CircuitsHandlerProvider {
    config: LocalProverConfig,
    circuits_handler_builder_map: HashMap<HardForkName, CircuitsHandlerBuilder>,
    current_fork_name: Option<HardForkName>,
    current_circuit: Option<Arc<Box<dyn CircuitsHandler>>>,
}

impl CircuitsHandlerProvider {
    pub fn new(config: LocalProverConfig) -> Result<Self> {
        let mut m: HashMap<HardForkName, CircuitsHandlerBuilder> = HashMap::new();

        if let Err(e) = AssetsDirEnvConfig::init() {
            panic!("AssetsDirEnvConfig init failed: {:#}", e);
        }

        fn handler_builder(
            proof_types: Vec<ProofType>,
            config: &LocalProverConfig,
        ) -> Result<Box<dyn CircuitsHandler>> {
            log::info!(
                "now init zk circuits handler, hard_fork_name: {}",
                &config.low_version_circuit.hard_fork_name
            );
            AssetsDirEnvConfig::enable_first();
            DarwinHandler::new(
                proof_types,
                &config.low_version_circuit.params_path,
                &config.low_version_circuit.assets_path,
            )
            .map(|handler| Box::new(handler) as Box<dyn CircuitsHandler>)
        }
        m.insert(
            config.low_version_circuit.hard_fork_name.clone(),
            handler_builder,
        );

        fn next_handler_builder(
            proof_types: Vec<ProofType>,
            config: &LocalProverConfig,
        ) -> Result<Box<dyn CircuitsHandler>> {
            log::info!(
                "now init zk circuits handler, hard_fork_name: {}",
                &config.high_version_circuit.hard_fork_name
            );
            AssetsDirEnvConfig::enable_second();
            DarwinV2Handler::new(
                proof_types,
                &config.high_version_circuit.params_path,
                &config.high_version_circuit.assets_path,
            )
            .map(|handler| Box::new(handler) as Box<dyn CircuitsHandler>)
        }

        m.insert(
            config.high_version_circuit.hard_fork_name.clone(),
            next_handler_builder,
        );

        let provider = CircuitsHandlerProvider {
            config,
            circuits_handler_builder_map: m,
            current_fork_name: None,
            current_circuit: None,
        };

        Ok(provider)
    }

    pub fn get_circuits_handler(
        &mut self,
        hard_fork_name: &String,
    ) -> Result<Arc<Box<dyn CircuitsHandler>>> {
        match &self.current_fork_name {
            Some(fork_name) if fork_name == hard_fork_name => {
                log::info!("get circuits handler from cache");
                if let Some(handler) = &self.current_circuit {
                    Ok(handler.clone())
                } else {
                    bail!("missing cached handler, there must be something wrong.")
                }
            }
            _ => {
                log::info!(
                    "failed to get circuits handler from cache, create a new one: {hard_fork_name}"
                );
                if let Some(builder) = self.circuits_handler_builder_map.get(hard_fork_name) {
                    log::info!("building circuits handler for {hard_fork_name}");
                    let handler = builder(
                        self.config.sdk_config.prover.supported_proof_types.clone(),
                        &self.config,
                    )
                    .expect("failed to build circuits handler");
                    self.current_fork_name = Some(hard_fork_name.clone());
                    let arc_handler = Arc::new(handler);
                    self.current_circuit = Some(arc_handler.clone());
                    Ok(arc_handler)
                } else {
                    bail!("missing builder, there must be something wrong.")
                }
            }
        }
    }

    pub async fn init_vks(
        &self,
        config: &LocalProverConfig,
        proof_types: Vec<ProofType>,
    ) -> Vec<String> {
        let mut vks = Vec::new();
        for (hard_fork_name, build) in self.circuits_handler_builder_map.iter() {
            let handler =
                build(proof_types.clone(), config).expect("failed to build circuits handler");

            for prover_type in &proof_types {
                let vk = handler
                    .get_vk(*prover_type)
                    .await
                    .map_or("".to_string(), utils::encode_vk);
                log::info!(
                    "vk for {hard_fork_name}, is {vk}, prover_type: {:?}",
                    prover_type
                );
                if !vk.is_empty() {
                    vks.push(vk)
                }
            }
        }
        vks
    }
}
