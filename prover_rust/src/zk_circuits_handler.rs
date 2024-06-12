mod bernoulli;
mod curie;

use anyhow::{bail, Result};
use bernoulli::BaseCircuitsHandler;
use curie::NextCircuitsHandler;
use std::collections::HashMap;
use std::{cell::RefCell, rc::Rc};
use super::geth_client::GethClient;
use crate::{config::{Config, AssetsDirEnvConfig}, types::{ProofType, Task}};

type HardForkName = String;

pub mod utils {
    pub fn encode_vk(vk: Vec<u8>) -> String {
        base64::encode(vk)
    }
}

pub trait CircuitsHandler {
    fn get_vk(&self, task_type: ProofType) -> Option<Vec<u8>>;

    fn get_proof_data(&self, task_type: ProofType, task: &Task) -> Result<String>;
}

type CircuitsHandlerBuilder = fn(proof_type: ProofType, config: &Config, geth_client: Option<Rc<RefCell<GethClient>>>) -> Result<Box<dyn CircuitsHandler>>;

pub struct CircuitsHandlerProvider<'a> {
    proof_type: ProofType,
    config: &'a Config,
    geth_client: Option<Rc<RefCell<GethClient>>>,
    circuits_handler_builder_map: HashMap<HardForkName, CircuitsHandlerBuilder>,

    current_hard_fork_name: Option<HardForkName>,
    current_circuit: Option<Rc<Box<dyn CircuitsHandler>>>,
    vks: Vec<String>,
}

impl<'a> CircuitsHandlerProvider<'a> {
    pub fn new(proof_type: ProofType, config: &'a Config, geth_client: Option<Rc<RefCell<GethClient>>>) -> Result<Self> {
        let mut m: HashMap<HardForkName, CircuitsHandlerBuilder> = HashMap::new();

        fn handler_builder(proof_type: ProofType, config: &Config, geth_client: Option<Rc<RefCell<GethClient>>>) -> Result<Box<dyn CircuitsHandler>> {
            log::info!("now init zk circuits handler, hard_fork_name: {}", &config.low_version_circuit.hard_fork_name);
            AssetsDirEnvConfig::enable_first();
            BaseCircuitsHandler::new(proof_type,
                &config.low_version_circuit.params_path,
                &config.low_version_circuit.assets_path,
                geth_client
            ).map(|handler| Box::new(handler) as Box<dyn CircuitsHandler>)
        }
        m.insert(config.low_version_circuit.hard_fork_name.clone(), handler_builder);

        fn next_handler_builder(proof_type: ProofType, config: &Config, geth_client: Option<Rc<RefCell<GethClient>>>) -> Result<Box<dyn CircuitsHandler>> {
            log::info!("now init zk circuits handler, hard_fork_name: {}", &config.high_version_circuit.hard_fork_name);
            AssetsDirEnvConfig::enable_second();
            NextCircuitsHandler::new(proof_type,
                &config.high_version_circuit.params_path,
                &config.high_version_circuit.assets_path,
                geth_client
            ).map(|handler| Box::new(handler) as Box<dyn CircuitsHandler>)
        }

        m.insert(config.high_version_circuit.hard_fork_name.clone(), next_handler_builder);

        let vks = CircuitsHandlerProvider::init_vks(proof_type, config, &m, geth_client.clone());

        let provider = CircuitsHandlerProvider {
            proof_type,
            config,
            geth_client,
            circuits_handler_builder_map: m,
            current_hard_fork_name: None,
            current_circuit: None,
            vks,
        };

        Ok(provider)
    }

    pub fn get_circuits_handler(&mut self, hard_fork_name: &String) -> Result<Rc<Box<dyn CircuitsHandler>>> {
        match &self.current_hard_fork_name {
            Some(name) if name == hard_fork_name => {
                log::info!("get circuits handler from cache");
                if let Some(handler) = &self.current_circuit {
                    Ok(handler.clone())
                } else {
                    log::error!("missing cached handler, there must be something wrong.");
                    bail!("missing cached handler, there must be something wrong.")
                }
            },
            _ => {
                log::info!("failed to get circuits handler from cache, create a new one: {hard_fork_name}");
                if let Some(builder) = self.circuits_handler_builder_map.get(hard_fork_name) {
                    log::info!("building circuits handler for {hard_fork_name}");
                    let handler = builder(self.proof_type, self.config, self.geth_client.clone()).expect("failed to build circuits handler");
                    self.current_hard_fork_name = Some(hard_fork_name.clone());
                    let rc_handler = Rc::new(handler);
                    self.current_circuit = Some(rc_handler.clone());
                    Ok(rc_handler)
                } else {
                    log::error!("missing builder, there must be something wrong.");
                    bail!("missing builder, there must be something wrong.")
                }
            }
        }
    }

    fn init_vks(proof_type: ProofType, config: &'a Config,
        circuits_handler_builder_map: &HashMap<HardForkName, CircuitsHandlerBuilder>,
        geth_client: Option<Rc<RefCell<GethClient>>>) -> Vec<String> {
        circuits_handler_builder_map
                .iter()
                .map(|(hard_fork_name, build)| {
                    let handler = build(proof_type, config, geth_client.clone()).expect("failed to build circuits handler");
                    let vk = handler.get_vk(proof_type)
                        .map_or("".to_string(),  utils::encode_vk);
                    log::info!("vk for {hard_fork_name} is {vk}");
                    vk
                })
                .collect::<Vec<String>>()
    }

    pub fn get_vks(&self) -> Vec<String> {
        self.vks.clone()
    }
}
